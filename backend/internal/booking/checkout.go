package booking

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/suwahadi/sewakos/backend/internal/authctx"
	"github.com/suwahadi/sewakos/backend/internal/midtrans"
	"github.com/suwahadi/sewakos/backend/internal/money"
	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
	"github.com/suwahadi/sewakos/backend/internal/response"
)

func pgtextFrom(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: s != ""}
}

func rentTypeLabel(rentType string) string {
	switch rentType {
	case "daily":
		return "Per Hari"
	case "weekly":
		return "Per Minggu"
	default:
		return "Per Bulan"
	}
}

func truncateItemName(name string) string {
	runes := []rune(name)
	if len(runes) > 50 {
		return string(runes[:50])
	}
	return name
}

func checkoutPaymentBody(orderID string, grossAmount int64, snapToken, redirectURL string, expiresAt time.Time) map[string]any {
	return map[string]any{
		"order_id": orderID, "gross_amount": grossAmount, "gross_amount_label": money.FormatRupiah(grossAmount),
		"snap_token": snapToken, "redirect_url": redirectURL, "expires_at": money.FormatExpiresAt(expiresAt),
	}
}

func CheckoutHandler(deps Deps, user authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bookingID, ok := parseBookingID(r)
		if !ok {
			response.NotFoundHandler(w, r)
			return
		}
		checkoutInitialBooking(deps, user, w, r, bookingID)
	}
}

func checkoutInitialBooking(deps Deps, user authctx.User, w http.ResponseWriter, r *http.Request, bookingID int64) {
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

	b, err := queries.SelectBookingForCheckout(ctx, bookingID)
	if errors.Is(err, pgx.ErrNoRows) {
		response.NotFoundHandler(w, r)
		return
	}
	if err != nil {
		response.InternalErrorHandler(w, r)
		return
	}
	if b.TenantID != user.UserID {
		response.Fail(w, http.StatusForbidden, response.MsgAccessDenied, nil)
		return
	}
	if b.Status != "pending" {
		response.Fail(w, http.StatusBadRequest, "Booking tidak dapat dibayar.", nil)
		return
	}
	if !b.ExpiresAt.Time.After(time.Now()) {
		response.Fail(w, http.StatusBadRequest, "Batas waktu pembayaran sudah lewat.", nil)
		return
	}
	listed, err := queries.IsPropertyListedForRoom(ctx, b.RoomID)
	if err != nil {
		response.InternalErrorHandler(w, r)
		return
	}
	if !listed {
		response.Fail(w, http.StatusBadRequest, "Properti tidak aktif.", nil)
		return
	}

	invoice, err := queries.GetInitialInvoiceForBooking(ctx, bookingID)
	if err != nil {
		response.InternalErrorHandler(w, r)
		return
	}

	active, err := queries.GetActivePaymentForInvoice(ctx, invoice.ID)
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

	if !b.StockHeld {
		rows, err := queries.DecrementRoomAvailable(ctx, b.RoomID)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if rows == 0 {
			response.Fail(w, http.StatusBadRequest, "Kamar tidak tersedia.", nil)
			return
		}
		if err := queries.SetBookingStockHeld(ctx, dbgen.SetBookingStockHeldParams{ID: bookingID, StockHeld: true}); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
	}

	grossAmount := b.Price + b.Deposit + b.AdminFee
	expiresAt := time.Now().Add(time.Duration(deps.PaymentExpiryMinutes) * time.Minute)
	paymentID, err := queries.InsertPendingPayment(ctx, dbgen.InsertPendingPaymentParams{
		InvoiceID: invoice.ID, GrossAmount: grossAmount, ExpiresAt: toPgTimestamptz(expiresAt),
	})
	if err != nil {
		response.InternalErrorHandler(w, r)
		return
	}
	orderID := fmt.Sprintf("KOS-%d-%d", bookingID, paymentID)
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

	items := []midtrans.ItemDetail{
		{ID: "rent", Name: truncateItemName(fmt.Sprintf("Sewa %s (%s)", b.RoomName, rentTypeLabel(b.RentType))), Price: b.Price, Quantity: 1},
		{ID: "deposit", Name: "Deposit", Price: b.Deposit, Quantity: 1},
	}
	if b.AdminFee > 0 {
		items = append(items, midtrans.ItemDetail{ID: "admin_fee", Name: "Biaya Admin", Price: b.AdminFee, Quantity: 1})
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
			failQueries := dbgen.New(failTx)
			_ = failQueries.MarkPaymentFailure(ctx, paymentID)
			_ = failQueries.SetBookingStockHeld(ctx, dbgen.SetBookingStockHeldParams{ID: bookingID, StockHeld: false})
			_ = failQueries.ReleaseRoomLock(ctx, b.RoomID)
			_ = failTx.Commit(ctx)
		}
		response.Fail(w, http.StatusBadGateway, "Layanan pembayaran sedang bermasalah. Silakan coba lagi.", nil)
		return
	}

	saveTx, err := deps.DB.Begin(ctx)
	if err != nil {
		response.InternalErrorHandler(w, r)
		return
	}
	defer func() { _ = saveTx.Rollback(ctx) }()
	saveQueries := dbgen.New(saveTx)
	if err := saveQueries.SetPaymentSnapDetails(ctx, dbgen.SetPaymentSnapDetailsParams{
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
