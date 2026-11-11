package contract

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/suwahadi/sewakos/backend/internal/midtrans"
	"github.com/suwahadi/sewakos/backend/internal/money"
	"github.com/suwahadi/sewakos/backend/internal/worker"
)

const jobPassesForSharedDatabase = 20

func repeatJob(job func()) {
	for range jobPassesForSharedDatabase {
		job()
	}
}

func settlingMidtransMock() (client *midtrans.MockClient, expectGross func(orderID string, gross int64)) {
	var mu sync.Mutex
	grossByOrder := map[string]int64{}
	expectGross = func(orderID string, gross int64) {
		mu.Lock()
		defer mu.Unlock()
		grossByOrder[orderID] = gross
	}
	return &midtrans.MockClient{
		CreateTransactionFunc: func(_ context.Context, req midtrans.TransactionRequest) (midtrans.TransactionResponse, error) {
			mu.Lock()
			defer mu.Unlock()
			grossByOrder[req.TransactionDetails.OrderID] = req.TransactionDetails.GrossAmount
			return midtrans.TransactionResponse{Token: "snap-token", RedirectURL: "https://example.test/redirect"}, nil
		},
		GetStatusFunc: func(_ context.Context, orderID string) (midtrans.StatusResponse, error) {
			mu.Lock()
			defer mu.Unlock()
			return midtrans.StatusResponse{
				OrderID: orderID, TransactionStatus: "settlement", FraudStatus: "accept",
				GrossAmount: grossAmountString(grossByOrder[orderID]), PaymentType: "bank_transfer",
			}, nil
		},
		ExpireTransactionFunc: func(context.Context, string) error { return nil },
	}, expectGross
}

func insertApprovedDailyBooking(t *testing.T, pool *pgxpool.Pool, start time.Time, duration int) tenancyFixture {
	t.Helper()
	ctx := context.Background()
	ownerID := seedOwner(t, pool)
	areaID := seedArea(t, pool, "Kota Harian", "city", false, -6.2, 106.8)
	propertyID := seedProperty(t, pool, propertyFixture{ownerID: ownerID, areaID: areaID, name: "Kost Harian", gender: "campur", lat: -6.2, lng: 106.8})
	roomID := seedRoom(t, pool, propertyID, "Kamar Harian", 2000000, 1)
	var tenantID, bookingID int64
	if err := pool.QueryRow(ctx, `INSERT INTO users (name, role, phone, phone_verified_at) VALUES ('Penyewa Harian', 'tenant', $1, now()) RETURNING id`,
		fmt.Sprintf("088%09d", time.Now().UnixNano()%1_000_000_000)).Scan(&tenantID); err != nil {
		t.Fatal(err)
	}
	startDate := start.Format(time.DateOnly)
	if err := pool.QueryRow(ctx, `
		INSERT INTO bookings (tenant_id, room_id, rent_type, start_date, duration, end_date, status, period_price, price, deposit,
			admin_fee, expires_at, stock_held, paid_at)
		VALUES ($1, $2, 'daily', $3::date, $4::int, $3::date + $4::int, 'approved', 150000, 150000, 150000, 0, now(), true, now())
		RETURNING id`, tenantID, roomID, startDate, duration).Scan(&bookingID); err != nil {
		t.Fatalf("insert daily booking: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO invoices (booking_id, number, type, status, period_start, period_end, periods, rent_amount, deposit, admin_fee, total, due_date, issued_at, paid_at)
		VALUES ($1, $2, 'initial', 'paid', $3::date, $3::date + $4::int, $4::int, 150000 * $4::int, 150000, 0, 150000 * $4::int + 150000, $3::date, now(), now())`,
		bookingID, fmt.Sprintf("INV/TEST/D/%d", bookingID), startDate, duration); err != nil {
		t.Fatalf("insert initial invoice: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE rooms SET available = 0 WHERE id = $1`, roomID); err != nil {
		t.Fatal(err)
	}
	return tenancyFixture{bookingID: bookingID, roomID: roomID, tenantID: tenantID, ownerID: ownerID}
}

type extensionInvoice struct {
	ID          int64  `json:"id"`
	Type        string `json:"type"`
	Status      string `json:"status"`
	PeriodStart string `json:"period_start"`
	PeriodEnd   string `json:"period_end"`
	DueDate     string `json:"due_date"`
	RentAmount  int64  `json:"rent_amount"`
	AdminFee    int64  `json:"admin_fee"`
	Total       int64  `json:"total"`
}

func requestExtension(t *testing.T, srv *httptest.Server, token string, bookingID int64, duration int) extensionInvoice {
	t.Helper()
	resp, body := bookingAuthedRequest(t, srv, http.MethodPost, fmt.Sprintf("/api/booking/%d/extend", bookingID), token, fmt.Sprintf(`{"duration":%d}`, duration))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("extend: status = %d, body = %s", resp.StatusCode, body)
	}
	var parsed struct {
		Invoice extensionInvoice `json:"invoice"`
	}
	mustUnmarshal(t, body, &parsed)
	return parsed.Invoice
}

