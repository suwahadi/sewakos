package contract

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/suwahadi/sewakos/backend/internal/booking"
	"github.com/suwahadi/sewakos/backend/internal/worker"
)

func newOwnerBookingFixture(t *testing.T, srv *httptest.Server, pool *pgxpool.Pool) (ownerUserID int64, ownerToken string, roomID int64, tenantToken string, tenantID int64, tenantPhone string) {
	t.Helper()
	ownerUserID, ownerToken = newOwner(t, srv, pool)
	propertyID := newOwnerProperty(t, pool, ownerUserID)
	roomID = seedRoom(t, pool, propertyID, "Kamar Tipe A", 1500000, 3)
	tenantPhone, tenantToken = bookingRegisterAndVerify(t, srv)
	if err := pool.QueryRow(context.Background(), `SELECT id FROM users WHERE phone = $1`, tenantPhone).Scan(&tenantID); err != nil {
		t.Fatal(err)
	}
	return ownerUserID, ownerToken, roomID, tenantToken, tenantID, tenantPhone
}

func markBookingPaid(t *testing.T, pool *pgxpool.Pool, bookingID, roomID int64, deadline time.Duration) {
	t.Helper()
	ctx := context.Background()
	for _, stmt := range []string{
		`UPDATE bookings SET status = 'paid', paid_at = now(), owner_deadline_at = now() + make_interval(secs => $2), stock_held = true, admin_fee = 5000 WHERE id = $1`,
		`UPDATE invoices SET status = 'paid', paid_at = now(), admin_fee = 5000, total = rent_amount - discount + deposit + 5000 WHERE booking_id = $1 AND type = 'initial'`,
	} {
		args := []any{bookingID}
		if strings.Contains(stmt, "$2") {
			args = append(args, deadline.Seconds())
		}
		if _, err := pool.Exec(ctx, stmt, args...); err != nil {
			t.Fatalf("mark booking paid: %v", err)
		}
	}
	if _, err := pool.Exec(ctx, `UPDATE rooms SET available = available - 1 WHERE id = $1`, roomID); err != nil {
		t.Fatal(err)
	}
}

func roomAvailable(t *testing.T, pool *pgxpool.Pool, roomID int64) int {
	t.Helper()
	var available int
	if err := pool.QueryRow(context.Background(), `SELECT available FROM rooms WHERE id = $1`, roomID).Scan(&available); err != nil {
		t.Fatal(err)
	}
	return available
}

func latestNotification(t *testing.T, pool *pgxpool.Pool, userID int64) (title, body string) {
	t.Helper()
	if err := pool.QueryRow(context.Background(), `SELECT title, body FROM user_notifications WHERE user_id = $1 ORDER BY id DESC LIMIT 1`, userID).Scan(&title, &body); err != nil {
		t.Fatalf("latest notification: %v", err)
	}
	return title, body
}

