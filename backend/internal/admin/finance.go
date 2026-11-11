package admin

import (
	"encoding/csv"
	"encoding/json"
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
	"github.com/suwahadi/sewakos/backend/internal/owner"
	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
	"github.com/suwahadi/sewakos/backend/internal/response"
	"github.com/suwahadi/sewakos/backend/internal/validation"
)

func bankAccountBody(bankName, accountNumber, accountName pgtype.Text) any {
	if !bankName.Valid {
		return nil
	}
	return map[string]any{"bank_name": bankName.String, "account_number": accountNumber.String, "account_name": accountName.String}
}

func refundBody(row dbgen.AdminListRefundsRow) map[string]any {
	return map[string]any{
		"id": row.ID, "booking_id": row.BookingID, "tenant": map[string]any{"id": row.TenantID, "name": row.TenantName},
		"amount": row.Amount, "amount_label": money.FormatRupiah(row.Amount), "reason": row.Reason, "status": row.Status,
		"bank_account":       bankAccountBody(row.BankName, row.AccountNumber, row.AccountName),
		"transfer_reference": nullableText(row.TransferReference), "note": nullableText(row.Note),
		"completed_at": nullableTimestamp(row.CompletedAt), "created_at": money.FormatTimestamp(row.CreatedAt.Time),
	}
}

func payoutBody(row dbgen.AdminListPayoutsRow) map[string]any {
	account := bankAccountBody(row.BankName, row.AccountNumber, row.AccountName)
	if row.Status == "pending" {
		account = bankAccountBody(row.CurrentBankName, row.CurrentAccountNumber, row.CurrentAccountName)
	}
	return map[string]any{
		"id": row.ID, "booking_id": row.BookingID, "owner": map[string]any{"id": row.OwnerID, "name": row.OwnerName},
		"amount": row.Amount, "amount_label": money.FormatRupiah(row.Amount), "status": row.Status,
		"bank_account": account, "transfer_reference": nullableText(row.TransferReference),
		"completed_at": nullableTimestamp(row.CompletedAt), "created_at": money.FormatTimestamp(row.CreatedAt.Time),
	}
}

func validateTransfer(body map[string]any) *validation.Validator {
	v := validation.New()
	v.Field("transfer_reference", body["transfer_reference"]).Required().String().MaxStringLen(100)
	v.Field("note", body["note"]).Optional().String().MaxStringLen(500)
	return v
}

func optionalNote(body map[string]any) pgtype.Text {
	note, _ := body["note"].(string)
	return pgtype.Text{String: note, Valid: note != ""}
}

func ListRefundsHandler(deps Deps, _ authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, limit := pageParam(r), limitParam(r)
		rows, err := deps.Store.AdminListRefunds(r.Context(), dbgen.AdminListRefundsParams{
			Status: queryText(r, "status"), RowLimit: limit, RowOffset: (page - 1) * limit,
		})
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		refunds := make([]map[string]any, len(rows))
		for i, row := range rows {
			refunds[i] = refundBody(row)
		}
		response.Success(w, http.StatusOK, "", map[string]any{"refunds": refunds, "current_page": page})
	}
}

