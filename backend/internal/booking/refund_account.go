package booking

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"

	"github.com/jackc/pgx/v5"

	"github.com/suwahadi/sewakos/backend/internal/authctx"
	"github.com/suwahadi/sewakos/backend/internal/money"
	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
	"github.com/suwahadi/sewakos/backend/internal/response"
	"github.com/suwahadi/sewakos/backend/internal/validation"
)

var accountNumberPattern = regexp.MustCompile(`^\d{5,20}$`)

func RefundAccountHandler(deps Deps, user authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bookingID, ok := parseBookingID(r)
		if !ok {
			response.NotFoundHandler(w, r)
			return
		}
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

		refund, err := queries.GetRefundForBookingInitialInvoice(ctx, bookingID)
		if errors.Is(err, pgx.ErrNoRows) {
			response.Fail(w, http.StatusBadRequest, "Tidak ada pengembalian dana untuk booking ini.", nil)
			return
		}
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if refund.TenantID != user.UserID {
			response.Fail(w, http.StatusForbidden, response.MsgAccessDenied, nil)
			return
		}
		if refund.Status == "completed" {
			response.Fail(w, http.StatusBadRequest, "Pengembalian dana sudah selesai.", nil)
			return
		}

		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		v := validation.New()
		v.Field("bank_name", body["bank_name"]).Required().String().MaxStringLen(100)
		v.Field("account_number", body["account_number"]).Required().String().Regex(accountNumberPattern)
		v.Field("account_name", body["account_name"]).Required().String().MaxStringLen(255)
		if !v.Valid() {
			response.Fail(w, http.StatusUnprocessableEntity, response.MsgValidationFailed, v.Errors())
			return
		}
		bankName, _ := body["bank_name"].(string)
		accountNumber, _ := body["account_number"].(string)
		accountName, _ := body["account_name"].(string)

		if err := queries.SetRefundAccount(ctx, dbgen.SetRefundAccountParams{
			ID: refund.ID, BankName: pgtextFrom(bankName), AccountNumber: pgtextFrom(accountNumber), AccountName: pgtextFrom(accountName),
		}); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if err := tx.Commit(ctx); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		committed = true

		response.Success(w, http.StatusOK, "Rekening pengembalian dana disimpan.", map[string]any{
			"refund": map[string]any{
				"amount": refund.Amount, "amount_label": money.FormatRupiah(refund.Amount),
				"status": "ready", "completed_at": nullableTimestamp(refund.CompletedAt),
			},
		})
	}
}
