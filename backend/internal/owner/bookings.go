package owner

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/suwahadi/sewakos/backend/internal/authctx"
	"github.com/suwahadi/sewakos/backend/internal/finance"
	"github.com/suwahadi/sewakos/backend/internal/money"
	"github.com/suwahadi/sewakos/backend/internal/notification"
	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
	"github.com/suwahadi/sewakos/backend/internal/response"
	"github.com/suwahadi/sewakos/backend/internal/validation"
)

const msgBookingNotProcessable = "Booking tidak dapat diproses."

func nullableTimestamp(t pgtype.Timestamptz) any {
	if !t.Valid {
		return nil
	}
	return money.FormatExpiresAt(t.Time)
}

func optionalQueryID(r *http.Request, key string) pgtype.Int8 {
	n, err := strconv.ParseInt(r.URL.Query().Get(key), 10, 64)
	return pgtype.Int8{Int64: n, Valid: err == nil}
}

func loadOwnerBookings(ctx context.Context, store dbgen.Querier, params dbgen.ListOwnerBookingsParams) ([]map[string]any, error) {
	rows, err := store.ListOwnerBookings(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("list owner bookings: %w", err)
	}
	bookings := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		var tenantPhone any
		if row.Status == "approved" {
			tenantPhone = row.TenantPhone
		}
		bookings = append(bookings, map[string]any{
			"id": row.ID, "status": row.Status, "rent_type": row.RentType,
			"start_date": row.StartDate.Time.Format(time.DateOnly), "duration": row.Duration,
			"end_date": row.EndDate.Time.Format(time.DateOnly),
			"tenant":   map[string]any{"id": row.TenantID, "name": row.TenantName, "phone_number": tenantPhone},
			"property": map[string]any{"id": row.PropertyID, "name": row.PropertyName},
			"room":     map[string]any{"id": row.RoomID, "name": row.RoomName},
			"price":    row.Price, "price_label": money.FormatRupiah(row.Price),
			"deposit": row.Deposit, "deposit_label": money.FormatRupiah(row.Deposit),
			"payout_amount": row.PayoutAmount, "payout_amount_label": money.FormatRupiah(row.PayoutAmount),
			"paid_at": nullableTimestamp(row.PaidAt), "owner_deadline_at": nullableTimestamp(row.OwnerDeadlineAt),
			"reject_reason": nullableText(row.RejectReason), "deposit_returned_at": nullableTimestamp(row.DepositReturnedAt),
			"created_at": money.FormatTimestamp(row.CreatedAt.Time),
		})
	}
	return bookings, nil
}

func respondWithBooking(w http.ResponseWriter, r *http.Request, deps Deps, ownerID, bookingID int64, message string) {
	bookings, err := loadOwnerBookings(r.Context(), deps.Store, dbgen.ListOwnerBookingsParams{
		OwnerID: ownerID, BookingID: pgtype.Int8{Int64: bookingID, Valid: true}, RowLimit: 1,
	})
	if err != nil {
		response.InternalErrorHandler(w, r)
		return
	}
	if len(bookings) == 0 {
		response.NotFoundHandler(w, r)
		return
	}
	response.Success(w, http.StatusOK, message, map[string]any{"booking": bookings[0]})
}

func NotifyUser(ctx context.Context, queries *dbgen.Queries, appPublicURL string, userID int64, category, title, body, link string) error {
	user, err := queries.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("load user %d for notification: %w", userID, err)
	}
	var email *string
	if user.Email.Valid {
		email = &user.Email.String
	}
	_, err = notification.Create(ctx, queries, appPublicURL, notification.Params{
		UserID: userID, RecipientEmail: email, Category: category, Title: title, Body: body, Link: &link,
	})
	return err
}

type Rejection struct {
	RejectedBy   string
	Reason       pgtype.Text
	RefundReason string
	TenantTitle  string
	TenantBody   string
}