func CompleteRefundHandler(deps Deps, user authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		refundID, ok := parseID(r, "refund")
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
		defer func() { _ = tx.Rollback(ctx) }()
		queries := dbgen.New(tx)

		refund, err := queries.AdminLockRefund(ctx, refundID)
		if errors.Is(err, pgx.ErrNoRows) {
			response.NotFoundHandler(w, r)
			return
		}
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if refund.Status != "ready" {
			response.Fail(w, http.StatusBadRequest, "Refund belum siap diproses.", nil)
			return
		}
		body := decodeBody(r)
		if v := validateTransfer(body); !v.Valid() {
			response.Fail(w, http.StatusUnprocessableEntity, response.MsgValidationFailed, v.Errors())
			return
		}
		reference := body["transfer_reference"].(string)
		if err := queries.CompleteRefund(ctx, dbgen.CompleteRefundParams{
			ID: refund.ID, TransferReference: pgtype.Text{String: reference, Valid: true}, Note: optionalNote(body),
			CompletedBy: pgtype.Int8{Int64: user.UserID, Valid: true},
		}); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if _, err := finance.Record(ctx, queries, finance.Entry{
			Type: finance.TypeRefundCompleted, Amount: -refund.Amount, TenantID: &refund.TenantID, BookingID: &refund.BookingID,
			InvoiceID: &refund.InvoiceID, RefundID: &refund.ID, CreatedBy: &user.UserID,
			Description: fmt.Sprintf("Refund transferred (%s)", reference),
		}); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if err := owner.NotifyUser(ctx, queries, deps.AppPublicURL, refund.TenantID, notification.CategoryPayment, "Pengembalian dana selesai",
			fmt.Sprintf("Dana %s untuk booking #%d sudah ditransfer.", money.FormatRupiah(refund.Amount), refund.BookingID),
			fmt.Sprintf("/booking/%d", refund.BookingID)); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if err := recordAudit(ctx, queries, r, user, "refund.complete", "refund", refund.ID, nil,
			map[string]any{"status": refund.Status}, map[string]any{"status": "completed", "transfer_reference": reference}); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		rows, err := queries.AdminListRefunds(ctx, dbgen.AdminListRefundsParams{RefundID: pgtype.Int8{Int64: refund.ID, Valid: true}, RowLimit: 1})
		if err != nil || len(rows) == 0 {
			response.InternalErrorHandler(w, r)
			return
		}
		if err := tx.Commit(ctx); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		response.Success(w, http.StatusOK, "Refund ditandai selesai.", map[string]any{"refund": refundBody(rows[0])})
	}
}

func ListPayoutsHandler(deps Deps, _ authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, limit := pageParam(r), limitParam(r)
		rows, err := deps.Store.AdminListPayouts(r.Context(), dbgen.AdminListPayoutsParams{
			Status: queryText(r, "status"), RowLimit: limit, RowOffset: (page - 1) * limit,
		})
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		payouts := make([]map[string]any, len(rows))
		for i, row := range rows {
			payouts[i] = payoutBody(row)
		}
		response.Success(w, http.StatusOK, "", map[string]any{"payouts": payouts, "current_page": page})
	}
}

func CompletePayoutHandler(deps Deps, user authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		payoutID, ok := parseID(r, "payout")
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
		defer func() { _ = tx.Rollback(ctx) }()
		queries := dbgen.New(tx)

		payout, err := queries.AdminLockPayout(ctx, payoutID)
		if errors.Is(err, pgx.ErrNoRows) {
			response.NotFoundHandler(w, r)
			return
		}
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if payout.Status != "pending" {
			response.Fail(w, http.StatusBadRequest, "Payout sudah diproses.", nil)
			return
		}
		account, err := queries.GetOwnerBankAccount(ctx, payout.OwnerID)
		if errors.Is(err, pgx.ErrNoRows) {
			response.Fail(w, http.StatusBadRequest, "Pemilik belum mengisi rekening bank.", nil)
			return
		}
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		body := decodeBody(r)
		if v := validateTransfer(body); !v.Valid() {
			response.Fail(w, http.StatusUnprocessableEntity, response.MsgValidationFailed, v.Errors())
			return
		}
		reference := body["transfer_reference"].(string)
		if err := queries.CompletePayout(ctx, dbgen.CompletePayoutParams{
			ID: payout.ID, BankName: pgtype.Text{String: account.BankName, Valid: true},
			AccountNumber: pgtype.Text{String: account.AccountNumber, Valid: true}, AccountName: pgtype.Text{String: account.AccountName, Valid: true},
			TransferReference: pgtype.Text{String: reference, Valid: true}, Note: optionalNote(body),
			CompletedBy: pgtype.Int8{Int64: user.UserID, Valid: true},
		}); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if _, err := finance.Record(ctx, queries, finance.Entry{
			Type: finance.TypePayoutCompleted, Amount: -payout.Amount, OwnerID: &payout.OwnerID, BookingID: &payout.BookingID,
			InvoiceID: &payout.InvoiceID, PayoutID: &payout.ID, CreatedBy: &user.UserID,
			Description: fmt.Sprintf("Payout transferred (%s)", reference),
		}); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if err := owner.NotifyUser(ctx, queries, deps.AppPublicURL, payout.OwnerID, notification.CategoryPayment, "Dana sudah ditransfer",
			fmt.Sprintf("Dana %s untuk booking #%d sudah ditransfer ke rekening kamu.", money.FormatRupiah(payout.Amount), payout.BookingID),
			"/owner/keuangan"); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if err := recordAudit(ctx, queries, r, user, "payout.complete", "payout", payout.ID, &payout.OwnerID,
			map[string]any{"status": payout.Status}, map[string]any{"status": "completed", "transfer_reference": reference}); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		rows, err := queries.AdminListPayouts(ctx, dbgen.AdminListPayoutsParams{PayoutID: pgtype.Int8{Int64: payout.ID, Valid: true}, RowLimit: 1})
		if err != nil || len(rows) == 0 {
			response.InternalErrorHandler(w, r)
			return
		}
		if err := tx.Commit(ctx); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		response.Success(w, http.StatusOK, "Payout ditandai selesai.", map[string]any{"payout": payoutBody(rows[0])})
	}
}

