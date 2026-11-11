package owner

import (
	"encoding/csv"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/suwahadi/sewakos/backend/internal/authctx"
	"github.com/suwahadi/sewakos/backend/internal/booking"
	"github.com/suwahadi/sewakos/backend/internal/finance"
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

func respondInvalidDateRange(w http.ResponseWriter, r *http.Request, errs map[string][]string) {
	response.Fail(w, http.StatusUnprocessableEntity, response.MsgValidationFailed, errs)
}

func ledgerType(r *http.Request) pgtype.Text {
	return optionalQueryText(r, "type")
}

func FinanceSummaryHandler(deps Deps, user authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		from, toExclusive, errs := finance.ParseDateRange(r)
		if errs != nil {
			respondInvalidDateRange(w, r, errs)
			return
		}
		ctx := r.Context()
		ownerID := pgtype.Int8{Int64: user.UserID, Valid: true}

		summary, err := deps.Store.GetOwnerFinanceSummary(ctx, dbgen.GetOwnerFinanceSummaryParams{
			OwnerID: ownerID, FromTime: toPgTimestamptz(from), ToTime: toPgTimestamptz(toExclusive),
		})
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		pendingBalance, err := deps.Store.GetOwnerPendingBalance(ctx, ownerID)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		depositsHeld, err := deps.Store.GetOwnerDepositsHeld(ctx, user.UserID)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}

		response.Success(w, http.StatusOK, "", map[string]any{"summary": map[string]any{
			"from": from.Format("2006-01-02"), "to": toExclusive.AddDate(0, 0, -1).Format("2006-01-02"),
			"earnings": summary.Earnings, "earnings_label": money.FormatRupiah(summary.Earnings),
			"payouts_completed": summary.PayoutsCompleted, "payouts_completed_label": money.FormatRupiah(summary.PayoutsCompleted),
			"pending_balance": pendingBalance, "pending_balance_label": money.FormatRupiah(pendingBalance),
			"deposits_held": depositsHeld, "deposits_held_label": money.FormatRupiah(depositsHeld),
			"paid_invoices": summary.PaidInvoices,
		}})
	}
}

func ledgerEntryBody(id int64, occurredAt pgtype.Timestamptz, entryType string, amount int64, bookingID pgtype.Int8, invoiceNumber pgtype.Text, description string) map[string]any {
	return map[string]any{
		"id": id, "occurred_at": money.FormatExpiresAt(occurredAt.Time), "type": entryType,
		"amount": amount, "amount_label": money.FormatRupiah(amount),
		"booking_id": nullableInt8(bookingID), "invoice_number": nullableText(invoiceNumber), "description": description,
	}
}

func FinanceLedgerHandler(deps Deps, user authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		from, toExclusive, errs := finance.ParseDateRange(r)
		if errs != nil {
			respondInvalidDateRange(w, r, errs)
			return
		}
		page, limit := pageParam(r), limitParam(r)
		rows, err := deps.Store.ListOwnerLedgerEntries(r.Context(), dbgen.ListOwnerLedgerEntriesParams{
			OwnerID: pgtype.Int8{Int64: user.UserID, Valid: true}, FromTime: toPgTimestamptz(from), ToTime: toPgTimestamptz(toExclusive),
			Type: ledgerType(r), RowLimit: limit, RowOffset: (page - 1) * limit,
		})
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		entries := make([]map[string]any, len(rows))
		for i, row := range rows {
			entries[i] = ledgerEntryBody(row.ID, row.OccurredAt, row.Type, row.Amount, row.BookingID, row.InvoiceNumber, row.Description)
		}
		response.Success(w, http.StatusOK, "", map[string]any{"entries": entries, "current_page": page})
	}
}

func FinanceLedgerCSVHandler(deps Deps, user authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		from, toExclusive, errs := finance.ParseDateRange(r)
		if errs != nil {
			respondInvalidDateRange(w, r, errs)
			return
		}
		rows, err := deps.Store.StreamOwnerLedgerEntries(r.Context(), dbgen.StreamOwnerLedgerEntriesParams{
			OwnerID: pgtype.Int8{Int64: user.UserID, Valid: true}, FromTime: toPgTimestamptz(from), ToTime: toPgTimestamptz(toExclusive),
			Type: ledgerType(r),
		})
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}

		lastDay := toExclusive.AddDate(0, 0, -1).Format("2006-01-02")
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="ledger-%s-%s.csv"`, from.Format("2006-01-02"), lastDay))
		writer := csv.NewWriter(w)
		_ = writer.Write([]string{"id", "occurred_at_wib", "type", "amount", "booking_id", "invoice_number", "description"})
		for _, row := range rows {
			_ = writer.Write([]string{
				strconv.FormatInt(row.ID, 10), money.FormatHumanWIB(row.OccurredAt.Time), row.Type, strconv.FormatInt(row.Amount, 10),
				csvNullableInt8(row.BookingID), csvNullableText(row.InvoiceNumber), row.Description,
			})
		}
		writer.Flush()
	}
}

func csvNullableInt8(n pgtype.Int8) string {
	if !n.Valid {
		return ""
	}
	return strconv.FormatInt(n.Int64, 10)
}

func csvNullableText(t pgtype.Text) string {
	if !t.Valid {
		return ""
	}
	return t.String
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
