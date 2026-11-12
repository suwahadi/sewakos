package booking

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/suwahadi/sewakos/backend/internal/authctx"
	"github.com/suwahadi/sewakos/backend/internal/midtrans"
	"github.com/suwahadi/sewakos/backend/internal/money"
	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
	"github.com/suwahadi/sewakos/backend/internal/response"
)

const msgInvoiceCannotBePaid = "Tagihan tidak dapat dibayar."

func InvoiceCheckoutHandler(deps Deps, user authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		invoiceID, ok := parseInvoiceID(r)
		if !ok {
			response.NotFoundHandler(w, r)
			return
		}
		ctx := r.Context()

		row, err := peekInvoiceForCheckout(ctx, deps, invoiceID)
		if errors.Is(err, pgx.ErrNoRows) {
			response.NotFoundHandler(w, r)
			return
		}
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if row.TenantID != user.UserID {
			response.Fail(w, http.StatusForbidden, response.MsgAccessDenied, nil)
			return
		}

		if row.Type == "initial" {
			checkoutInitialBooking(deps, user, w, r, row.BookingID)
			return
		}

		if row.Status != "unpaid" || row.BookingStatus != "approved" {
			response.Fail(w, http.StatusBadRequest, msgInvoiceCannotBePaid, nil)
			return
		}
		checkoutRenewalOrExtension(deps, user, w, r, row)
	}
}

func peekInvoiceForCheckout(ctx context.Context, deps Deps, invoiceID int64) (dbgen.SelectInvoiceForCheckoutRow, error) {
	tx, err := deps.DB.Begin(ctx)
	if err != nil {
		return dbgen.SelectInvoiceForCheckoutRow{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	return dbgen.New(tx).SelectInvoiceForCheckout(ctx, invoiceID)
}

func checkoutRenewalOrExtension(deps Deps, user authctx.User, w http.ResponseWriter, r *http.Request, preRead dbgen.SelectInvoiceForCheckoutRow) {
	ctx := r.Context()

	tx, err := deps.DB.Begin(ctx)
	if err != nil {
		response.InternalErrorHandler(w, r)
		return
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()
	queries := dbgen.New(tx)

	row, err := queries.SelectInvoiceForCheckout(ctx, preRead.ID)
	if err != nil {
		response.InternalErrorHandler(w, r)
		return
	}
	if row.Status != "unpaid" || row.BookingStatus != "approved" {
		response.Fail(w, http.StatusBadRequest, msgInvoiceCannotBePaid, nil)
		return
	}

	if _, err := queries.ExpireStalePendingPaymentsForInvoice(ctx, row.ID); err != nil {
		response.InternalErrorHandler(w, r)
		return
	}

	active, err := queries.GetActivePaymentForInvoice(ctx, row.ID)
	if err == nil {
		if err := tx.Commit(ctx); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		committed = true
		response.Success(w, http.StatusOK, "Silakan selesaikan pembayaran.", map[string]any{
			"payment": checkoutPaymentBody(active.OrderID, active.GrossAmount, active.SnapToken.String, active.RedirectUrl.String, active.ExpiresAt.Time),
		})
		return
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		response.InternalErrorHandler(w, r)
		return
	}

	grossAmount := row.Total
	expiresAt := time.Now().Add(time.Duration(deps.PaymentExpiryMinutes) * time.Minute)
	paymentID, err := queries.InsertPendingPayment(ctx, dbgen.InsertPendingPaymentParams{
		InvoiceID: row.ID, GrossAmount: grossAmount, ExpiresAt: toPgTimestamptz(expiresAt),
	})
	if err != nil {
		response.InternalErrorHandler(w, r)
		return
	}
	orderID := fmt.Sprintf("KOS-%d-%d", row.BookingID, paymentID)
	if err := queries.SetPaymentOrderID(ctx, dbgen.SetPaymentOrderIDParams{ID: paymentID, OrderID: orderID}); err != nil {
		response.InternalErrorHandler(w, r)
		return
	}

	tenant, err := queries.GetUserByID(ctx, user.UserID)
	if err != nil {
		response.InternalErrorHandler(w, r)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		response.InternalErrorHandler(w, r)
		return
	}
	committed = true

	rentAmount := grossAmount - row.AdminFee
	items := []midtrans.ItemDetail{
		{ID: "rent", Name: truncateItemName(fmt.Sprintf("Sewa %s (%s)", row.RoomName, rentTypeLabel(row.RentType))), Price: rentAmount, Quantity: 1},
	}
	if row.AdminFee > 0 {
		items = append(items, midtrans.ItemDetail{ID: "admin_fee", Name: "Biaya Admin", Price: row.AdminFee, Quantity: 1})
	}

	req := midtrans.TransactionRequest{
		TransactionDetails: midtrans.TransactionDetails{OrderID: orderID, GrossAmount: grossAmount},
		ItemDetails:        items,
		CustomerDetails:    midtrans.CustomerDetails{FirstName: tenant.Name, Phone: tenant.Phone},
		Expiry: &midtrans.Expiry{
			StartTime: time.Now().In(money.WIB).Format("2006-01-02 15:04:05 -0700"),
			Unit:      "minutes", Duration: deps.PaymentExpiryMinutes,
		},
	}
	if deps.MidtransFinishURL != "" {
		req.Callbacks = &midtrans.Callbacks{Finish: deps.MidtransFinishURL}
	}

	snap, err := deps.Midtrans.CreateTransaction(ctx, req)
	if err != nil {
		failTx, beginErr := deps.DB.Begin(ctx)
		if beginErr == nil {
			_ = dbgen.New(failTx).MarkPaymentFailure(ctx, paymentID)
			_ = failTx.Commit(ctx)
		}
		response.Fail(w, http.StatusBadGateway, response.MsgPaymentGatewayError, nil)
		return
	}

	saveTx, err := deps.DB.Begin(ctx)
	if err != nil {
		response.InternalErrorHandler(w, r)
		return
	}
	defer func() { _ = saveTx.Rollback(ctx) }()
	if err := dbgen.New(saveTx).SetPaymentSnapDetails(ctx, dbgen.SetPaymentSnapDetailsParams{
		ID: paymentID, SnapToken: pgtextFrom(snap.Token), RedirectUrl: pgtextFrom(snap.RedirectURL),
	}); err != nil {
		response.InternalErrorHandler(w, r)
		return
	}
	if err := saveTx.Commit(ctx); err != nil {
		response.InternalErrorHandler(w, r)
		return
	}

	response.Success(w, http.StatusOK, "Silakan selesaikan pembayaran.", map[string]any{
		"payment": checkoutPaymentBody(orderID, grossAmount, snap.Token, snap.RedirectURL, expiresAt),
	})
}