func respondInvalidDateRange(w http.ResponseWriter, r *http.Request, errs map[string][]string) {
	response.Fail(w, http.StatusUnprocessableEntity, response.MsgValidationFailed, errs)
}

func toPgTimestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

func FinanceSummaryHandler(deps Deps, _ authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		from, toExclusive, errs := finance.ParseDateRange(r)
		if errs != nil {
			respondInvalidDateRange(w, r, errs)
			return
		}
		ctx := r.Context()

		summary, err := deps.Store.GetAdminFinanceSummary(ctx, dbgen.GetAdminFinanceSummaryParams{
			FromTime: toPgTimestamptz(from), ToTime: toPgTimestamptz(toExclusive),
		})
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		outstanding, err := deps.Store.GetAdminFinanceOutstanding(ctx)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}

		response.Success(w, http.StatusOK, "", map[string]any{"summary": map[string]any{
			"from": from.Format("2006-01-02"), "to": toExclusive.AddDate(0, 0, -1).Format("2006-01-02"),
			"payments_received": summary.PaymentsReceived, "payments_received_label": money.FormatRupiah(summary.PaymentsReceived),
			"admin_fee_revenue": summary.AdminFeeRevenue, "admin_fee_revenue_label": money.FormatRupiah(summary.AdminFeeRevenue),
			"refunds_completed": summary.RefundsCompleted, "refunds_completed_label": money.FormatRupiah(summary.RefundsCompleted),
			"refunds_outstanding": outstanding.RefundsOutstanding, "refunds_outstanding_label": money.FormatRupiah(outstanding.RefundsOutstanding),
			"payouts_completed": summary.PayoutsCompleted, "payouts_completed_label": money.FormatRupiah(summary.PayoutsCompleted),
			"payouts_outstanding": outstanding.PayoutsOutstanding, "payouts_outstanding_label": money.FormatRupiah(outstanding.PayoutsOutstanding),
			"open_reconciliation_issues": outstanding.OpenReconciliationIssues,
		}})
	}
}

func adminLedgerEntryBody(id int64, occurredAt pgtype.Timestamptz, entryType string, amount int64, ownerID, bookingID pgtype.Int8, invoiceNumber pgtype.Text, description string) map[string]any {
	return map[string]any{
		"id": id, "occurred_at": money.FormatExpiresAt(occurredAt.Time), "type": entryType,
		"amount": amount, "amount_label": money.FormatRupiah(amount),
		"owner_id": nullableInt8(ownerID), "booking_id": nullableInt8(bookingID),
		"invoice_number": nullableText(invoiceNumber), "description": description,
	}
}

