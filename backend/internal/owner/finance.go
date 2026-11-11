package owner

import (
	"errors"
	"net/http"
	"regexp"

	"github.com/jackc/pgx/v5"

	"github.com/suwahadi/sewakos/backend/internal/authctx"
	"github.com/suwahadi/sewakos/backend/internal/booking"
	"github.com/suwahadi/sewakos/backend/internal/money"
	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
	"github.com/suwahadi/sewakos/backend/internal/response"
	"github.com/suwahadi/sewakos/backend/internal/validation"
)

var accountNumberPattern = regexp.MustCompile(`^\d{5,20}$`)

func ListInvoicesHandler(deps Deps, user authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, limit := pageParam(r), limitParam(r)
		rows, err := deps.Store.ListInvoicesForOwner(r.Context(), dbgen.ListInvoicesForOwnerParams{
			OwnerID: user.UserID, BookingID: optionalQueryID(r, "booking_id"), Status: optionalQueryText(r, "status"),
			RowLimit: limit, RowOffset: (page - 1) * limit,
		})
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		invoices := make([]map[string]any, len(rows))
		for i, row := range rows {
			invoices[i] = booking.InvoiceBody(row.ID, row.Number, row.BookingID, row.Type, row.Status,
				row.PeriodStart, row.PeriodEnd, row.RentAmount, row.Discount, row.Deposit, row.AdminFee, row.Total,
				row.DueDate, row.PaidAt, row.IssuedAt)
		}
		response.Success(w, http.StatusOK, "", map[string]any{"invoices": invoices, "current_page": page})
	}
}

func bankAccountBody(account dbgen.GetOwnerBankAccountRow) map[string]any {
	return map[string]any{"bank_name": account.BankName, "account_number": account.AccountNumber, "account_name": account.AccountName}
}

func GetBankAccountHandler(deps Deps, user authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		account, err := deps.Store.GetOwnerBankAccount(r.Context(), user.UserID)
		if errors.Is(err, pgx.ErrNoRows) {
			response.Success(w, http.StatusOK, "", map[string]any{"bank_account": nil})
			return
		}
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		response.Success(w, http.StatusOK, "", map[string]any{"bank_account": bankAccountBody(account)})
	}
}

func PutBankAccountHandler(deps Deps, user authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body := decodeBody(r)
		v := validation.New()
		v.Field("bank_name", body["bank_name"]).Required().String().MaxStringLen(100)
		v.Field("account_number", body["account_number"]).Required().String().Regex(accountNumberPattern)
		v.Field("account_name", body["account_name"]).Required().String().MaxStringLen(255)
		if !v.Valid() {
			response.Fail(w, http.StatusUnprocessableEntity, response.MsgValidationFailed, v.Errors())
			return
		}
		account := dbgen.GetOwnerBankAccountRow{
			BankName: body["bank_name"].(string), AccountNumber: body["account_number"].(string), AccountName: body["account_name"].(string),
		}

		ctx := r.Context()
		tx, err := deps.DB.Begin(ctx)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		defer func() { _ = tx.Rollback(ctx) }()
		queries := dbgen.New(tx)

		var before map[string]any
		if current, err := queries.GetOwnerBankAccount(ctx, user.UserID); err == nil {
			before = bankAccountBody(current)
		} else if !errors.Is(err, pgx.ErrNoRows) {
			response.InternalErrorHandler(w, r)
			return
		}
		if err := queries.UpsertOwnerBankAccount(ctx, dbgen.UpsertOwnerBankAccountParams{
			UserID: user.UserID, BankName: account.BankName, AccountNumber: account.AccountNumber, AccountName: account.AccountName,
		}); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if err := recordAudit(ctx, queries, r, user, "bank_account.update", "owner_bank_account", user.UserID, before, bankAccountBody(account)); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if err := tx.Commit(ctx); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		response.Success(w, http.StatusOK, "Rekening disimpan.", map[string]any{"bank_account": bankAccountBody(account)})
	}
}

func ListPayoutsHandler(deps Deps, user authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, limit := pageParam(r), limitParam(r)
		rows, err := deps.Store.ListOwnerPayouts(r.Context(), dbgen.ListOwnerPayoutsParams{
			OwnerID: user.UserID, Status: optionalQueryText(r, "status"), RowLimit: limit, RowOffset: (page - 1) * limit,
		})
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		payouts := make([]map[string]any, len(rows))
		for i, row := range rows {
			payouts[i] = map[string]any{
				"id": row.ID, "booking_id": row.BookingID, "amount": row.Amount, "amount_label": money.FormatRupiah(row.Amount),
				"status": row.Status, "transfer_reference": nullableText(row.TransferReference),
				"completed_at": nullableTimestamp(row.CompletedAt), "created_at": money.FormatTimestamp(row.CreatedAt.Time),
			}
		}
		response.Success(w, http.StatusOK, "", map[string]any{"payouts": payouts, "current_page": page})
	}
}