func RejectPaidBooking(ctx context.Context, queries *dbgen.Queries, appPublicURL string, booking dbgen.LockPaidBookingRow, rejection Rejection) error {
	if err := queries.RejectPaidBooking(ctx, dbgen.RejectPaidBookingParams{
		ID: booking.ID, RejectedBy: pgtype.Text{String: rejection.RejectedBy, Valid: true}, RejectReason: rejection.Reason,
	}); err != nil {
		return fmt.Errorf("reject booking %d: %w", booking.ID, err)
	}
	if booking.StockHeld {
		if err := queries.ReleaseRoomLock(ctx, booking.RoomID); err != nil {
			return fmt.Errorf("release room %d: %w", booking.RoomID, err)
		}
	}
	refundID, err := queries.InsertRefund(ctx, dbgen.InsertRefundParams{
		BookingID: booking.ID, InvoiceID: booking.InvoiceID, Amount: booking.InvoiceTotal, Reason: rejection.RefundReason,
	})
	if err != nil {
		return fmt.Errorf("insert refund for booking %d: %w", booking.ID, err)
	}
	ledger := finance.Entry{
		Type: finance.TypeRefundDue, Amount: booking.InvoiceTotal, OwnerID: &booking.OwnerID, TenantID: &booking.TenantID,
		BookingID: &booking.ID, InvoiceID: &booking.InvoiceID, RefundID: &refundID,
		Description: fmt.Sprintf("Refund due for invoice %s (%s)", booking.InvoiceNumber, rejection.RefundReason),
	}
	if _, err := finance.Record(ctx, queries, ledger); err != nil {
		return err
	}
	if booking.InvoiceAdminFee > 0 {
		ledger.Type, ledger.Amount = finance.TypeAdminFeeReversed, -booking.InvoiceAdminFee
		ledger.Description = fmt.Sprintf("Admin fee reversed for invoice %s (%s)", booking.InvoiceNumber, rejection.RefundReason)
		if _, err := finance.Record(ctx, queries, ledger); err != nil {
			return err
		}
	}
	return NotifyUser(ctx, queries, appPublicURL, booking.TenantID, notification.CategoryBooking,
		rejection.TenantTitle, rejection.TenantBody, fmt.Sprintf("/booking/%d", booking.ID))
}

func lockRespondableBooking(ctx context.Context, w http.ResponseWriter, r *http.Request, queries *dbgen.Queries, user authctx.User) (dbgen.LockPaidBookingRow, bool) {
	bookingID, ok := parseID(r, "booking")
	if !ok {
		response.NotFoundHandler(w, r)
		return dbgen.LockPaidBookingRow{}, false
	}
	booking, err := queries.LockPaidBooking(ctx, dbgen.LockPaidBookingParams{ID: bookingID, OwnerID: pgtype.Int8{Int64: user.UserID, Valid: true}})
	if errors.Is(err, pgx.ErrNoRows) {
		response.NotFoundHandler(w, r)
		return dbgen.LockPaidBookingRow{}, false
	}
	if err != nil {
		response.InternalErrorHandler(w, r)
		return dbgen.LockPaidBookingRow{}, false
	}
	if booking.Status != "paid" || !time.Now().Before(booking.OwnerDeadlineAt.Time) {
		response.Fail(w, http.StatusBadRequest, msgBookingNotProcessable, nil)
		return dbgen.LockPaidBookingRow{}, false
	}
	return booking, true
}

func ListBookingsHandler(deps Deps, user authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, limit := pageParam(r), limitParam(r)
		bookings, err := loadOwnerBookings(r.Context(), deps.Store, dbgen.ListOwnerBookingsParams{
			OwnerID: user.UserID, Status: optionalQueryText(r, "status"), PropertyID: optionalQueryID(r, "property_id"),
			RowLimit: limit, RowOffset: (page - 1) * limit,
		})
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		response.Success(w, http.StatusOK, "", map[string]any{"bookings": bookings, "current_page": page})
	}
}

func GetBookingHandler(deps Deps, user authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bookingID, ok := parseID(r, "booking")
		if !ok {
			response.NotFoundHandler(w, r)
			return
		}
		respondWithBooking(w, r, deps, user.UserID, bookingID, "")
	}
}