func FinanceLedgerHandler(deps Deps, _ authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		from, toExclusive, errs := finance.ParseDateRange(r)
		if errs != nil {
			respondInvalidDateRange(w, r, errs)
			return
		}
		page, limit := pageParam(r), limitParam(r)
		rows, err := deps.Store.ListAdminLedgerEntries(r.Context(), dbgen.ListAdminLedgerEntriesParams{
			FromTime: toPgTimestamptz(from), ToTime: toPgTimestamptz(toExclusive),
			Type: queryText(r, "type"), OwnerID: queryID(r, "owner_id"), RowLimit: limit, RowOffset: (page - 1) * limit,
		})
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		entries := make([]map[string]any, len(rows))
		for i, row := range rows {
			entries[i] = adminLedgerEntryBody(row.ID, row.OccurredAt, row.Type, row.Amount, row.OwnerID, row.BookingID, row.InvoiceNumber, row.Description)
		}
		response.Success(w, http.StatusOK, "", map[string]any{"entries": entries, "current_page": page})
	}
}

func FinanceLedgerCSVHandler(deps Deps, _ authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		from, toExclusive, errs := finance.ParseDateRange(r)
		if errs != nil {
			respondInvalidDateRange(w, r, errs)
			return
		}
		rows, err := deps.Store.StreamAdminLedgerEntries(r.Context(), dbgen.StreamAdminLedgerEntriesParams{
			FromTime: toPgTimestamptz(from), ToTime: toPgTimestamptz(toExclusive),
			Type: queryText(r, "type"), OwnerID: queryID(r, "owner_id"),
		})
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}

		lastDay := toExclusive.AddDate(0, 0, -1).Format("2006-01-02")
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="ledger-%s-%s.csv"`, from.Format("2006-01-02"), lastDay))
		writer := csv.NewWriter(w)
		_ = writer.Write([]string{"id", "occurred_at_wib", "type", "amount", "owner_id", "booking_id", "invoice_number", "description"})
		for _, row := range rows {
			_ = writer.Write([]string{
				strconv.FormatInt(row.ID, 10), money.FormatHumanWIB(row.OccurredAt.Time), row.Type, strconv.FormatInt(row.Amount, 10),
				csvNullableInt8(row.OwnerID), csvNullableInt8(row.BookingID), csvNullableText(row.InvoiceNumber), row.Description,
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

func reconciliationIssueBody(row dbgen.ListReconciliationIssuesRow) map[string]any {
	return map[string]any{
		"id": row.ID, "order_id": row.OrderID, "kind": row.Kind,
		"local_status": row.LocalStatus, "midtrans_status": nullableText(row.MidtransStatus),
		"local_amount": row.LocalAmount, "midtrans_amount": nullableInt8(row.MidtransAmount),
		"status": row.Status, "note": nullableText(row.Note), "detected_at": money.FormatExpiresAt(row.DetectedAt.Time),
	}
}

func ListReconciliationIssuesHandler(deps Deps, _ authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, limit := pageParam(r), limitParam(r)
		rows, err := deps.Store.ListReconciliationIssues(r.Context(), dbgen.ListReconciliationIssuesParams{
			Status: queryText(r, "status"), RowLimit: limit, RowOffset: (page - 1) * limit,
		})
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		issues := make([]map[string]any, len(rows))
		for i, row := range rows {
			issues[i] = reconciliationIssueBody(row)
		}
		response.Success(w, http.StatusOK, "", map[string]any{"issues": issues, "current_page": page})
	}
}

func ResolveReconciliationIssueHandler(deps Deps, user authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		issueID, ok := parseID(r, "issue")
		if !ok {
			response.NotFoundHandler(w, r)
			return
		}
		body := decodeBody(r)
		v := validation.New()
		v.Field("note", body["note"]).Required().String().MaxStringLen(500)
		if !v.Valid() {
			response.Fail(w, http.StatusUnprocessableEntity, response.MsgValidationFailed, v.Errors())
			return
		}

		ctx := r.Context()
		tx, err := deps.DB.Begin(ctx)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		defer func() { _ = tx.Rollback(ctx) }()
		queries := dbgen.New(tx)

		issue, err := queries.LockReconciliationIssue(ctx, issueID)
		if errors.Is(err, pgx.ErrNoRows) {
			response.NotFoundHandler(w, r)
			return
		}
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if issue.Status != "open" {
			response.Fail(w, http.StatusBadRequest, "Selisih sudah ditandai selesai.", nil)
			return
		}
		note := body["note"].(string)
		if err := queries.ResolveReconciliationIssue(ctx, dbgen.ResolveReconciliationIssueParams{
			ID: issue.ID, Note: pgtype.Text{String: note, Valid: true}, ResolvedBy: pgtype.Int8{Int64: user.UserID, Valid: true},
		}); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if err := recordAudit(ctx, queries, r, user, "reconciliation_issue.resolve", "reconciliation_issue", issue.ID, nil,
			map[string]any{"status": "open"}, map[string]any{"status": "resolved", "note": note}); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if err := tx.Commit(ctx); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		response.Success(w, http.StatusOK, "Selisih ditandai selesai.", nil)
	}
}

func settingsBody(adminFee int64) map[string]any {
	return map[string]any{"settings": map[string]any{"admin_fee": adminFee, "admin_fee_label": money.FormatRupiah(adminFee)}}
}

func currentAdminFee(r *http.Request, store dbgen.Querier) (int64, error) {
	raw, err := store.GetSetting(r.Context(), "admin_fee")
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	var fee int64
	return fee, json.Unmarshal(raw, &fee)
}

func GetSettingsHandler(deps Deps, _ authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fee, err := currentAdminFee(r, deps.Store)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		response.Success(w, http.StatusOK, "", settingsBody(fee))
	}
}

func PatchSettingsHandler(deps Deps, user authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body := decodeBody(r)
		v := validation.New()
		v.Field("admin_fee", body["admin_fee"]).Required().Integer().BetweenNumber(0, 1000000)
		if !v.Valid() {
			response.Fail(w, http.StatusUnprocessableEntity, response.MsgValidationFailed, v.Errors())
			return
		}
		fee, _ := wholeNumber(body["admin_fee"])
		before, err := currentAdminFee(r, deps.Store)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}

		ctx := r.Context()
		tx, err := deps.DB.Begin(ctx)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		defer func() { _ = tx.Rollback(ctx) }()
		queries := dbgen.New(tx)
		if err := queries.UpsertSetting(ctx, dbgen.UpsertSettingParams{Key: "admin_fee", Value: []byte(fmt.Sprint(fee))}); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if err := recordAudit(ctx, queries, r, user, "settings.update", "setting", 0, nil,
			map[string]any{"admin_fee": before}, map[string]any{"admin_fee": fee}); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if err := tx.Commit(ctx); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if deps.AdminFee != nil {
			deps.AdminFee.Invalidate()
		}
		response.Success(w, http.StatusOK, "Pengaturan disimpan.", settingsBody(fee))
	}
}

func BroadcastHandler(deps Deps, user authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body := decodeBody(r)
		v := validation.New()
		v.Field("target", body["target"]).Required().In("all", "tenant", "owner")
		v.Field("title", body["title"]).Required().String().MaxStringLen(255)
		v.Field("body", body["body"]).Required().String().MaxStringLen(1000)
		if !v.Valid() {
			response.Fail(w, http.StatusUnprocessableEntity, response.MsgValidationFailed, v.Errors())
			return
		}

		ctx := r.Context()
		tx, err := deps.DB.Begin(ctx)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		defer func() { _ = tx.Rollback(ctx) }()
		queries := dbgen.New(tx)
		params := dbgen.BroadcastNotificationParams{Target: body["target"].(string), Title: body["title"].(string), Body: body["body"].(string)}
		sent, err := queries.BroadcastNotification(ctx, params)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if err := recordAudit(ctx, queries, r, user, "notification.broadcast", "notification_broadcast", 0, nil, nil,
			map[string]any{"target": params.Target, "title": params.Title, "sent": sent}); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if err := tx.Commit(ctx); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		response.Success(w, http.StatusOK, "Notifikasi dikirim.", map[string]any{"sent": sent})
	}
}