func checkoutInvoiceOrderID(t *testing.T, srv *httptest.Server, token string, invoiceID int64) (orderID string, grossAmount int64) {
	t.Helper()
	resp, body := invoiceCheckout(t, srv, token, invoiceID)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("invoice checkout: status = %d, body = %s", resp.StatusCode, body)
	}
	var parsed struct {
		Payment struct {
			OrderID     string `json:"order_id"`
			GrossAmount int64  `json:"gross_amount"`
		} `json:"payment"`
	}
	mustUnmarshal(t, body, &parsed)
	return parsed.Payment.OrderID, parsed.Payment.GrossAmount
}

func settleOrder(t *testing.T, srv *httptest.Server, orderID string, grossAmount int64) {
	t.Helper()
	if resp, body := postWebhookNotification(t, srv, orderID, "200", grossAmountString(grossAmount)); resp.StatusCode != http.StatusOK {
		t.Fatalf("webhook: status = %d, body = %s", resp.StatusCode, body)
	}
}

type bookingTerm struct {
	status   string
	duration int
	endDate  string
}

func loadBookingTerm(t *testing.T, pool *pgxpool.Pool, bookingID int64) bookingTerm {
	t.Helper()
	var term bookingTerm
	if err := pool.QueryRow(context.Background(), `SELECT status, duration, end_date::text FROM bookings WHERE id = $1`, bookingID).
		Scan(&term.status, &term.duration, &term.endDate); err != nil {
		t.Fatalf("load booking: %v", err)
	}
	return term
}

func invoiceStatus(t *testing.T, pool *pgxpool.Pool, invoiceID int64) string {
	t.Helper()
	var status string
	if err := pool.QueryRow(context.Background(), `SELECT status FROM invoices WHERE id = $1`, invoiceID).Scan(&status); err != nil {
		t.Fatalf("load invoice: %v", err)
	}
	return status
}

