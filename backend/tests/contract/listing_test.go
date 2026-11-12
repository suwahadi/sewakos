package contract

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func seedPropertyPhoto(t *testing.T, pool *pgxpool.Pool, propertyID int64) int64 {
	t.Helper()
	key := fmt.Sprintf("properties/%d/%d", propertyID, time.Now().UnixNano())
	var id int64
	err := pool.QueryRow(context.Background(),
		`INSERT INTO property_photos (property_id, object_key, thumb_key, width, height, size_bytes, sort_order)
		 VALUES ($1, $2, $3, 1600, 900, 1000, 0) RETURNING id`,
		propertyID, key+".webp", key+"_thumb.webp").Scan(&id)
	if err != nil {
		t.Fatalf("seed property photo: %v", err)
	}
	return id
}

func TestListingDetailFixture(t *testing.T) {
	srv, pool := newTestServerAndPool(t)
	ownerID := seedOwner(t, pool)
	areaID := seedArea(t, pool, "Kota Detail", "city", false, -6.2, 106.8)
	propertyID := seedProperty(t, pool, propertyFixture{
		ownerID: ownerID, areaID: areaID, name: "Kost Detail", gender: "campur", lat: -6.2, lng: 106.8,
	})
	roomID := seedRoom(t, pool, propertyID, "Kamar Detail", 2250000, 2)
	seedPropertyPhoto(t, pool, propertyID)

	resp, body := discoveryGetJSON(t, srv, fmt.Sprintf("/api/listings/%d", propertyID))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, body)
	}
	var parsed struct {
		Property struct {
			ID         int64 `json:"id"`
			IsFavorite bool  `json:"is_favorite"`
			Owner      struct {
				Name        string `json:"name"`
				ActiveSince string `json:"active_since"`
			} `json:"owner"`
			Photos []struct {
				ID int64 `json:"id"`
			} `json:"photos"`
			Rating struct {
				Average any `json:"average"`
				Count   int `json:"count"`
			} `json:"rating"`
		} `json:"property"`
		Rooms []struct {
			ID           int64    `json:"id"`
			Facilities   []string `json:"facilities"`
			PriceMonthly int64    `json:"price_monthly"`
		} `json:"rooms"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("parse body: %v", err)
	}
	if parsed.Property.ID != propertyID {
		t.Fatalf("property.id = %d, want %d", parsed.Property.ID, propertyID)
	}
	if parsed.Property.IsFavorite {
		t.Fatalf("is_favorite should be false for an anonymous request")
	}
	if parsed.Property.Owner.Name == "" || parsed.Property.Owner.ActiveSince == "" {
		t.Fatalf("owner.name and owner.active_since must be set, got %+v", parsed.Property.Owner)
	}
	if parsed.Property.Rating.Average != nil {
		t.Fatalf("rating.average should be null with no reviews, got %v", parsed.Property.Rating.Average)
	}
	if len(parsed.Property.Photos) != 1 {
		t.Fatalf("expected 1 property photo, got %+v", parsed.Property.Photos)
	}
	if len(parsed.Rooms) != 1 || parsed.Rooms[0].ID != roomID || parsed.Rooms[0].PriceMonthly != 2250000 {
		t.Fatalf("rooms = %+v, want one matching room %d", parsed.Rooms, roomID)
	}
	if parsed.Rooms[0].Facilities == nil {
		t.Fatalf("facilities should be [] not null")
	}
}

func TestListingDetailRejectsMissing(t *testing.T) {
	srv := newTestServer(t)
	resp, _ := discoveryGetJSON(t, srv, "/api/listings/999999999")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

func TestListingDetailFavoriteForOwningTenant(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv, pool := newTestServerAndPool(t)
	_, token := registerAndVerify(t, srv)
	uid := currentUserID(t, srv, token)

	ownerID := seedOwner(t, pool)
	areaID := seedArea(t, pool, "Kota Favorit", "city", false, -6.2, 106.8)
	propertyID := seedProperty(t, pool, propertyFixture{
		ownerID: ownerID, areaID: areaID, name: "Kost Favorit", gender: "campur", lat: -6.2, lng: 106.8,
	})
	seedRoom(t, pool, propertyID, "Kamar Favorit", 1500000, 1)

	if _, err := pool.Exec(context.Background(),
		`INSERT INTO favorites (user_id, property_id) VALUES ($1, $2)`, uid, propertyID); err != nil {
		t.Fatalf("seed favorite: %v", err)
	}

	resp, body := authedRequest(t, srv, http.MethodGet, fmt.Sprintf("/api/listings/%d", propertyID), token, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, body)
	}
	var parsed struct {
		Property struct {
			IsFavorite bool `json:"is_favorite"`
		} `json:"property"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("parse body: %v", err)
	}
	if !parsed.Property.IsFavorite {
		t.Fatalf("is_favorite should be true for the tenant who favorited it")
	}
}

