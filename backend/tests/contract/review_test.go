package contract

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/suwahadi/sewakos/backend/internal/money"
)

func bookingPropertyID(t *testing.T, pool *pgxpool.Pool, bookingID int64) int64 {
	t.Helper()
	var propertyID int64
	if err := pool.QueryRow(context.Background(),
		`SELECT r.property_id FROM bookings b JOIN rooms r ON r.id = b.room_id WHERE b.id = $1`, bookingID).Scan(&propertyID); err != nil {
		t.Fatal(err)
	}
	return propertyID
}

func reviewsBody(t *testing.T, srv *httptest.Server, propertyID int64) reviewsResponse {
	t.Helper()
	resp, body := bookingAuthedRequest(t, srv, http.MethodGet, fmt.Sprintf("/api/listings/%d/reviews", propertyID), "", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list reviews: status = %d, body = %s", resp.StatusCode, body)
	}
	var parsed reviewsResponse
	mustUnmarshal(t, body, &parsed)
	return parsed
}

type reviewsResponse struct {
	Rating struct {
		Average *float64 `json:"average"`
		Count   int      `json:"count"`
	} `json:"rating"`
	Reviews []struct {
		ID         int64   `json:"id"`
		Rating     int     `json:"rating"`
		Comment    *string `json:"comment"`
		TenantName string  `json:"tenant_name"`
		RoomName   string  `json:"room_name"`
	} `json:"reviews"`
	CurrentPage int `json:"current_page"`
}

func TestReviewCreateValidationOwnershipAndEligibility(t *testing.T) {
	srv, pool := newTestServerAndPool(t)
	future := insertApprovedMonthlyBooking(t, pool, money.TodayWIB().AddDate(0, 0, 3).Format(time.DateOnly), 1)
	token := mintTenantToken(t, pool, future.tenantID)
	path := fmt.Sprintf("/api/booking/%d/review", future.bookingID)

	resp, body := bookingAuthedRequest(t, srv, http.MethodPost, path, token, `{}`)
	if resp.StatusCode != http.StatusUnprocessableEntity || !strings.Contains(string(body), "The rating field is required.") {
		t.Errorf("missing rating: status = %d, body = %s", resp.StatusCode, body)
	}
	resp, body = bookingAuthedRequest(t, srv, http.MethodPost, path, token, `{"rating":6}`)
	if resp.StatusCode != http.StatusUnprocessableEntity || !strings.Contains(string(body), "The rating field must be between 1 and 5.") {
		t.Errorf("rating out of range: status = %d, body = %s", resp.StatusCode, body)
	}
	resp, body = bookingAuthedRequest(t, srv, http.MethodPost, path, token, `{"rating":4.5}`)
	if resp.StatusCode != http.StatusUnprocessableEntity || !strings.Contains(string(body), "The rating field must be an integer.") {
		t.Errorf("fractional rating: status = %d, body = %s", resp.StatusCode, body)
	}
	resp, body = bookingAuthedRequest(t, srv, http.MethodPost, path, token, fmt.Sprintf(`{"rating":5,"comment":%q}`, strings.Repeat("a", 1001)))
	if resp.StatusCode != http.StatusUnprocessableEntity || !strings.Contains(string(body), "The comment field must not be greater than 1000 characters.") {
		t.Errorf("long comment: status = %d, body = %s", resp.StatusCode, body)
	}

	resp, body = bookingAuthedRequest(t, srv, http.MethodPost, path, token, `{"rating":5}`)
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(string(body), "Booking belum bisa diulas.") {
		t.Errorf("approved booking before start_date: status = %d, body = %s", resp.StatusCode, body)
	}

	other := insertApprovedMonthlyBooking(t, pool, money.TodayWIB().Format(time.DateOnly), 1)
	resp, body = bookingAuthedRequest(t, srv, http.MethodPost, path, mintTenantToken(t, pool, other.tenantID), `{"rating":5}`)
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("other tenant: status = %d, body = %s", resp.StatusCode, body)
	}
	ownerToken := mintTenantToken(t, pool, future.ownerID)
	resp, body = bookingAuthedRequest(t, srv, http.MethodPost, path, ownerToken, `{"rating":5}`)
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("owner role: status = %d, body = %s", resp.StatusCode, body)
	}
	resp, body = bookingAuthedRequest(t, srv, http.MethodPost, "/api/booking/999999999/review", token, `{"rating":5}`)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("unknown booking: status = %d, body = %s", resp.StatusCode, body)
	}

	if _, err := pool.Exec(context.Background(), `UPDATE bookings SET status = 'paid' WHERE id = $1`, future.bookingID); err != nil {
		t.Fatal(err)
	}
	resp, body = bookingAuthedRequest(t, srv, http.MethodPost, path, token, `{"rating":5}`)
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(string(body), "Booking belum bisa diulas.") {
		t.Errorf("paid booking: status = %d, body = %s", resp.StatusCode, body)
	}
	for _, status := range []string{"completed", "terminated"} {
		f := insertApprovedMonthlyBooking(t, pool, money.TodayWIB().AddDate(0, 0, 3).Format(time.DateOnly), 1)
		if _, err := pool.Exec(context.Background(), `UPDATE bookings SET status = $2 WHERE id = $1`, f.bookingID, status); err != nil {
			t.Fatal(err)
		}
		resp, body = bookingAuthedRequest(t, srv, http.MethodPost, fmt.Sprintf("/api/booking/%d/review", f.bookingID),
			mintTenantToken(t, pool, f.tenantID), `{"rating":3}`)
		if resp.StatusCode != http.StatusOK {
			t.Errorf("%s booking: status = %d, body = %s", status, resp.StatusCode, body)
		}
	}
}

