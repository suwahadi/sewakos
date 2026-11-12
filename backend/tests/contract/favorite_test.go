package contract

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func seedFavoriteTenant(t *testing.T, pool *pgxpool.Pool) (int64, string) {
	t.Helper()
	var tenantID int64
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO users (name, role, phone, phone_verified_at) VALUES ('Penyewa Favorit', 'tenant', $1, now()) RETURNING id`,
		fmt.Sprintf("088%09d", time.Now().UnixNano()%1_000_000_000)).Scan(&tenantID); err != nil {
		t.Fatal(err)
	}
	return tenantID, mintTenantToken(t, pool, tenantID)
}

func favoriteCount(t *testing.T, pool *pgxpool.Pool, tenantID, propertyID int64) int {
	t.Helper()
	var count int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM favorites WHERE user_id = $1 AND property_id = $2`, tenantID, propertyID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}

type favoritesResponse struct {
	Favorites []struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
		Area struct {
			Name string `json:"name"`
		} `json:"area"`
		Gender               string   `json:"gender"`
		ThumbnailURL         *string  `json:"thumbnail_url"`
		MinPriceMonthly      *int64   `json:"min_price_monthly"`
		MinPriceMonthlyLabel *string  `json:"min_price_monthly_label"`
		RatingAverage        *float64 `json:"rating_average"`
		IsAvailable          bool     `json:"is_available"`
	} `json:"favorites"`
	CurrentPage int `json:"current_page"`
}

func TestFavoriteAddAndRemoveAreIdempotentAndReflectedInDetail(t *testing.T) {
	srv, pool := newTestServerAndPool(t)
	tenantID, token := seedFavoriteTenant(t, pool)
	ownerID := seedOwner(t, pool)
	areaID := seedArea(t, pool, "Kota Favorit", "city", false, -6.2, 106.8)
	propertyID := seedProperty(t, pool, propertyFixture{ownerID: ownerID, areaID: areaID, name: "Kost Favorit", gender: "putri", lat: -6.2, lng: 106.8})
	seedRoom(t, pool, propertyID, "Kamar A", 1750000, 2)
	path := fmt.Sprintf("/api/favorites/%d", propertyID)
	detailPath := fmt.Sprintf("/api/listings/%d", propertyID)

	isFavorite := func() bool {
		_, body := bookingAuthedRequest(t, srv, http.MethodGet, detailPath, token, "")
		return strings.Contains(string(body), `"is_favorite":true`)
	}
	if isFavorite() {
		t.Fatal("is_favorite = true before adding")
	}

	for i := 0; i < 2; i++ {
		resp, body := bookingAuthedRequest(t, srv, http.MethodPut, path, token, "")
		if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), `"Ditambahkan ke favorit."`) {
			t.Fatalf("add #%d: status = %d, body = %s", i+1, resp.StatusCode, body)
		}
	}
	if count := favoriteCount(t, pool, tenantID, propertyID); count != 1 {
		t.Errorf("favorites rows = %d, want 1", count)
	}
	if !isFavorite() {
		t.Error("is_favorite = false after adding")
	}
	_, anonymous := bookingAuthedRequest(t, srv, http.MethodGet, detailPath, "", "")
	if !strings.Contains(string(anonymous), `"is_favorite":false`) {
		t.Errorf("anonymous detail is_favorite: %s", anonymous)
	}

	for i := 0; i < 2; i++ {
		resp, body := bookingAuthedRequest(t, srv, http.MethodDelete, path, token, "")
		if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), `"Dihapus dari favorit."`) {
			t.Fatalf("remove #%d: status = %d, body = %s", i+1, resp.StatusCode, body)
		}
	}
	if count := favoriteCount(t, pool, tenantID, propertyID); count != 0 {
		t.Errorf("favorites rows = %d, want 0", count)
	}
	if isFavorite() {
		t.Error("is_favorite = true after removing")
	}
}

func TestFavoriteRejectsUnknownPropertyGuestsAndOtherRoles(t *testing.T) {
	srv, pool := newTestServerAndPool(t)
	tenantID, token := seedFavoriteTenant(t, pool)
	ownerID := seedOwner(t, pool)
	areaID := seedArea(t, pool, "Kota Favorit", "city", false, -6.2, 106.8)
	propertyID := seedProperty(t, pool, propertyFixture{ownerID: ownerID, areaID: areaID, name: "Kost Favorit", gender: "putra", lat: -6.2, lng: 106.8})
	path := fmt.Sprintf("/api/favorites/%d", propertyID)

	for _, method := range []string{http.MethodPut, http.MethodDelete} {
		resp, body := bookingAuthedRequest(t, srv, method, "/api/favorites/999999999", token, "")
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("%s unknown property: status = %d, body = %s", method, resp.StatusCode, body)
		}
		resp, body = bookingAuthedRequest(t, srv, method, path, "", "")
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("%s without token: status = %d, body = %s", method, resp.StatusCode, body)
		}
		resp, body = bookingAuthedRequest(t, srv, method, path, mintTenantToken(t, pool, ownerID), "")
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("%s as owner: status = %d, body = %s", method, resp.StatusCode, body)
		}
	}
	resp, body := bookingAuthedRequest(t, srv, http.MethodGet, "/api/favorites", mintTenantToken(t, pool, ownerID), "")
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("list as owner: status = %d, body = %s", resp.StatusCode, body)
	}

	if _, err := pool.Exec(context.Background(), `UPDATE properties SET is_active = false WHERE id = $1`, propertyID); err != nil {
		t.Fatal(err)
	}
	resp, body = bookingAuthedRequest(t, srv, http.MethodPut, path, token, "")
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("add unlisted property: status = %d, body = %s", resp.StatusCode, body)
	}
	if count := favoriteCount(t, pool, tenantID, propertyID); count != 0 {
		t.Errorf("favorites rows after rejected add = %d", count)
	}
}

