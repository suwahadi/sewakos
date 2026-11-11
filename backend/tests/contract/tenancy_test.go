package contract

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/suwahadi/sewakos/backend/internal/booking"
	"github.com/suwahadi/sewakos/backend/internal/midtrans"
	"github.com/suwahadi/sewakos/backend/internal/money"
	"github.com/suwahadi/sewakos/backend/internal/worker"
)

func wibDate(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.ParseInLocation(time.DateOnly, value, money.WIB)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

type tenancyFixture struct {
	bookingID, roomID, tenantID, ownerID int64
}

func insertApprovedMonthlyBooking(t *testing.T, pool *pgxpool.Pool, start string, duration int) tenancyFixture {
	t.Helper()
	ctx := context.Background()
	ownerID := seedOwner(t, pool)
	areaID := seedArea(t, pool, "Kota Sewa", "city", false, -6.2, 106.8)
	propertyID := seedProperty(t, pool, propertyFixture{ownerID: ownerID, areaID: areaID, name: "Kost Siklus", gender: "campur", lat: -6.2, lng: 106.8})
	roomID := seedRoom(t, pool, propertyID, "Kamar Siklus", 2000000, 1)
	var tenantID, bookingID int64
	if err := pool.QueryRow(ctx, `INSERT INTO users (name, role, phone, phone_verified_at) VALUES ('Penyewa Siklus', 'tenant', $1, now()) RETURNING id`,
		fmt.Sprintf("089%09d", time.Now().UnixNano()%1_000_000_000)).Scan(&tenantID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO bookings (tenant_id, room_id, rent_type, start_date, duration, end_date, status, period_price, price, deposit,
			admin_fee, expires_at, stock_held, paid_at)
		VALUES ($1, $2, 'monthly', $3::date, $4::int, ($3::date + make_interval(months => $4::int))::date, 'approved', 2000000, 2000000, 2000000,
			0, now(), true, now())
		RETURNING id`, tenantID, roomID, start, duration).Scan(&bookingID); err != nil {
		t.Fatalf("insert booking: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO invoices (booking_id, number, type, status, period_start, period_end, periods, rent_amount, deposit, admin_fee, total, due_date, issued_at, paid_at)
		VALUES ($1, $2, 'initial', 'paid', $3::date, ($3::date + interval '1 month')::date, 1, 2000000, 2000000, 0, 4000000, $3::date, now(), now())`,
		bookingID, fmt.Sprintf("INV/TEST/%d", bookingID), start); err != nil {
		t.Fatalf("insert initial invoice: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE rooms SET available = 0 WHERE id = $1`, roomID); err != nil {
		t.Fatal(err)
	}
	return tenancyFixture{bookingID: bookingID, roomID: roomID, tenantID: tenantID, ownerID: ownerID}
}

func renewalPeriods(t *testing.T, pool *pgxpool.Pool, bookingID int64) []string {
	t.Helper()
	rows, err := pool.Query(context.Background(),
		`SELECT period_start::text || '..' || period_end::text || ' due ' || due_date::text FROM invoices WHERE booking_id = $1 AND type = 'renewal' ORDER BY period_start`, bookingID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var periods []string
	for rows.Next() {
		var period string
		if err := rows.Scan(&period); err != nil {
			t.Fatal(err)
		}
		periods = append(periods, period)
	}
	return periods
}

func tenancyDeps(pool *pgxpool.Pool) worker.Deps {
	return worker.Deps{Booking: booking.Deps{DB: pool}, RenewalGraceDays: 3}
}

func TestIssueRenewalsFromHMinus3WithMonthEndClamping(t *testing.T) {
	_, pool := newTestServerAndPool(t)
	ctx := context.Background()
	f := insertApprovedMonthlyBooking(t, pool, "2025-01-31", 3)
	deps := tenancyDeps(pool)

	worker.IssueRenewals(ctx, deps, discardLogger(), wibDate(t, "2025-02-24"))
	if got := renewalPeriods(t, pool, f.bookingID); len(got) != 0 {
		t.Fatalf("renewals before H-3 = %v, want none", got)
	}
	worker.IssueRenewals(ctx, deps, discardLogger(), wibDate(t, "2025-02-25"))
	worker.IssueRenewals(ctx, deps, discardLogger(), wibDate(t, "2025-02-25"))
	if got := renewalPeriods(t, pool, f.bookingID); len(got) != 1 || got[0] != "2025-02-28..2025-03-31 due 2025-02-28" {
		t.Fatalf("renewals at H-3 = %v, want exactly the clamped second period", got)
	}
	var total, adminFee int64
	var number string
	if err := pool.QueryRow(ctx, `SELECT total, admin_fee, number FROM invoices WHERE booking_id = $1 AND type = 'renewal'`, f.bookingID).Scan(&total, &adminFee, &number); err != nil {
		t.Fatal(err)
	}
	if total != 2000000+adminFee || !strings.HasPrefix(number, "INV/") {
		t.Errorf("renewal total = %d (admin fee %d), number %q", total, adminFee, number)
	}
	if title, body := latestNotification(t, pool, f.tenantID); title != "Tagihan baru" || !strings.Contains(body, number) {
		t.Errorf("tenant notification = %q / %q", title, body)
	}

	worker.IssueRenewals(ctx, deps, discardLogger(), wibDate(t, "2025-03-28"))
	got := renewalPeriods(t, pool, f.bookingID)
	if len(got) != 2 || got[1] != "2025-03-31..2025-04-30 due 2025-03-31" {
		t.Errorf("renewals after third period H-3 = %v", got)
	}
}

func TestRenewalReminderAndTerminationAfterGrace(t *testing.T) {
	_, pool := newTestServerAndPool(t)
	ctx := context.Background()
	f := insertApprovedMonthlyBooking(t, pool, "2024-05-15", 3)
	deps := tenancyDeps(pool)
	worker.IssueRenewals(ctx, deps, discardLogger(), wibDate(t, "2024-06-12"))

	worker.RemindRenewalsDue(ctx, deps, discardLogger(), wibDate(t, "2024-06-15"))
	worker.RemindRenewalsDue(ctx, deps, discardLogger(), wibDate(t, "2024-06-15"))
	var reminders int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM user_notifications WHERE user_id = $1 AND title = 'Tagihan jatuh tempo hari ini' AND body LIKE '%18 Jun 2024%'`, f.tenantID).Scan(&reminders); err != nil {
		t.Fatal(err)
	}
	if reminders != 1 {
		t.Errorf("due-day reminders = %d, want 1 mentioning the last payable day", reminders)
	}

	worker.TerminateOverdue(ctx, deps, discardLogger(), wibDate(t, "2024-06-18"))
	var status string
	if err := pool.QueryRow(ctx, `SELECT status FROM bookings WHERE id = $1`, f.bookingID).Scan(&status); err != nil || status != "approved" {
		t.Fatalf("status on H+3 = %q (err %v), want still approved", status, err)
	}

	worker.TerminateOverdue(ctx, deps, discardLogger(), wibDate(t, "2024-06-19"))
	var stockHeld bool
	var invoiceStatus, voidReason string
	if err := pool.QueryRow(ctx, `SELECT b.status, b.stock_held, i.status, i.void_reason FROM bookings b JOIN invoices i ON i.booking_id = b.id AND i.type = 'renewal' WHERE b.id = $1`, f.bookingID).
		Scan(&status, &stockHeld, &invoiceStatus, &voidReason); err != nil {
		t.Fatal(err)
	}
	if status != "terminated" || stockHeld || invoiceStatus != "void" || voidReason != "overdue" {
		t.Errorf("after H+4: booking %s (held %v), invoice %s/%s", status, stockHeld, invoiceStatus, voidReason)
	}
	if got := roomAvailable(t, pool, f.roomID); got != 1 {
		t.Errorf("room available = %d, want 1 after release", got)
	}
	if title, _ := latestNotification(t, pool, f.ownerID); title != "Sewa penyewa berakhir" {
		t.Errorf("owner notification = %q", title)
	}

	var invoiceID, total int64
	if err := pool.QueryRow(ctx, `SELECT id, total FROM invoices WHERE booking_id = $1 AND type = 'renewal'`, f.bookingID).Scan(&invoiceID, &total); err != nil {
		t.Fatal(err)
	}
	orderID := fmt.Sprintf("KOS-VOID-%d", invoiceID)
	if _, err := pool.Exec(ctx, `INSERT INTO payments (invoice_id, order_id, gross_amount, expires_at) VALUES ($1, $2, $3, now() + interval '15 minutes')`, invoiceID, orderID, total); err != nil {
		t.Fatal(err)
	}
	if err := booking.ApplyPaymentStatus(ctx, booking.Deps{DB: pool}, orderID, midtrans.StatusResponse{
		OrderID: orderID, TransactionStatus: "settlement", GrossAmount: fmt.Sprintf("%d.00", total),
	}, []byte(`{}`)); err != nil {
		t.Fatalf("apply settlement on void invoice: %v", err)
	}
	var refundAmount int64
	var refundReason string
	if err := pool.QueryRow(ctx, `SELECT amount, reason FROM refunds WHERE invoice_id = $1`, invoiceID).Scan(&refundAmount, &refundReason); err != nil {
		t.Fatalf("load refund: %v", err)
	}
	if refundAmount != total || refundReason != "invoice_void" {
		t.Errorf("refund = %d %s, want %d invoice_void", refundAmount, refundReason, total)
	}
}

func TestCompleteTenanciesReleasesTheRoom(t *testing.T) {
	_, pool := newTestServerAndPool(t)
	ctx := context.Background()
	f := insertApprovedMonthlyBooking(t, pool, "2024-01-10", 1)
	deps := tenancyDeps(pool)

	worker.CompleteTenancies(ctx, deps, discardLogger(), wibDate(t, "2024-02-09"))
	var status string
	if err := pool.QueryRow(ctx, `SELECT status FROM bookings WHERE id = $1`, f.bookingID).Scan(&status); err != nil || status != "approved" {
		t.Fatalf("status before end_date = %q (err %v)", status, err)
	}
	worker.CompleteTenancies(ctx, deps, discardLogger(), wibDate(t, "2024-02-10"))
	if err := pool.QueryRow(ctx, `SELECT status FROM bookings WHERE id = $1`, f.bookingID).Scan(&status); err != nil || status != "completed" {
		t.Fatalf("status on end_date = %q (err %v), want completed", status, err)
	}
	if got := roomAvailable(t, pool, f.roomID); got != 1 {
		t.Errorf("room available = %d, want 1", got)
	}
	if title, body := latestNotification(t, pool, f.ownerID); title != "Masa sewa selesai" || !strings.Contains(body, "kembalikan deposit Rp2.000.000") {
		t.Errorf("owner notification = %q / %q", title, body)
	}
}