func TestMonthlyExtensionCycleWithAdvancedClock(t *testing.T) {
	t.Setenv("MIDTRANS_SERVER_KEY", webhookTestServerKey)
	mock, _ := settlingMidtransMock()
	srv, pool := newTestServerAndPoolWithMidtrans(t, mock)
	ctx := context.Background()
	start := time.Date(money.TodayWIB().Year()+1, time.January, 31, 0, 0, 0, 0, money.WIB)
	f := insertApprovedMonthlyBooking(t, pool, start.Format(time.DateOnly), 1)
	if _, err := pool.Exec(ctx, `UPDATE rooms SET monthly_duration_options = '[1,3]' WHERE id = $1`, f.roomID); err != nil {
		t.Fatal(err)
	}
	token := mintTenantToken(t, pool, f.tenantID)
	deps := tenancyDeps(pool)
	originalEnd := money.AddPeriod(start, "monthly", 1)
	extendedEnd := money.AddPeriod(start, "monthly", 4)
	if extendedEnd.Day() != 31 || extendedEnd.Month() != time.May {
		t.Fatalf("extended end = %s, want 31 May (clamped from the original start day, not chained from February)", extendedEnd.Format(time.DateOnly))
	}

	invoice := requestExtension(t, srv, token, f.bookingID, 3)
	if invoice.Type != "extension" || invoice.PeriodStart != originalEnd.Format(time.DateOnly) || invoice.PeriodEnd != extendedEnd.Format(time.DateOnly) ||
		invoice.DueDate != originalEnd.AddDate(0, 0, -1).Format(time.DateOnly) || invoice.RentAmount != 2000000 || invoice.Total != 2000000+invoice.AdminFee {
		t.Fatalf("extension invoice = %+v", invoice)
	}

	repeatJob(func() { worker.VoidExtensions(ctx, deps, discardLogger(), originalEnd.AddDate(0, 0, -1)) })
	if got := invoiceStatus(t, pool, invoice.ID); got != "unpaid" {
		t.Fatalf("invoice the day before end_date = %q, want unpaid", got)
	}

	orderID, gross := checkoutInvoiceOrderID(t, srv, token, invoice.ID)
	settleOrder(t, srv, orderID, gross)
	if term := loadBookingTerm(t, pool, f.bookingID); term.status != "approved" || term.duration != 4 || term.endDate != extendedEnd.Format(time.DateOnly) {
		t.Fatalf("booking after paid extension = %+v, want approved/4/%s", term, extendedEnd.Format(time.DateOnly))
	}
	var payoutAmount int64
	if err := pool.QueryRow(ctx, `SELECT amount FROM payouts WHERE invoice_id = $1`, invoice.ID).Scan(&payoutAmount); err != nil || payoutAmount != 2000000 {
		t.Errorf("extension payout = %d (err %v), want 2000000", payoutAmount, err)
	}

	repeatJob(func() { worker.VoidExtensions(ctx, deps, discardLogger(), originalEnd) })
	repeatJob(func() { worker.CompleteTenancies(ctx, deps, discardLogger(), originalEnd) })
	if term := loadBookingTerm(t, pool, f.bookingID); term.status != "approved" {
		t.Fatalf("status on the original end_date = %q, want approved (paid extension keeps the tenancy running)", term.status)
	}
	if got := invoiceStatus(t, pool, invoice.ID); got != "paid" {
		t.Errorf("paid extension invoice = %q after void_extensions", got)
	}

	thirdPeriod := money.AddPeriod(start, "monthly", 2)
	fourthPeriod := money.AddPeriod(start, "monthly", 3)
	repeatJob(func() { worker.IssueRenewals(ctx, deps, discardLogger(), thirdPeriod.AddDate(0, 0, -4)) })
	if got := renewalPeriods(t, pool, f.bookingID); len(got) != 0 {
		t.Fatalf("renewals before H-3 of the third period = %v, want none (the extension invoice covers the second period)", got)
	}
	repeatJob(func() { worker.IssueRenewals(ctx, deps, discardLogger(), thirdPeriod.AddDate(0, 0, -3)) })
	repeatJob(func() { worker.IssueRenewals(ctx, deps, discardLogger(), fourthPeriod.AddDate(0, 0, -3)) })
	wantRenewals := []string{
		fmt.Sprintf("%s..%s due %s", thirdPeriod.Format(time.DateOnly), fourthPeriod.Format(time.DateOnly), thirdPeriod.Format(time.DateOnly)),
		fmt.Sprintf("%s..%s due %s", fourthPeriod.Format(time.DateOnly), extendedEnd.Format(time.DateOnly), fourthPeriod.Format(time.DateOnly)),
	}
	if got := renewalPeriods(t, pool, f.bookingID); !slices.Equal(got, wantRenewals) {
		t.Fatalf("renewals = %v, want %v", got, wantRenewals)
	}

	repeatJob(func() { worker.CompleteTenancies(ctx, deps, discardLogger(), extendedEnd.AddDate(0, 0, -1)) })
	if term := loadBookingTerm(t, pool, f.bookingID); term.status != "approved" {
		t.Fatalf("status the day before the extended end_date = %q, want approved", term.status)
	}
	repeatJob(func() { worker.CompleteTenancies(ctx, deps, discardLogger(), extendedEnd) })
	if term := loadBookingTerm(t, pool, f.bookingID); term.status != "completed" {
		t.Fatalf("status on the extended end_date = %q, want completed", term.status)
	}
	if got := roomAvailable(t, pool, f.roomID); got != 1 {
		t.Errorf("room available = %d, want 1 after completion", got)
	}
}

func TestDailyExtensionCycleWithAdvancedClock(t *testing.T) {
	t.Setenv("MIDTRANS_SERVER_KEY", webhookTestServerKey)
	mock, _ := settlingMidtransMock()
	srv, pool := newTestServerAndPoolWithMidtrans(t, mock)
	ctx := context.Background()
	start := money.TodayWIB()
	f := insertApprovedDailyBooking(t, pool, start, 2)
	token := mintTenantToken(t, pool, f.tenantID)
	deps := tenancyDeps(pool)
	originalEnd := start.AddDate(0, 0, 2)
	extendedEnd := start.AddDate(0, 0, 5)

	invoice := requestExtension(t, srv, token, f.bookingID, 3)
	if invoice.PeriodStart != originalEnd.Format(time.DateOnly) || invoice.PeriodEnd != extendedEnd.Format(time.DateOnly) ||
		invoice.DueDate != originalEnd.AddDate(0, 0, -1).Format(time.DateOnly) || invoice.RentAmount != 450000 || invoice.Total != 450000+invoice.AdminFee {
		t.Fatalf("extension invoice = %+v, want the whole added duration billed up front", invoice)
	}

	orderID, gross := checkoutInvoiceOrderID(t, srv, token, invoice.ID)
	if gross != invoice.Total {
		t.Errorf("checkout gross_amount = %d, want %d", gross, invoice.Total)
	}
	settleOrder(t, srv, orderID, gross)
	if term := loadBookingTerm(t, pool, f.bookingID); term.status != "approved" || term.duration != 5 || term.endDate != extendedEnd.Format(time.DateOnly) {
		t.Fatalf("booking after paid extension = %+v, want approved/5/%s", term, extendedEnd.Format(time.DateOnly))
	}

	repeatJob(func() { worker.VoidExtensions(ctx, deps, discardLogger(), originalEnd) })
	repeatJob(func() { worker.CompleteTenancies(ctx, deps, discardLogger(), originalEnd) })
	if term := loadBookingTerm(t, pool, f.bookingID); term.status != "approved" {
		t.Fatalf("status on the original end_date = %q, want approved", term.status)
	}
	repeatJob(func() { worker.CompleteTenancies(ctx, deps, discardLogger(), extendedEnd) })
	if term := loadBookingTerm(t, pool, f.bookingID); term.status != "completed" {
		t.Fatalf("status on the extended end_date = %q, want completed", term.status)
	}
	if got := roomAvailable(t, pool, f.roomID); got != 1 {
		t.Errorf("room available = %d, want 1 after completion", got)
	}
}