func TestOwnerBookingApproveFixture(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv, pool := newTestServerAndPool(t)
	ownerID, ownerToken, roomID, tenantToken, tenantID, tenantPhone := newOwnerBookingFixture(t, srv, pool)
	bookingID := submitBookingFixture(t, srv, tenantToken, roomID)
	markBookingPaid(t, pool, bookingID, roomID, 24*time.Hour)
	ctx := context.Background()

	resp, body := bookingAuthedRequest(t, srv, http.MethodGet, "/api/owner/bookings", ownerToken, "")
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), `"phone_number":null`) {
		t.Fatalf("list before approve: status = %d, body = %s", resp.StatusCode, body)
	}

	resp, body = bookingAuthedRequest(t, srv, http.MethodPost, fmt.Sprintf("/api/owner/bookings/%d/approve", bookingID), ownerToken, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("approve: status = %d, body = %s", resp.StatusCode, body)
	}
	startDate := time.Now().AddDate(0, 0, 1)
	want := fmt.Sprintf(`{
		"status": true,
		"meta": {"response_code": 200, "code": 200, "severity": "OK", "message": "Booking disetujui."},
		"booking": {
			"id": %d, "status": "approved", "rent_type": "monthly", "start_date": %q,
			"duration": 1, "end_date": %q,
			"tenant": {"id": %d, "name": "Test User", "phone_number": %q},
			"property": {"id": 0, "name": "Kost Foto"},
			"room": {"id": %d, "name": "Kamar Tipe A"},
			"price": 1500000, "price_label": "Rp1.500.000",
			"deposit": 1500000, "deposit_label": "Rp1.500.000",
			"payout_amount": 3000000, "payout_amount_label": "Rp3.000.000",
			"paid_at": "", "owner_deadline_at": "",
			"reject_reason": null, "deposit_returned_at": null,
			"created_at": ""
		}
	}`, bookingID, startDate.Format("2006-01-02"), startDate.AddDate(0, 1, 0).Format("2006-01-02"), tenantID, tenantPhone, roomID)
	timestamp := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\+00:00$`)
	AssertJSONMatches(t, body, []byte(want), map[string]*regexp.Regexp{
		"$.booking.property.id":       regexp.MustCompile(`^\d+$`),
		"$.booking.tenant.name":       regexp.MustCompile(`.+`),
		"$.booking.paid_at":           timestamp,
		"$.booking.owner_deadline_at": timestamp,
		"$.booking.created_at":        regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{6}Z$`),
	})

	var payoutAmount int64
	var payoutStatus string
	if err := pool.QueryRow(ctx, `SELECT amount, status FROM payouts WHERE booking_id = $1`, bookingID).Scan(&payoutAmount, &payoutStatus); err != nil {
		t.Fatalf("load payout: %v", err)
	}
	if payoutAmount != 3000000 || payoutStatus != "pending" {
		t.Errorf("payout = %d %s, want 3000000 pending", payoutAmount, payoutStatus)
	}
	var earning int64
	if err := pool.QueryRow(ctx, `SELECT amount FROM ledger_entries WHERE booking_id = $1 AND type = 'owner_earning' AND owner_id = $2`, bookingID, ownerID).Scan(&earning); err != nil || earning != 3000000 {
		t.Errorf("owner_earning ledger = %d (err %v), want 3000000", earning, err)
	}
	if title, body := latestNotification(t, pool, tenantID); title != "Booking disetujui" || !strings.Contains(body, "Mulai sewa") {
		t.Errorf("tenant notification = %q / %q", title, body)
	}

	resp, body = bookingAuthedRequest(t, srv, http.MethodPost, fmt.Sprintf("/api/owner/bookings/%d/approve", bookingID), ownerToken, "")
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(string(body), "Booking tidak dapat diproses.") {
		t.Errorf("second approve: status = %d, body = %s", resp.StatusCode, body)
	}
}

func TestOwnerBookingReject(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv, pool := newTestServerAndPool(t)
	_, ownerToken, roomID, tenantToken, tenantID, _ := newOwnerBookingFixture(t, srv, pool)
	bookingID := submitBookingFixture(t, srv, tenantToken, roomID)
	markBookingPaid(t, pool, bookingID, roomID, 24*time.Hour)
	ctx := context.Background()
	rejectPath := fmt.Sprintf("/api/owner/bookings/%d/reject", bookingID)

	resp, body := bookingAuthedRequest(t, srv, http.MethodPost, rejectPath, ownerToken, fmt.Sprintf(`{"reason":%q}`, strings.Repeat("x", 501)))
	if resp.StatusCode != http.StatusUnprocessableEntity || !strings.Contains(string(body), "The reason field must not be greater than 500 characters.") {
		t.Fatalf("long reason: status = %d, body = %s", resp.StatusCode, body)
	}

	availableBefore := roomAvailable(t, pool, roomID)
	resp, body = bookingAuthedRequest(t, srv, http.MethodPost, rejectPath, ownerToken, `{"reason":"Kamar sedang renovasi."}`)
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), `"Booking ditolak. Dana akan dikembalikan ke penyewa."`) ||
		!strings.Contains(string(body), `"reject_reason":"Kamar sedang renovasi."`) || !strings.Contains(string(body), `"status":"rejected"`) {
		t.Fatalf("reject: status = %d, body = %s", resp.StatusCode, body)
	}
	if got := roomAvailable(t, pool, roomID); got != availableBefore+1 {
		t.Errorf("room available = %d, want %d after releasing the lock", got, availableBefore+1)
	}

	var refundAmount, invoiceTotal int64
	var refundReason, refundStatus string
	if err := pool.QueryRow(ctx, `SELECT r.amount, r.reason, r.status, i.total FROM refunds r JOIN invoices i ON i.id = r.invoice_id WHERE r.booking_id = $1`, bookingID).
		Scan(&refundAmount, &refundReason, &refundStatus, &invoiceTotal); err != nil {
		t.Fatalf("load refund: %v", err)
	}
	if refundAmount != invoiceTotal || refundReason != "owner_rejected" || refundStatus != "awaiting_account" {
		t.Errorf("refund = %d %s %s, want %d owner_rejected awaiting_account", refundAmount, refundReason, refundStatus, invoiceTotal)
	}
	var reversed int64
	if err := pool.QueryRow(ctx, `SELECT amount FROM ledger_entries WHERE booking_id = $1 AND type = 'admin_fee_reversed'`, bookingID).Scan(&reversed); err != nil || reversed != -5000 {
		t.Errorf("admin_fee_reversed = %d (err %v), want -5000", reversed, err)
	}
	if title, body := latestNotification(t, pool, tenantID); title != "Booking ditolak" || !strings.HasSuffix(body, " Alasan: Kamar sedang renovasi.") {
		t.Errorf("tenant notification = %q / %q", title, body)
	}
}

