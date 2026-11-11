package booking

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/suwahadi/sewakos/backend/internal/finance"
	"github.com/suwahadi/sewakos/backend/internal/midtrans"
	"github.com/suwahadi/sewakos/backend/internal/money"
	"github.com/suwahadi/sewakos/backend/internal/notification"
	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
	"github.com/suwahadi/sewakos/backend/internal/response"
)

type notificationBody struct {
	OrderID      string `json:"order_id"`
	StatusCode   string `json:"status_code"`
	GrossAmount  string `json:"gross_amount"`
	SignatureKey string `json:"signature_key"`
}

var errPaymentNotFound = errors.New("payment not found")

func WebhookHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		rawBody, err := io.ReadAll(r.Body)
		if err != nil {
			response.Fail(w, http.StatusBadRequest, response.MsgValidationFailed, nil)
			return
		}
		var body notificationBody
		if err := json.Unmarshal(rawBody, &body); err != nil {
			response.Fail(w, http.StatusBadRequest, response.MsgValidationFailed, nil)
			return
		}
		if !midtrans.VerifyNotificationSignature(deps.MidtransServerKey, body.OrderID, body.StatusCode, body.GrossAmount, body.SignatureKey) {
			response.Fail(w, http.StatusForbidden, response.MsgAccessDenied, nil)
			return
		}

		if _, err := getPaymentByOrderID(ctx, deps, body.OrderID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				response.NotFoundHandler(w, r)
				return
			}
			response.InternalErrorHandler(w, r)
			return
		}

		status, err := deps.Midtrans.GetStatus(ctx, body.OrderID)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}

		authoritativeAmount, ok := ParseMidtransGrossAmount(status.GrossAmount)
		if !ok {
			response.InternalErrorHandler(w, r)
			return
		}
		payment, err := getPaymentByOrderID(ctx, deps, body.OrderID)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if authoritativeAmount != payment.GrossAmount {
			slog.Error("midtrans notification gross_amount mismatch",
				"order_id", body.OrderID, "expected", payment.GrossAmount, "got", authoritativeAmount)
			if err := finalizePaymentAsFailure(ctx, deps, body.OrderID, rawBody); err != nil {
				response.InternalErrorHandler(w, r)
				return
			}
			response.Success(w, http.StatusOK, "OK", nil)
			return
		}

		if err := ApplyPaymentStatus(ctx, deps, body.OrderID, status, rawBody); err != nil {
			if errors.Is(err, errPaymentNotFound) {
				response.NotFoundHandler(w, r)
				return
			}
			response.InternalErrorHandler(w, r)
			return
		}
		response.Success(w, http.StatusOK, "OK", nil)
	}
}