func ApproveBookingHandler(deps Deps, user authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		tx, err := deps.DB.Begin(ctx)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		defer func() { _ = tx.Rollback(ctx) }()
		queries := dbgen.New(tx)

		booking, ok := lockRespondableBooking(ctx, w, r, queries, user)
		if !ok {
			return
		}
		if err := approveBooking(ctx, deps, r, user, queries, booking); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if err := tx.Commit(ctx); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		respondWithBooking(w, r, deps, user.UserID, booking.ID, "Booking disetujui.")
	}
}

func approveBooking(ctx context.Context, deps Deps, r *http.Request, user authctx.User, queries *dbgen.Queries, booking dbgen.LockPaidBookingRow) error {
	if err := queries.ApproveBooking(ctx, booking.ID); err != nil {
		return fmt.Errorf("approve booking %d: %w", booking.ID, err)
	}
	payoutAmount := booking.InvoiceTotal - booking.InvoiceAdminFee
	payoutID, err := queries.InsertPayout(ctx, dbgen.InsertPayoutParams{
		BookingID: booking.ID, InvoiceID: booking.InvoiceID, OwnerID: booking.OwnerID, Amount: payoutAmount,
	})
	if err != nil {
		return fmt.Errorf("insert payout for booking %d: %w", booking.ID, err)
	}
	if _, err := finance.Record(ctx, queries, finance.Entry{
		Type: finance.TypeOwnerEarning, Amount: payoutAmount, OwnerID: &booking.OwnerID, TenantID: &booking.TenantID,
		BookingID: &booking.ID, InvoiceID: &booking.InvoiceID, PayoutID: &payoutID, CreatedBy: &user.UserID,
		Description: fmt.Sprintf("Owner earning for invoice %s", booking.InvoiceNumber),
	}); err != nil {
		return err
	}
	if err := NotifyUser(ctx, queries, deps.AppPublicURL, booking.TenantID, notification.CategoryBooking, "Booking disetujui",
		fmt.Sprintf("Pemilik menyetujui booking %s di %s. Mulai sewa %s.", booking.RoomName, booking.PropertyName, money.FormatDateWIB(booking.StartDate.Time)),
		fmt.Sprintf("/booking/%d", booking.ID)); err != nil {
		return err
	}
	return recordAudit(ctx, queries, r, user, "booking.approve", "booking", booking.ID,
		map[string]any{"status": "paid"}, map[string]any{"status": "approved", "payout_amount": payoutAmount})
}

func RejectBookingHandler(deps Deps, user authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		tx, err := deps.DB.Begin(ctx)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		defer func() { _ = tx.Rollback(ctx) }()
		queries := dbgen.New(tx)

		booking, ok := lockRespondableBooking(ctx, w, r, queries, user)
		if !ok {
			return
		}
		body := decodeBody(r)
		v := validation.New()
		v.Field("reason", body["reason"]).Optional().String().MaxStringLen(500)
		if !v.Valid() {
			response.Fail(w, http.StatusUnprocessableEntity, response.MsgValidationFailed, v.Errors())
			return
		}

		reason := optionalText(body["reason"])
		tenantBody := fmt.Sprintf("Pemilik menolak booking %s di %s. Dana akan dikembalikan penuh. Silakan isi rekening pengembalian dana.",
			booking.RoomName, booking.PropertyName)
		if reason.Valid {
			tenantBody += " Alasan: " + reason.String
		}
		if err := RejectPaidBooking(ctx, queries, deps.AppPublicURL, booking, Rejection{
			RejectedBy: "owner", Reason: reason, RefundReason: "owner_rejected", TenantTitle: "Booking ditolak", TenantBody: tenantBody,
		}); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if err := recordAudit(ctx, queries, r, user, "booking.reject", "booking", booking.ID,
			map[string]any{"status": "paid"}, map[string]any{"status": "rejected", "reject_reason": nullableText(reason)}); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if err := tx.Commit(ctx); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		respondWithBooking(w, r, deps, user.UserID, booking.ID, "Booking ditolak. Dana akan dikembalikan ke penyewa.")
	}
}