func TestReviewOncePerBookingUpdatesRatingCanReviewAndNotifiesOwner(t *testing.T) {
	srv, pool := newTestServerAndPool(t)
	f := insertApprovedMonthlyBooking(t, pool, money.TodayWIB().Format(time.DateOnly), 1)
	token := mintTenantToken(t, pool, f.tenantID)
	propertyID := bookingPropertyID(t, pool, f.bookingID)

	canReview := func() bool {
		_, body := bookingAuthedRequest(t, srv, http.MethodGet, fmt.Sprintf("/api/booking/%d", f.bookingID), token, "")
		var parsed struct {
			Booking struct {
				CanReview bool `json:"can_review"`
			} `json:"booking"`
		}
		mustUnmarshal(t, body, &parsed)
		return parsed.Booking.CanReview
	}
	if !canReview() {
		t.Fatal("can_review = false before the review")
	}
	if before := reviewsBody(t, srv, propertyID); before.Rating.Average != nil || before.Rating.Count != 0 || len(before.Reviews) != 0 {
		t.Fatalf("reviews before = %+v", before)
	}

	path := fmt.Sprintf("/api/booking/%d/review", f.bookingID)
	resp, body := bookingAuthedRequest(t, srv, http.MethodPost, path, token, `{"rating":5,"comment":"Bersih dan dekat stasiun."}`)
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), `"Terima kasih atas ulasannya."`) {
		t.Fatalf("create review: status = %d, body = %s", resp.StatusCode, body)
	}
	var created struct {
		Review struct {
			ID         int64  `json:"id"`
			Rating     int    `json:"rating"`
			Comment    string `json:"comment"`
			TenantName string `json:"tenant_name"`
			RoomName   string `json:"room_name"`
			CreatedAt  string `json:"created_at"`
		} `json:"review"`
	}
	mustUnmarshal(t, body, &created)
	if created.Review.ID == 0 || created.Review.Rating != 5 || created.Review.Comment != "Bersih dan dekat stasiun." ||
		created.Review.TenantName != "Penyewa S." || created.Review.RoomName != "Kamar Siklus" || !strings.HasSuffix(created.Review.CreatedAt, "Z") {
		t.Errorf("review = %+v", created.Review)
	}

	resp, body = bookingAuthedRequest(t, srv, http.MethodPost, path, token, `{"rating":1}`)
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(string(body), "Booking ini sudah diulas.") {
		t.Errorf("second review: status = %d, body = %s", resp.StatusCode, body)
	}
	if canReview() {
		t.Error("can_review = true after the review")
	}

	after := reviewsBody(t, srv, propertyID)
	if after.Rating.Average == nil || *after.Rating.Average != 5 || after.Rating.Count != 1 || len(after.Reviews) != 1 ||
		after.Reviews[0].TenantName != "Penyewa S." || after.CurrentPage != 1 {
		t.Errorf("reviews after = %+v", after)
	}

	_, detail := bookingAuthedRequest(t, srv, http.MethodGet, fmt.Sprintf("/api/listings/%d", propertyID), "", "")
	if !strings.Contains(string(detail), `"rating":{"average":5,"count":1}`) {
		t.Errorf("listing detail rating not updated: %s", detail)
	}
	if title, text := latestNotification(t, pool, f.ownerID); title != "Ulasan baru" || text != "Kost Siklus mendapat ulasan bintang 5." {
		t.Errorf("owner notification = %q / %q", title, text)
	}
	var link string
	if err := pool.QueryRow(context.Background(),
		`SELECT link FROM user_notifications WHERE user_id = $1 ORDER BY id DESC LIMIT 1`, f.ownerID).Scan(&link); err != nil || link != "/owner/ulasan" {
		t.Errorf("owner notification link = %q (err %v)", link, err)
	}
}

