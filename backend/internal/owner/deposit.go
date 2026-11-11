package owner

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"

	"github.com/suwahadi/sewakos/backend/internal/authctx"
	"github.com/suwahadi/sewakos/backend/internal/finance"
	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
	"github.com/suwahadi/sewakos/backend/internal/response"
	"github.com/suwahadi/sewakos/backend/internal/validation"
)

const msgDepositReturnNotRecordable = "Pengembalian deposit tidak dapat dicatat."

func DepositReturnedHandler(deps Deps, user authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bookingID, ok := parseID(r, "booking")
		if !ok {
			response.NotFoundHandler(w, r)
			return
		}
		ctx := r.Context()

		body := decodeBody(r)
		v := validation.New()
		v.Field("note", body["note"]).Optional().String().MaxStringLen(500)
		if !v.Valid() {
			response.Fail(w, http.StatusUnprocessableEntity, response.MsgValidationFailed, v.Errors())
			return
		}

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

		booking, err := queries.LockBookingForDepositReturn(ctx, dbgen.LockBookingForDepositReturnParams{ID: bookingID, OwnerID: user.UserID})
		if errors.Is(err, pgx.ErrNoRows) {
			response.NotFoundHandler(w, r)
			return
		}
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if (booking.Status != "completed" && booking.Status != "terminated") || booking.DepositReturnedAt.Valid {
			response.Fail(w, http.StatusBadRequest, msgDepositReturnNotRecordable, nil)
			return
		}

		note := optionalText(body["note"])
		if err := queries.MarkDepositReturned(ctx, dbgen.MarkDepositReturnedParams{ID: bookingID, Note: note}); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if _, err := finance.Record(ctx, queries, finance.Entry{
			Type: finance.TypeDepositReturned, Amount: booking.Deposit, OwnerID: &booking.OwnerID, TenantID: &booking.TenantID,
			BookingID: &booking.ID, CreatedBy: &user.UserID, Description: "Deposit returned to tenant by owner",
		}); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if err := recordAudit(ctx, queries, r, user, "booking.deposit_returned", "booking", booking.ID,
			map[string]any{"deposit_returned_at": nil}, map[string]any{"deposit_returned_at": "now", "note": nullableText(note)}); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if err := tx.Commit(ctx); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		committed = true

		respondWithBooking(w, r, deps, user.UserID, bookingID, "Pengembalian deposit dicatat.")
	}
}
