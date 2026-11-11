package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/suwahadi/sewakos/backend/internal/audit"
	"github.com/suwahadi/sewakos/backend/internal/booking"
	"github.com/suwahadi/sewakos/backend/internal/money"
	"github.com/suwahadi/sewakos/backend/internal/notification"
	"github.com/suwahadi/sewakos/backend/internal/owner"
	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
)

// Postgres dates decode as UTC midnight while "today" is WIB midnight, so both are compared as calendar dates.
func calendarDate(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func pgDate(t time.Time) pgtype.Date {
	return pgtype.Date{Time: calendarDate(t), Valid: true}
}

func adminFeeSetting(ctx context.Context, q *dbgen.Queries) (int64, error) {
	raw, err := q.GetSetting(ctx, "admin_fee")
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	var fee int64
	return fee, json.Unmarshal(raw, &fee)
}

func IssueRenewals(ctx context.Context, deps Deps, logger *slog.Logger, today time.Time) {
	var due []dbgen.SelectDueRenewalsRow
	if err := inTx(ctx, deps, func(q *dbgen.Queries) (err error) {
		due, err = q.SelectDueRenewals(ctx, pgDate(today))
		return err
	}); err != nil {
		logger.Error("issue_renewals: select", "error", err)
		return
	}
	for _, row := range due {
		if err := inTx(ctx, deps, func(q *dbgen.Queries) error { return issueRenewal(ctx, deps, q, row) }); err != nil {
			logger.Error("issue_renewals: process", "booking_id", row.BookingID, "period_index", row.PeriodIndex, "error", err)
		}
	}
}

func issueRenewal(ctx context.Context, deps Deps, q *dbgen.Queries, row dbgen.SelectDueRenewalsRow) error {
	b, err := q.LockBookingForRenewal(ctx, dbgen.LockBookingForRenewalParams{ID: row.BookingID, PeriodIndex: row.PeriodIndex})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if b.Status != "approved" {
		return nil
	}
	exists, err := q.InvoiceExistsForPeriod(ctx, dbgen.InvoiceExistsForPeriodParams{BookingID: b.ID, PeriodStart: b.PeriodStart})
	if err != nil || exists {
		return err
	}
	fee, err := adminFeeSetting(ctx, q)
	if err != nil {
		return fmt.Errorf("read admin fee: %w", err)
	}
	total := b.PeriodPrice + fee
	issuedAt := time.Now()
	invoiceID, err := q.InsertInvoice(ctx, dbgen.InsertInvoiceParams{
		BookingID: b.ID, Type: "renewal", PeriodStart: b.PeriodStart, PeriodEnd: b.PeriodEnd, Periods: 1,
		RentAmount: b.PeriodPrice, AdminFee: fee, Total: total, DueDate: b.PeriodStart,
		IssuedAt: pgtype.Timestamptz{Time: issuedAt, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("insert renewal invoice: %w", err)
	}
	number := booking.InvoiceNumber(issuedAt, invoiceID)
	if err := q.SetInvoiceNumber(ctx, dbgen.SetInvoiceNumberParams{ID: invoiceID, Number: number}); err != nil {
		return err
	}
	return owner.NotifyUser(ctx, q, deps.Booking.AppPublicURL, b.TenantID, notification.CategoryPayment, "Tagihan baru",
		fmt.Sprintf("Tagihan %s sebesar %s jatuh tempo %s.", number, money.FormatRupiah(total), money.FormatDateWIB(b.PeriodStart.Time)),
		fmt.Sprintf("/booking/%d", b.ID))
}

func RemindRenewalsDue(ctx context.Context, deps Deps, logger *slog.Logger, today time.Time) {
	var ids []int64
	if err := inTx(ctx, deps, func(q *dbgen.Queries) (err error) {
		ids, err = q.SelectDueRenewalReminderIDs(ctx, pgDate(today))
		return err
	}); err != nil {
		logger.Error("renewal_due_reminder: select", "error", err)
		return
	}
	for _, id := range ids {
		if err := inTx(ctx, deps, func(q *dbgen.Queries) error {
			invoice, err := q.MarkInvoiceReminderSent(ctx, id)
			if errors.Is(err, pgx.ErrNoRows) {
				return nil
			}
			if err != nil {
				return err
			}
			lastDay := invoice.DueDate.Time.AddDate(0, 0, deps.RenewalGraceDays)
			return owner.NotifyUser(ctx, q, deps.Booking.AppPublicURL, invoice.TenantID, notification.CategoryPayment, "Tagihan jatuh tempo hari ini",
				fmt.Sprintf("Bayar tagihan %s paling lambat %s agar sewa tidak berakhir.", invoice.Number, money.FormatDateWIB(lastDay)),
				fmt.Sprintf("/booking/%d", invoice.BookingID))
		}); err != nil {
			logger.Error("renewal_due_reminder: process", "invoice_id", id, "error", err)
		}
	}
}

func TerminateOverdue(ctx context.Context, deps Deps, logger *slog.Logger, today time.Time) {
	var ids []int64
	if err := inTx(ctx, deps, func(q *dbgen.Queries) (err error) {
		ids, err = q.SelectOverdueRenewalIDs(ctx, dbgen.SelectOverdueRenewalIDsParams{Today: pgDate(today), GraceDays: int32(deps.RenewalGraceDays)})
		return err
	}); err != nil {
		logger.Error("terminate_overdue: select", "error", err)
		return
	}
	for _, id := range ids {
		if err := inTx(ctx, deps, func(q *dbgen.Queries) error { return terminateOverdue(ctx, deps, q, id, today) }); err != nil {
			logger.Error("terminate_overdue: process", "invoice_id", id, "error", err)
		}
	}
}

func terminateOverdue(ctx context.Context, deps Deps, q *dbgen.Queries, invoiceID int64, today time.Time) error {
	r, err := q.LockOverdueRenewal(ctx, invoiceID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if r.InvoiceStatus != "unpaid" || r.BookingStatus != "approved" ||
		!calendarDate(today).After(calendarDate(r.DueDate.Time).AddDate(0, 0, deps.RenewalGraceDays)) {
		return nil
	}
	if err := q.TerminateBooking(ctx, r.BookingID); err != nil {
		return err
	}
	if r.StockHeld {
		if err := q.ReleaseRoomLock(ctx, r.RoomID); err != nil {
			return err
		}
	}
	if err := q.VoidInvoice(ctx, dbgen.VoidInvoiceParams{ID: r.ID, VoidReason: pgtype.Text{String: "overdue", Valid: true}}); err != nil {
		return err
	}
	if payment, err := q.GetPendingPaymentOrderID(ctx, r.ID); err == nil {
		if deps.Booking.Midtrans != nil {
			_ = deps.Booking.Midtrans.ExpireTransaction(ctx, payment.OrderID)
		}
		if err := q.MarkPaymentExpired(ctx, payment.ID); err != nil {
			return err
		}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if err := owner.NotifyUser(ctx, q, deps.Booking.AppPublicURL, r.TenantID, notification.CategoryBooking, "Sewa berakhir",
		fmt.Sprintf("Sewa %s di %s berakhir karena tagihan %s tidak dibayar.", r.RoomName, r.PropertyName, r.Number),
		fmt.Sprintf("/booking/%d", r.BookingID)); err != nil {
		return err
	}
	if err := owner.NotifyUser(ctx, q, deps.Booking.AppPublicURL, r.OwnerID, notification.CategoryBooking, "Sewa penyewa berakhir",
		fmt.Sprintf("Sewa %s di %s berakhir karena tagihan tidak dibayar. Kamar kembali tersedia.", r.TenantName, r.RoomName),
		fmt.Sprintf("/owner/booking/%d", r.BookingID)); err != nil {
		return err
	}
	_, err = audit.Record(ctx, q, audit.Params{
		Action: "booking.terminate", EntityType: "booking", EntityID: r.BookingID, OwnerID: &r.OwnerID,
		Before: map[string]any{"status": "approved"}, After: map[string]any{"status": "terminated", "overdue_invoice": r.Number},
	})
	return err
}

func CompleteTenancies(ctx context.Context, deps Deps, logger *slog.Logger, today time.Time) {
	var ids []int64
	if err := inTx(ctx, deps, func(q *dbgen.Queries) (err error) {
		ids, err = q.SelectCompletableBookingIDs(ctx, pgDate(today))
		return err
	}); err != nil {
		logger.Error("complete_tenancies: select", "error", err)
		return
	}
	for _, id := range ids {
		if err := inTx(ctx, deps, func(q *dbgen.Queries) error { return completeTenancy(ctx, deps, q, id, today) }); err != nil {
			logger.Error("complete_tenancies: process", "booking_id", id, "error", err)
		}
	}
}

func completeTenancy(ctx context.Context, deps Deps, q *dbgen.Queries, id int64, today time.Time) error {
	b, err := q.LockBookingForCompletion(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if b.Status != "approved" || calendarDate(b.EndDate.Time).After(calendarDate(today)) {
		return nil
	}
	if err := q.CompleteBooking(ctx, b.ID); err != nil {
		return err
	}
	if b.StockHeld {
		if err := q.ReleaseRoomLock(ctx, b.RoomID); err != nil {
			return err
		}
	}
	if err := owner.NotifyUser(ctx, q, deps.Booking.AppPublicURL, b.TenantID, notification.CategoryBooking, "Masa sewa selesai",
		fmt.Sprintf("Masa sewa %s di %s sudah selesai. Bagikan pengalamanmu lewat ulasan.", b.RoomName, b.PropertyName),
		fmt.Sprintf("/booking/%d", b.ID)); err != nil {
		return err
	}
	if err := owner.NotifyUser(ctx, q, deps.Booking.AppPublicURL, b.OwnerID, notification.CategoryBooking, "Masa sewa selesai",
		fmt.Sprintf("Sewa %s di %s selesai. Jangan lupa kembalikan deposit %s dan catat di aplikasi.", b.TenantName, b.RoomName, money.FormatRupiah(b.Deposit)),
		fmt.Sprintf("/owner/booking/%d", b.ID)); err != nil {
		return err
	}
	_, err = audit.Record(ctx, q, audit.Params{
		Action: "booking.complete", EntityType: "booking", EntityID: b.ID, OwnerID: &b.OwnerID,
		Before: map[string]any{"status": "approved"}, After: map[string]any{"status": "completed"},
	})
	return err
}