func TestReviewListIsPublicNewestFirstAndHiddenForUnlistedProperty(t *testing.T) {
	srv, pool := newTestServerAndPool(t)
	f := insertApprovedMonthlyBooking(t, pool, money.TodayWIB().Format(time.DateOnly), 1)
	propertyID := bookingPropertyID(t, pool, f.bookingID)
	ctx := context.Background()
	for i, rating := range []int{2, 4} {
		var bookingID int64
		if err := pool.QueryRow(ctx, `
			INSERT INTO bookings (tenant_id, room_id, rent_type, start_date, duration, end_date, status, period_price, price, deposit, admin_fee, expires_at)
			SELECT tenant_id, room_id, rent_type, start_date, duration, end_date, 'completed', period_price, price, deposit, admin_fee, expires_at
			FROM bookings WHERE id = $1 RETURNING id`, f.bookingID).Scan(&bookingID); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO reviews (booking_id, property_id, tenant_id, rating, created_at)
			VALUES ($1, $2, $3, $4, now() - make_interval(hours => $5))`, bookingID, propertyID, f.tenantID, rating, 2-i); err != nil {
			t.Fatal(err)
		}
	}

	list := reviewsBody(t, srv, propertyID)
	if list.Rating.Count != 2 || list.Rating.Average == nil || *list.Rating.Average != 3 ||
		len(list.Reviews) != 2 || list.Reviews[0].Rating != 4 || list.Reviews[1].Rating != 2 || list.Reviews[0].Comment != nil {
		t.Errorf("list = %+v", list)
	}

	resp, body := bookingAuthedRequest(t, srv, http.MethodGet, fmt.Sprintf("/api/listings/%d/reviews?limit=1&page=2", propertyID), "", "")
	var paged reviewsResponse
	mustUnmarshal(t, body, &paged)
	if resp.StatusCode != http.StatusOK || len(paged.Reviews) != 1 || paged.Reviews[0].Rating != 2 || paged.CurrentPage != 2 {
		t.Errorf("page 2 = %+v", paged)
	}

	if _, err := pool.Exec(ctx, `UPDATE properties SET is_active = false WHERE id = $1`, propertyID); err != nil {
		t.Fatal(err)
	}
	resp, body = bookingAuthedRequest(t, srv, http.MethodGet, fmt.Sprintf("/api/listings/%d/reviews", propertyID), "", "")
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("inactive property: status = %d, body = %s", resp.StatusCode, body)
	}
}

func TestReviewModerationHidesFromPublicAndRecalculatesRating(t *testing.T) {
	srv, pool := newTestServerAndPool(t)
	admin := adminToken(t, srv)
	ctx := context.Background()
	f := insertApprovedMonthlyBooking(t, pool, money.TodayWIB().Format(time.DateOnly), 1)
	propertyID := bookingPropertyID(t, pool, f.bookingID)
	tenantToken := mintTenantToken(t, pool, f.tenantID)
	ownerToken := mintTenantToken(t, pool, f.ownerID)

	_, body := bookingAuthedRequest(t, srv, http.MethodPost, fmt.Sprintf("/api/booking/%d/review", f.bookingID), tenantToken, `{"rating":1,"comment":"Kasar"}`)
	var created struct {
		Review struct {
			ID int64 `json:"id"`
		} `json:"review"`
	}
	mustUnmarshal(t, body, &created)
	reviewID := created.Review.ID
	if cached := reviewsBody(t, srv, propertyID); cached.Rating.Count != 1 {
		t.Fatalf("rating before hide = %+v", cached.Rating)
	}

	hidePath := fmt.Sprintf("/api/admin/reviews/%d/hide", reviewID)
	resp, body := bookingAuthedRequest(t, srv, http.MethodPost, hidePath, admin, `{}`)
	if resp.StatusCode != http.StatusUnprocessableEntity || !strings.Contains(string(body), "The reason field is required.") {
		t.Errorf("hide without reason: status = %d, body = %s", resp.StatusCode, body)
	}
	resp, body = bookingAuthedRequest(t, srv, http.MethodPost, hidePath, admin, fmt.Sprintf(`{"reason":%q}`, strings.Repeat("a", 501)))
	if resp.StatusCode != http.StatusUnprocessableEntity || !strings.Contains(string(body), "The reason field must not be greater than 500 characters.") {
		t.Errorf("hide with long reason: status = %d, body = %s", resp.StatusCode, body)
	}
	resp, body = bookingAuthedRequest(t, srv, http.MethodPost, hidePath, tenantToken, `{"reason":"x"}`)
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("tenant hide: status = %d, body = %s", resp.StatusCode, body)
	}
	resp, body = bookingAuthedRequest(t, srv, http.MethodPost, "/api/admin/reviews/999999999/hide", admin, `{"reason":"x"}`)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("unknown review: status = %d, body = %s", resp.StatusCode, body)
	}

	resp, body = bookingAuthedRequest(t, srv, http.MethodPost, hidePath, admin, `{"reason":"Kata kasar"}`)
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), `"Ulasan disembunyikan."`) {
		t.Fatalf("hide: status = %d, body = %s", resp.StatusCode, body)
	}
	if hidden := reviewsBody(t, srv, propertyID); hidden.Rating.Count != 0 || hidden.Rating.Average != nil || len(hidden.Reviews) != 0 {
		t.Errorf("public list after hide = %+v", hidden)
	}
	_, detail := bookingAuthedRequest(t, srv, http.MethodGet, fmt.Sprintf("/api/listings/%d", propertyID), "", "")
	if !strings.Contains(string(detail), `"rating":{"average":null,"count":0}`) {
		t.Errorf("listing detail still counts the hidden review: %s", detail)
	}

	_, body = bookingAuthedRequest(t, srv, http.MethodGet, fmt.Sprintf("/api/admin/reviews?is_hidden=true&property_id=%d", propertyID), admin, "")
	if !strings.Contains(string(body), fmt.Sprintf(`"id":%d`, reviewID)) || !strings.Contains(string(body), `"hidden_reason":"Kata kasar"`) ||
		!strings.Contains(string(body), `"tenant_name":"Penyewa Siklus"`) || !strings.Contains(string(body), `"is_hidden":true`) {
		t.Errorf("admin hidden list = %s", body)
	}
	_, body = bookingAuthedRequest(t, srv, http.MethodGet, fmt.Sprintf("/api/admin/reviews?is_hidden=false&property_id=%d", propertyID), admin, "")
	if strings.Contains(string(body), fmt.Sprintf(`"id":%d,`, reviewID)) {
		t.Errorf("admin visible list contains the hidden review: %s", body)
	}

	_, body = bookingAuthedRequest(t, srv, http.MethodGet, fmt.Sprintf("/api/owner/reviews?property_id=%d", propertyID), ownerToken, "")
	if !strings.Contains(string(body), fmt.Sprintf(`"id":%d`, reviewID)) || !strings.Contains(string(body), `"is_hidden":true`) ||
		!strings.Contains(string(body), `"tenant_name":"Penyewa S."`) || !strings.Contains(string(body), fmt.Sprintf(`"property_id":%d`, propertyID)) ||
		strings.Contains(string(body), "hidden_reason") {
		t.Errorf("owner list = %s", body)
	}
	otherOwner := insertApprovedMonthlyBooking(t, pool, money.TodayWIB().Format(time.DateOnly), 1)
	_, body = bookingAuthedRequest(t, srv, http.MethodGet, "/api/owner/reviews", mintTenantToken(t, pool, otherOwner.ownerID), "")
	if strings.Contains(string(body), fmt.Sprintf(`"id":%d,`, reviewID)) {
		t.Errorf("another owner sees the review: %s", body)
	}
	resp, _ = bookingAuthedRequest(t, srv, http.MethodGet, "/api/owner/reviews", tenantToken, "")
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("tenant on owner reviews: status = %d", resp.StatusCode)
	}

	var action string
	var auditOwner int64
	if err := pool.QueryRow(ctx, `SELECT action, owner_id FROM audit_logs WHERE entity_type = 'review' AND entity_id = $1 ORDER BY id DESC LIMIT 1`,
		reviewID).Scan(&action, &auditOwner); err != nil || action != "review.hide" || auditOwner != f.ownerID {
		t.Errorf("audit log = %q owner %d (err %v)", action, auditOwner, err)
	}

	resp, body = bookingAuthedRequest(t, srv, http.MethodPost, fmt.Sprintf("/api/admin/reviews/%d/unhide", reviewID), admin, "")
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), `"Ulasan ditampilkan kembali."`) {
		t.Fatalf("unhide: status = %d, body = %s", resp.StatusCode, body)
	}
	if shown := reviewsBody(t, srv, propertyID); shown.Rating.Count != 1 || shown.Rating.Average == nil || *shown.Rating.Average != 1 || len(shown.Reviews) != 1 {
		t.Errorf("public list after unhide = %+v", shown)
	}
	var hiddenReason *string
	if err := pool.QueryRow(ctx, `SELECT hidden_reason FROM reviews WHERE id = $1`, reviewID).Scan(&hiddenReason); err != nil || hiddenReason != nil {
		t.Errorf("hidden_reason after unhide = %v (err %v)", hiddenReason, err)
	}
	if err := pool.QueryRow(ctx, `SELECT action FROM audit_logs WHERE entity_type = 'review' AND entity_id = $1 ORDER BY id DESC LIMIT 1`,
		reviewID).Scan(&action); err != nil || action != "review.unhide" {
		t.Errorf("audit log after unhide = %q (err %v)", action, err)
	}
}