func TestFavoriteListShowsNewestFirstAndKeepsUnlistedPropertiesRemovable(t *testing.T) {
	srv, pool := newTestServerAndPool(t)
	ctx := context.Background()
	tenantID, token := seedFavoriteTenant(t, pool)
	otherTenantID, otherToken := seedFavoriteTenant(t, pool)
	ownerID := seedOwner(t, pool)
	areaID := seedArea(t, pool, "Kota Favorit", "city", false, -6.2, 106.8)

	first := seedProperty(t, pool, propertyFixture{ownerID: ownerID, areaID: areaID, name: "Kost Pertama", gender: "campur", lat: -6.2, lng: 106.8})
	seedRoom(t, pool, first, "Kamar Mahal", 2500000, 1)
	roomID := seedRoom(t, pool, first, "Kamar Murah", 1250000, 1)
	seedPropertyPhoto(t, pool, first)
	second := seedProperty(t, pool, propertyFixture{ownerID: ownerID, areaID: areaID, name: "Kost Kedua", gender: "putri", lat: -6.2, lng: 106.8})

	for i, propertyID := range []int64{first, second} {
		if _, err := pool.Exec(ctx, `INSERT INTO favorites (user_id, property_id, created_at) VALUES ($1, $2, now() - make_interval(hours => $3))`,
			tenantID, propertyID, 2-i); err != nil {
			t.Fatal(err)
		}
	}
	for _, review := range []struct {
		rating int
		hidden bool
	}{{4, false}, {5, false}, {1, true}} {
		var bookingID int64
		if err := pool.QueryRow(ctx, `
			INSERT INTO bookings (tenant_id, room_id, rent_type, start_date, duration, end_date, status, period_price, price, deposit, admin_fee, expires_at)
			VALUES ($1, $2, 'monthly', current_date, 1, current_date + 30, 'completed', 1250000, 1250000, 1250000, 0, now()) RETURNING id`,
			otherTenantID, roomID).Scan(&bookingID); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO reviews (booking_id, property_id, tenant_id, rating, is_hidden) VALUES ($1, $2, $3, $4, $5)`,
			bookingID, first, otherTenantID, review.rating, review.hidden); err != nil {
			t.Fatal(err)
		}
	}

	list := func(path, token string) favoritesResponse {
		resp, body := bookingAuthedRequest(t, srv, http.MethodGet, path, token, "")
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("list favorites: status = %d, body = %s", resp.StatusCode, body)
		}
		var parsed favoritesResponse
		mustUnmarshal(t, body, &parsed)
		return parsed
	}

	all := list("/api/favorites", token)
	if len(all.Favorites) != 2 || all.CurrentPage != 1 || all.Favorites[0].ID != second || all.Favorites[1].ID != first {
		t.Fatalf("favorites = %+v", all)
	}
	withRooms := all.Favorites[1]
	if withRooms.Name != "Kost Pertama" || withRooms.Gender != "campur" || withRooms.Area.Name != "Kota Favorit" ||
		withRooms.ThumbnailURL == nil || !strings.HasSuffix(*withRooms.ThumbnailURL, "_thumb.webp") ||
		withRooms.MinPriceMonthly == nil || *withRooms.MinPriceMonthly != 1250000 ||
		withRooms.MinPriceMonthlyLabel == nil || *withRooms.MinPriceMonthlyLabel != "Rp1.250.000" ||
		withRooms.RatingAverage == nil || *withRooms.RatingAverage != 4.5 || !withRooms.IsAvailable {
		t.Errorf("favorite with rooms = %+v", withRooms)
	}
	empty := all.Favorites[0]
	if empty.ThumbnailURL != nil || empty.MinPriceMonthly != nil || empty.MinPriceMonthlyLabel != nil || empty.RatingAverage != nil || !empty.IsAvailable {
		t.Errorf("favorite without rooms, photos or reviews = %+v", empty)
	}

	paged := list("/api/favorites?limit=1&page=2", token)
	if len(paged.Favorites) != 1 || paged.Favorites[0].ID != first || paged.CurrentPage != 2 {
		t.Errorf("page 2 = %+v", paged)
	}
	if others := list("/api/favorites", otherToken); len(others.Favorites) != 0 {
		t.Errorf("another tenant sees favorites: %+v", others)
	}

	if _, err := pool.Exec(ctx, `UPDATE properties SET moderation_status = 'suspended' WHERE id = $1`, first); err != nil {
		t.Fatal(err)
	}
	unlisted := list("/api/favorites", token)
	if len(unlisted.Favorites) != 2 || unlisted.Favorites[1].ID != first || unlisted.Favorites[1].IsAvailable {
		t.Errorf("unlisted property in favorites = %+v", unlisted)
	}
	resp, body := bookingAuthedRequest(t, srv, http.MethodDelete, fmt.Sprintf("/api/favorites/%d", first), token, "")
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), `"Dihapus dari favorit."`) {
		t.Errorf("remove unlisted favorite: status = %d, body = %s", resp.StatusCode, body)
	}
	if remaining := list("/api/favorites", token); len(remaining.Favorites) != 1 || remaining.Favorites[0].ID != second {
		t.Errorf("favorites after removal = %+v", remaining)
	}
}