func TestListingViewFixture(t *testing.T) {
	srv, pool := newTestServerAndPool(t)
	ownerID := seedOwner(t, pool)
	areaID := seedArea(t, pool, "Kota View", "city", false, -6.2, 106.8)
	propertyID := seedProperty(t, pool, propertyFixture{
		ownerID: ownerID, areaID: areaID, name: "Kost View", gender: "campur", lat: -6.2, lng: 106.8,
	})
	seedRoom(t, pool, propertyID, "Kamar View", 1500000, 1)

	resp, body := discoveryPostJSON(t, srv, fmt.Sprintf("/api/listings/%d/view", propertyID), "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, body)
	}
	var parsed struct {
		Views int64 `json:"views"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("parse body: %v", err)
	}
	if parsed.Views != 1 {
		t.Fatalf("views = %d, want 1", parsed.Views)
	}

	resp, body = discoveryPostJSON(t, srv, fmt.Sprintf("/api/listings/%d/view", propertyID), "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("2nd view: status = %d, body = %s", resp.StatusCode, body)
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("parse body: %v", err)
	}
	if parsed.Views != 2 {
		t.Fatalf("views after 2nd call = %d, want 2", parsed.Views)
	}

	var dailyViews int32
	err := pool.QueryRow(context.Background(),
		`SELECT views FROM property_views_daily WHERE property_id = $1`, propertyID).Scan(&dailyViews)
	if err != nil {
		t.Fatalf("read property_views_daily: %v", err)
	}
	if dailyViews != 2 {
		t.Fatalf("property_views_daily.views = %d, want 2", dailyViews)
	}
}

func TestListingViewRejectsMissing(t *testing.T) {
	srv := newTestServer(t)
	resp, _ := discoveryPostJSON(t, srv, "/api/listings/999999999/view", "")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

func TestListingGalleryFixture(t *testing.T) {
	srv, pool := newTestServerAndPool(t)
	ownerID := seedOwner(t, pool)
	areaID := seedArea(t, pool, "Kota Galeri", "city", false, -6.2, 106.8)
	propertyID := seedProperty(t, pool, propertyFixture{
		ownerID: ownerID, areaID: areaID, name: "Kost Galeri", gender: "campur", lat: -6.2, lng: 106.8,
	})
	roomID := seedRoom(t, pool, propertyID, "Kamar Galeri", 1500000, 1)

	key := fmt.Sprintf("properties/%d/rooms/%d", propertyID, time.Now().UnixNano())
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO property_photos (property_id, room_id, object_key, thumb_key, width, height, size_bytes, sort_order)
		 VALUES ($1, $2, $3, $4, 1600, 900, 1000, 0)`,
		propertyID, roomID, key+".webp", key+"_thumb.webp"); err != nil {
		t.Fatalf("seed room photo: %v", err)
	}

	resp, body := discoveryGetJSON(t, srv, fmt.Sprintf("/api/listings/%d/gallery", propertyID))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, body)
	}
	var parsed struct {
		Gallery []struct {
			RoomID int64 `json:"room_id"`
			Photos []struct {
				ID int64 `json:"id"`
			} `json:"photos"`
		} `json:"gallery"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("parse body: %v", err)
	}
	if len(parsed.Gallery) != 1 || parsed.Gallery[0].RoomID != roomID {
		t.Fatalf("gallery = %+v, want one entry for room %d", parsed.Gallery, roomID)
	}
	if len(parsed.Gallery[0].Photos) != 1 {
		t.Fatalf("expected 1 photo for the room, got %+v", parsed.Gallery[0].Photos)
	}
}

func TestABTestFixture(t *testing.T) {
	srv := newTestServer(t)

	resp, body := discoveryGetJSON(t, srv, "/api/config/ab-test/discovery-communication?visitor_id=visitor-abc")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, body)
	}
	var parsed struct {
		Experiment string `json:"experiment"`
		Variant    string `json:"variant"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("parse body: %v", err)
	}
	if parsed.Experiment != "discovery-communication" {
		t.Fatalf("experiment = %q", parsed.Experiment)
	}
	allowed := map[string]bool{"control": true, "chat-first": true, "booking-first": true}
	if !allowed[parsed.Variant] {
		t.Fatalf("variant = %q, want one of control/chat-first/booking-first", parsed.Variant)
	}

	// Deterministic: same visitor_id, same experiment -> same variant every time.
	resp2, body2 := discoveryGetJSON(t, srv, "/api/config/ab-test/discovery-communication?visitor_id=visitor-abc")
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("2nd call: status = %d, body = %s", resp2.StatusCode, body2)
	}
	var parsed2 struct {
		Variant string `json:"variant"`
	}
	if err := json.Unmarshal(body2, &parsed2); err != nil {
		t.Fatalf("parse body: %v", err)
	}
	if parsed2.Variant != parsed.Variant {
		t.Fatalf("variant changed across calls: %q then %q", parsed.Variant, parsed2.Variant)
	}
}

func TestABTestRejectsUnknownExperiment(t *testing.T) {
	srv := newTestServer(t)
	resp, _ := discoveryGetJSON(t, srv, "/api/config/ab-test/unknown-experiment?visitor_id=visitor-abc")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

func TestABTestRejectsMissingVisitorID(t *testing.T) {
	srv := newTestServer(t)
	resp, _ := discoveryGetJSON(t, srv, "/api/config/ab-test/discovery-communication")
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", resp.StatusCode)
	}
}