func TestUnpaidExtensionIsVoidedAtEndDateAndLatePaymentIsRefunded(t *testing.T) {
	t.Setenv("MIDTRANS_SERVER_KEY", webhookTestServerKey)
	mock, expectGross := settlingMidtransMock()
	srv, pool := newTestServerAndPoolWithMidtrans(t, mock)
	ctx := context.Background()
	start := money.TodayWIB()
	f := insertApprovedDailyBooking(t, pool, start, 2)
	token := mintTenantToken(t, pool, f.tenantID)
	deps := tenancyDeps(pool)
	originalEnd := start.AddDate(0, 0, 2)

	invoice := requestExtension(t, srv, token, f.bookingID, 3)
	orderID, _ := checkoutInvoiceOrderID(t, srv, token, invoice.ID)

	repeatJob(func() { worker.VoidExtensions(ctx, deps, discardLogger(), originalEnd.AddDate(0, 0, -1)) })
	if got := invoiceStatus(t, pool, invoice.ID); got != "unpaid" {
		t.Fatalf("invoice the day before end_date = %q, want unpaid", got)
	}
	repeatJob(func() { worker.VoidExtensions(ctx, deps, discardLogger(), originalEnd) })
	repeatJob(func() { worker.CompleteTenancies(ctx, deps, discardLogger(), originalEnd) })
	if got := invoiceStatus(t, pool, invoice.ID); got != "void" {
		t.Fatalf("invoice on end_date = %q, want void", got)
	}
	var paymentStatus string
	if err := pool.QueryRow(ctx, `SELECT status FROM payments WHERE order_id = $1`, orderID).Scan(&paymentStatus); err != nil || paymentStatus != "expire" {
		t.Errorf("pending payment = %q (err %v), want expire", paymentStatus, err)
	}
	if term := loadBookingTerm(t, pool, f.bookingID); term.status != "completed" || term.duration != 2 || term.endDate != originalEnd.Format(time.DateOnly) {
		t.Fatalf("booking = %+v, want completed with the original term", term)
	}

	resp, body := bookingAuthedRequest(t, srv, http.MethodPost, fmt.Sprintf("/api/booking/%d/extend", f.bookingID), token, `{"duration":1}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("extend after completion: status = %d, body = %s, want 400", resp.StatusCode, body)
	}

	lateOrderID := fmt.Sprintf("KOS-LATE-EXT-%d", invoice.ID)
	if _, err := pool.Exec(ctx, `INSERT INTO payments (invoice_id, order_id, gross_amount, expires_at) VALUES ($1, $2, $3, now() + interval '15 minutes')`,
		invoice.ID, lateOrderID, invoice.Total); err != nil {
		t.Fatal(err)
	}
	expectGross(lateOrderID, invoice.Total)
	settleOrder(t, srv, lateOrderID, invoice.Total)
	var refundAmount int64
	var refundReason string
	if err := pool.QueryRow(ctx, `SELECT amount, reason FROM refunds WHERE invoice_id = $1`, invoice.ID).Scan(&refundAmount, &refundReason); err != nil {
		t.Fatalf("load refund: %v", err)
	}
	if refundAmount != invoice.Total || refundReason != "invoice_void" {
		t.Errorf("refund = %d %s, want %d invoice_void", refundAmount, refundReason, invoice.Total)
	}
	if term := loadBookingTerm(t, pool, f.bookingID); term.duration != 2 {
		t.Errorf("duration after late payment = %d, want 2 (a void extension never extends the tenancy)", term.duration)
	}
}