func TestOwnerBookingVisibilityAndDeadline(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv, pool := newTestServerAndPool(t)
	_, ownerToken, roomID, tenantToken, _, _ := newOwnerBookingFixture(t, srv, pool)
	unpaidID := submitBookingFixture(t, srv, tenantToken, roomID)
	soonID := submitBookingFixture(t, srv, tenantToken, roomID)
	laterID := submitBookingFixture(t, srv, tenantToken, roomID)
	markBookingPaid(t, pool, laterID, roomID, 20*time.Hour)
	markBookingPaid(t, pool, soonID, roomID, 2*time.Hour)

	resp, body := bookingAuthedRequest(t, srv, http.MethodGet, "/api/owner/bookings?status=paid", ownerToken, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list: status = %d, body = %s", resp.StatusCode, body)
	}
	var list struct {
		Bookings []struct {
			ID int64 `json:"id"`
		} `json:"bookings"`
	}
	mustUnmarshal(t, body, &list)
	if len(list.Bookings) != 2 || list.Bookings[0].ID != soonID || list.Bookings[1].ID != laterID {
		t.Errorf("paid bookings = %s, want [%d, %d] (nearest deadline first)", body, soonID, laterID)
	}
	if resp, _ := bookingAuthedRequest(t, srv, http.MethodGet, fmt.Sprintf("/api/owner/bookings/%d", unpaidID), ownerToken, ""); resp.StatusCode != http.StatusNotFound {
		t.Errorf("unpaid booking detail: status = %d, want 404", resp.StatusCode)
	}

	_, otherOwnerToken := newOwner(t, srv, pool)
	if resp, _ := bookingAuthedRequest(t, srv, http.MethodPost, fmt.Sprintf("/api/owner/bookings/%d/approve", soonID), otherOwnerToken, ""); resp.StatusCode != http.StatusNotFound {
		t.Errorf("other owner approve: status = %d, want 404", resp.StatusCode)
	}

	if _, err := pool.Exec(context.Background(), `UPDATE bookings SET owner_deadline_at = now() - interval '1 minute' WHERE id = $1`, soonID); err != nil {
		t.Fatal(err)
	}
	resp, body = bookingAuthedRequest(t, srv, http.MethodPost, fmt.Sprintf("/api/owner/bookings/%d/approve", soonID), ownerToken, "")
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(string(body), "Booking tidak dapat diproses.") {
		t.Errorf("approve after deadline: status = %d, body = %s", resp.StatusCode, body)
	}
}