func getPaymentByOrderID(ctx context.Context, deps Deps, orderID string) (dbgen.GetPaymentByOrderIDRow, error) {
	tx, err := deps.DB.Begin(ctx)
	if err != nil {
		return dbgen.GetPaymentByOrderIDRow{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	return dbgen.New(tx).GetPaymentByOrderID(ctx, orderID)
}

func ParseMidtransGrossAmount(s string) (int64, bool) {
	whole, _, found := strings.Cut(s, ".")
	if !found {
		whole = s
	}
	amount, err := strconv.ParseInt(whole, 10, 64)
	return amount, err == nil
}

func finalizePaymentAsFailure(ctx context.Context, deps Deps, orderID string, rawBody []byte) error {
	tx, err := deps.DB.Begin(ctx)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()
	queries := dbgen.New(tx)

	row, err := queries.GetPaymentForWebhook(ctx, orderID)
	if errors.Is(err, pgx.ErrNoRows) {
		return errPaymentNotFound
	}
	if err != nil {
		return err
	}
	if row.PaymentStatus != "pending" {
		committed = true
		return tx.Commit(ctx)
	}

	if err := queries.FinalizePayment(ctx, dbgen.FinalizePaymentParams{
		ID: row.PaymentID, Status: "failure", LastNotification: rawBody,
	}); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	committed = true
	return nil
}

// ApplyPaymentStatus is shared by the webhook and the expire_payments worker
// job. It always re-reads the payment under a lock on its booking row, so
// concurrent and out-of-order calls for the same order_id serialize, and any
// call after the payment is already final is a no-op.
func ApplyPaymentStatus(ctx context.Context, deps Deps, orderID string, status midtrans.StatusResponse, rawBody []byte) error {
	tx, err := deps.DB.Begin(ctx)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()
	queries := dbgen.New(tx)

	row, err := queries.GetPaymentForWebhook(ctx, orderID)
	if errors.Is(err, pgx.ErrNoRows) {
		return errPaymentNotFound
	}
	if err != nil {
		return err
	}
	if row.PaymentStatus != "pending" {
		committed = true
		return tx.Commit(ctx)
	}

	success := status.TransactionStatus == "settlement" ||
		(status.TransactionStatus == "capture" && status.FraudStatus == "accept")
	failed := status.TransactionStatus == "expire" || status.TransactionStatus == "cancel" || status.TransactionStatus == "deny"

	if !success && !failed {
		committed = true
		return tx.Commit(ctx)
	}

	if failed {
		if err := queries.FinalizePayment(ctx, dbgen.FinalizePaymentParams{
			ID: row.PaymentID, Status: status.TransactionStatus,
			PaymentType: pgtextFrom(status.PaymentType), MidtransTransactionID: pgtextFrom(status.TransactionID),
			LastNotification: rawBody,
		}); err != nil {
			return err
		}
		if row.InvoiceType == "initial" && row.BookingStatus == "pending" && row.StockHeld {
			if err := queries.ReleaseRoomLock(ctx, row.RoomID); err != nil {
				return err
			}
			if err := queries.SetBookingStockHeld(ctx, dbgen.SetBookingStockHeldParams{ID: row.BookingID, StockHeld: false}); err != nil {
				return err
			}
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
		committed = true
		return nil
	}

	paidAt := time.Now()
	if err := queries.FinalizePayment(ctx, dbgen.FinalizePaymentParams{
		ID: row.PaymentID, Status: "settlement",
		PaymentType: pgtextFrom(status.PaymentType), MidtransTransactionID: pgtextFrom(status.TransactionID),
		PaidAt: toPgTimestamptz(paidAt), LastNotification: rawBody,
	}); err != nil {
		return err
	}
	if _, err := finance.Record(ctx, queries, finance.Entry{
		Type: finance.TypePaymentReceived, Amount: row.InvoiceTotal, OwnerID: &row.OwnerID, TenantID: &row.TenantID,
		BookingID: &row.BookingID, InvoiceID: &row.InvoiceID, PaymentID: &row.PaymentID,
		Description: fmt.Sprintf("Payment received for invoice %s", row.InvoiceNumber),
	}); err != nil {
		return err
	}
	if row.InvoiceAdminFee > 0 {
		if _, err := finance.Record(ctx, queries, finance.Entry{
			Type: finance.TypeAdminFeeEarned, Amount: row.InvoiceAdminFee, OwnerID: &row.OwnerID, TenantID: &row.TenantID,
			BookingID: &row.BookingID, InvoiceID: &row.InvoiceID, PaymentID: &row.PaymentID,
			Description: fmt.Sprintf("Admin fee for invoice %s", row.InvoiceNumber),
		}); err != nil {
			return err
		}
	}

	if err := applyPaymentSuccess(ctx, deps, queries, row); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}
	committed = true
	return nil
}

func applyPaymentSuccess(ctx context.Context, deps Deps, queries *dbgen.Queries, row dbgen.GetPaymentForWebhookRow) error {
	switch {
	case row.InvoiceStatus == "void":
		return refundPayment(ctx, queries, row, "invoice_void")
	case row.BookingStatus == "cancelled":
		return refundPayment(ctx, queries, row, "tenant_cancelled")
	case row.InvoiceType == "initial" && row.BookingStatus == "pending":
		return settleInitialPayment(ctx, deps, queries, row)
	case row.InvoiceType == "renewal" && row.InvoiceStatus == "unpaid":
		return settleRenewalOrExtension(ctx, deps, queries, row, false)
	case row.InvoiceType == "extension" && row.InvoiceStatus == "unpaid":
		return settleRenewalOrExtension(ctx, deps, queries, row, true)
	default:
		return nil
	}
}

func refundPayment(ctx context.Context, queries *dbgen.Queries, row dbgen.GetPaymentForWebhookRow, reason string) error {
	refundID, err := queries.InsertRefund(ctx, dbgen.InsertRefundParams{
		BookingID: row.BookingID, InvoiceID: row.InvoiceID, Amount: row.InvoiceTotal, Reason: reason,
	})
	if err != nil {
		return err
	}
	_, err = finance.Record(ctx, queries, finance.Entry{
		Type: finance.TypeRefundDue, Amount: row.InvoiceTotal, TenantID: &row.TenantID,
		BookingID: &row.BookingID, InvoiceID: &row.InvoiceID, RefundID: &refundID,
		Description: fmt.Sprintf("Refund due for invoice %s (%s)", row.InvoiceNumber, reason),
	})
	return err
}

func settleInitialPayment(ctx context.Context, deps Deps, queries *dbgen.Queries, row dbgen.GetPaymentForWebhookRow) error {
	if !row.StockHeld {
		rows, err := queries.DecrementRoomAvailable(ctx, row.RoomID)
		if err != nil {
			return err
		}
		if rows == 0 {
			if err := queries.RejectBookingForStockUnavailable(ctx, dbgen.RejectBookingForStockUnavailableParams{
				ID: row.BookingID, RejectReason: pgtextFrom("Kamar tidak tersedia saat pembayaran diterima."),
			}); err != nil {
				return err
			}
			if err := refundPayment(ctx, queries, row, "stock_unavailable"); err != nil {
				return err
			}
			return notifyTenant(ctx, deps, queries, row, notification.CategoryBooking, "Booking dibatalkan",
				"Kamar sudah tidak tersedia saat pembayaran diterima. Dana akan dikembalikan penuh. Silakan isi rekening pengembalian dana.",
				fmt.Sprintf("/booking/%d", row.BookingID))
		}
		if err := queries.SetBookingStockHeld(ctx, dbgen.SetBookingStockHeldParams{ID: row.BookingID, StockHeld: true}); err != nil {
			return err
		}
	}

	paidAt := time.Now()
	ownerDeadline := paidAt.Add(24 * time.Hour)
	if err := queries.MarkBookingPaid(ctx, dbgen.MarkBookingPaidParams{
		ID: row.BookingID, PaidAt: toPgTimestamptz(paidAt), OwnerDeadlineAt: toPgTimestamptz(ownerDeadline),
	}); err != nil {
		return err
	}

	if err := notifyTenant(ctx, deps, queries, row, notification.CategoryPayment, "Pembayaran berhasil",
		fmt.Sprintf("Pembayaran untuk %s di %s sudah kami terima. Menunggu persetujuan pemilik.", row.RoomName, row.PropertyName),
		fmt.Sprintf("/booking/%d", row.BookingID)); err != nil {
		return err
	}

	tenant, err := queries.GetUserByID(ctx, row.TenantID)
	if err != nil {
		return err
	}
	return notifyOwner(ctx, deps, queries, row, notification.CategoryBooking, "Booking baru",
		fmt.Sprintf("%s sudah membayar booking %s di %s. Mohon respons sebelum %s.",
			tenant.Name, row.RoomName, row.PropertyName, money.FormatHumanWIB(ownerDeadline)),
		fmt.Sprintf("/owner/booking/%d", row.BookingID))
}

func settleRenewalOrExtension(ctx context.Context, deps Deps, queries *dbgen.Queries, row dbgen.GetPaymentForWebhookRow, extension bool) error {
	paidAt := time.Now()
	if err := queries.MarkInvoicePaid(ctx, dbgen.MarkInvoicePaidParams{ID: row.InvoiceID, PaidAt: toPgTimestamptz(paidAt)}); err != nil {
		return err
	}
	if extension {
		if err := queries.ExtendBookingDuration(ctx, dbgen.ExtendBookingDurationParams{
			ID: row.BookingID, Periods: row.InvoicePeriods, EndDate: row.InvoicePeriodEnd,
		}); err != nil {
			return err
		}
	}

	payoutAmount := row.InvoiceTotal - row.InvoiceAdminFee
	payoutID, err := queries.InsertPayout(ctx, dbgen.InsertPayoutParams{
		BookingID: row.BookingID, InvoiceID: row.InvoiceID, OwnerID: row.OwnerID, Amount: payoutAmount,
	})
	if err != nil {
		return err
	}
	if _, err := finance.Record(ctx, queries, finance.Entry{
		Type: finance.TypeOwnerEarning, Amount: payoutAmount, OwnerID: &row.OwnerID,
		BookingID: &row.BookingID, InvoiceID: &row.InvoiceID, PayoutID: &payoutID,
		Description: fmt.Sprintf("Owner earning for invoice %s", row.InvoiceNumber),
	}); err != nil {
		return err
	}

	if err := notifyTenant(ctx, deps, queries, row, notification.CategoryPayment, "Pembayaran berhasil",
		fmt.Sprintf("Pembayaran tagihan %s sudah kami terima.", row.InvoiceNumber),
		fmt.Sprintf("/booking/%d", row.BookingID)); err != nil {
		return err
	}

	tenant, err := queries.GetUserByID(ctx, row.TenantID)
	if err != nil {
		return err
	}
	return notifyOwner(ctx, deps, queries, row, notification.CategoryPayment, "Tagihan dibayar",
		fmt.Sprintf("%s membayar tagihan %s untuk %s di %s.", tenant.Name, row.InvoiceNumber, row.RoomName, row.PropertyName),
		fmt.Sprintf("/owner/booking/%d", row.BookingID))
}

func notifyTenant(ctx context.Context, deps Deps, queries *dbgen.Queries, row dbgen.GetPaymentForWebhookRow, category, title, body, link string) error {
	tenant, err := queries.GetUserByID(ctx, row.TenantID)
	if err != nil {
		return err
	}
	_, err = notification.Create(ctx, queries, deps.AppPublicURL, notification.Params{
		UserID: row.TenantID, RecipientEmail: textFieldOrNil(tenant.Email), Category: category,
		Title: title, Body: body, Link: &link,
	})
	return err
}

func notifyOwner(ctx context.Context, deps Deps, queries *dbgen.Queries, row dbgen.GetPaymentForWebhookRow, category, title, body, link string) error {
	owner, err := queries.GetUserByID(ctx, row.OwnerID)
	if err != nil {
		return err
	}
	_, err = notification.Create(ctx, queries, deps.AppPublicURL, notification.Params{
		UserID: row.OwnerID, RecipientEmail: textFieldOrNil(owner.Email), Category: category,
		Title: title, Body: body, Link: &link,
	})
	return err
}

func textFieldOrNil(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	return &t.String
}