func TestOwnerBookingDepositReturned(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv, pool := newTestServerAndPool(t)
	_, ownerToken, roomID, tenantToken, _, _ := newOwnerBookingFixture(t, srv, pool)
	bookingID := submitBookingFixture(t, srv, tenantToken, roomID)
	markBookingPaid(t, pool, bookingID, roomID, 24*time.Hour)
	ctx := context.Background()
	depositPath := fmt.Sprintf("/api/owner/bookings/%d/deposit-returned", bookingID)

	resp, body := bookingAuthedRequest(t, srv, http.MethodPost, depositPath, ownerToken, "")
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(string(body), "Pengembalian deposit tidak dapat dicatat.") {
		t.Fatalf("before completed: status = %d, body = %s", resp.StatusCode, body)
	}

	if _, err := pool.Exec(ctx, `UPDATE bookings SET status = 'completed', ended_at = now() WHERE id = $1`, bookingID); err != nil {
		t.Fatal(err)
	}

	resp, body = bookingAuthedRequest(t, srv, http.MethodPost, depositPath, ownerToken, `{"note":"Transfer BCA 1 Mar 2027"}`)
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "Pengembalian deposit dicatat.") ||
		!strings.Contains(string(body), `"deposit_returned_at":"`) {
		t.Fatalf("deposit-returned: status = %d, body = %s", resp.StatusCode, body)
	}

	var amount int64
	var description string
	if err := pool.QueryRow(ctx, `SELECT amount, description FROM ledger_entries WHERE booking_id = $1 AND type = 'deposit_returned'`, bookingID).
		Scan(&amount, &description); err != nil {
		t.Fatalf("load ledger entry: %v", err)
	}
	if amount != 1500000 {
		t.Errorf("deposit_returned ledger amount = %d, want 1500000", amount)
	}

	resp, body = bookingAuthedRequest(t, srv, http.MethodPost, depositPath, ownerToken, "")
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(string(body), "Pengembalian deposit tidak dapat dicatat.") {
		t.Errorf("second deposit-returned: status = %d, body = %s", resp.StatusCode, body)
	}

	_, otherOwnerToken := newOwner(t, srv, pool)
	if resp, _ := bookingAuthedRequest(t, srv, http.MethodPost, depositPath, otherOwnerToken, ""); resp.StatusCode != http.StatusNotFound {
		t.Errorf("other owner: status = %d, want 404", resp.StatusCode)
	}
}

func TestOwnerTimeoutAndReminderJobs(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv, pool := newTestServerAndPool(t)
	ownerID, _, roomID, tenantToken, tenantID, _ := newOwnerBookingFixture(t, srv, pool)
	expiredID := submitBookingFixture(t, srv, tenantToken, roomID)
	reminderID := submitBookingFixture(t, srv, tenantToken, roomID)
	markBookingPaid(t, pool, expiredID, roomID, -time.Minute)
	markBookingPaid(t, pool, reminderID, roomID, time.Hour)
	ctx := context.Background()
	deps := worker.Deps{Booking: booking.Deps{DB: pool}, OwnerReminderBeforeMinutes: 120}

	availableBefore := roomAvailable(t, pool, roomID)
	worker.OwnerTimeout(ctx, deps, discardLogger())
	var status, rejectedBy, refundReason string
	if err := pool.QueryRow(ctx, `SELECT b.status, b.rejected_by, r.reason FROM bookings b JOIN refunds r ON r.booking_id = b.id WHERE b.id = $1`, expiredID).
		Scan(&status, &rejectedBy, &refundReason); err != nil {
		t.Fatalf("load timed-out booking: %v", err)
	}
	if status != "rejected" || rejectedBy != "system" || refundReason != "owner_timeout" {
		t.Errorf("timed-out booking = %s/%s/%s", status, rejectedBy, refundReason)
	}
	if got := roomAvailable(t, pool, roomID); got != availableBefore+1 {
		t.Errorf("room available = %d, want %d", got, availableBefore+1)
	}
	if title, _ := latestNotification(t, pool, tenantID); title != "Booking ditolak otomatis" {
		t.Errorf("tenant notification = %q", title)
	}

	worker.OwnerReminder(ctx, deps, discardLogger())
	worker.OwnerReminder(ctx, deps, discardLogger())
	var reminders int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM user_notifications WHERE user_id = $1 AND title = 'Segera respons booking' AND body LIKE $2`,
		ownerID, fmt.Sprintf("Booking #%d %%", reminderID)).Scan(&reminders); err != nil {
		t.Fatal(err)
	}
	if reminders != 1 {
		t.Errorf("reminders sent = %d, want exactly 1", reminders)
	}
}
