package contract

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// discoveryTestIP gives each test its own synthetic X-Forwarded-For address, so
// this file's requests don't share the "api" rate limiter's per-IP bucket
// (SPEC-GO.md §5.1, 60/min) with unrelated tests when the whole suite runs
// together (see DECISIONS.md).
func discoveryTestIP(t *testing.T) string {
	t.Helper()
	h := fnv.New32a()
	_, _ = h.Write([]byte(t.Name()))
	n := h.Sum32()
	return fmt.Sprintf("10.%d.%d.%d", (n>>16)&0xff, (n>>8)&0xff, n&0xff)
}

func discoveryGetJSON(t *testing.T, srv *httptest.Server, path string) (*http.Response, []byte) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, srv.URL+path, nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("x-api-client", "KOSAPI WEB:WEB")
	req.Header.Set("X-Forwarded-For", discoveryTestIP(t))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		t.Fatalf("read response: %v", err)
	}
	return resp, buf.Bytes()
}

func discoveryPostJSON(t *testing.T, srv *httptest.Server, path, body string) (*http.Response, []byte) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, srv.URL+path, bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("x-api-client", "KOSAPI WEB:WEB")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-For", discoveryTestIP(t))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		t.Fatalf("read response: %v", err)
	}
	return resp, buf.Bytes()
}

func seedOwner(t *testing.T, pool *pgxpool.Pool) int64 {
	t.Helper()
	phone := fmt.Sprintf("081%09d", time.Now().UnixNano()%1_000_000_000)
	var id int64
	err := pool.QueryRow(context.Background(),
		`INSERT INTO users (name, role, phone, phone_verified_at) VALUES ($1, 'owner', $2, now()) RETURNING id`,
		"Pemilik Test", phone).Scan(&id)
	if err != nil {
		t.Fatalf("seed owner: %v", err)
	}
	return id
}

func seedArea(t *testing.T, pool *pgxpool.Pool, name, areaType string, isTop bool, lat, lng float64) int64 {
	t.Helper()
	slug := fmt.Sprintf("%s-%d", name, time.Now().UnixNano())
	var id int64
	err := pool.QueryRow(context.Background(),
		`INSERT INTO areas (name, slug, type, is_top, latitude, longitude) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		name, slug, areaType, isTop, lat, lng).Scan(&id)
	if err != nil {
		t.Fatalf("seed area: %v", err)
	}
	return id
}

type propertyFixture struct {
	ownerID   int64
	areaID    int64
	name      string
	gender    string
	lat, lng  float64
	flashSale bool
}

func seedProperty(t *testing.T, pool *pgxpool.Pool, f propertyFixture) int64 {
	t.Helper()
	slug := fmt.Sprintf("%s-%d", f.name, time.Now().UnixNano())
	var id int64
	err := pool.QueryRow(context.Background(),
		`INSERT INTO properties (owner_id, area_id, name, slug, gender, latitude, longitude,
			is_active, moderation_status, is_flash_sale, flash_sale_discount)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, true, 'approved', $8, 250000) RETURNING id`,
		f.ownerID, f.areaID, f.name, slug, f.gender, f.lat, f.lng, f.flashSale).Scan(&id)
	if err != nil {
		t.Fatalf("seed property: %v", err)
	}
	return id
}

func seedRoom(t *testing.T, pool *pgxpool.Pool, propertyID int64, name string, priceMonthly int64, available int) int64 {
	t.Helper()
	var id int64
	err := pool.QueryRow(context.Background(),
		`INSERT INTO rooms (property_id, name, price_monthly, available) VALUES ($1, $2, $3, $4) RETURNING id`,
		propertyID, name, priceMonthly, available).Scan(&id)
	if err != nil {
		t.Fatalf("seed room: %v", err)
	}
	return id
}

func TestAreaListFixture(t *testing.T) {
	srv, pool := newTestServerAndPool(t)

	topID := seedArea(t, pool, "Kota Top", "city", true, -6.2, 106.8)
	nonTopID := seedArea(t, pool, "Kota Bukan Top", "city", false, -6.3, 106.9)

	resp, body := discoveryGetJSON(t, srv, "/api/area/city?top=true")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, body)
	}
	var parsed struct {
		Areas []struct {
			ID int64 `json:"id"`
		} `json:"areas"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("parse body: %v", err)
	}
	var sawTop, sawNonTop bool
	for _, a := range parsed.Areas {
		if a.ID == topID {
			sawTop = true
		}
		if a.ID == nonTopID {
			sawNonTop = true
		}
	}
	if !sawTop {
		t.Fatalf("expected the top area %d in result, got %+v", topID, parsed.Areas)
	}
	if sawNonTop {
		t.Fatalf("non-top area %d leaked into top=true result", nonTopID)
	}
}

func TestAreaListRejectsUnknownType(t *testing.T) {
	srv := newTestServer(t)
	resp, _ := discoveryGetJSON(t, srv, "/api/area/planet")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

func TestSuggestionFixture(t *testing.T) {
	srv, pool := newTestServerAndPool(t)
	ownerID := seedOwner(t, pool)
	keyword := fmt.Sprintf("Depok%d", time.Now().UnixNano())
	areaID := seedArea(t, pool, "Kota "+keyword, "city", false, -6.3878486, 106.8177948)
	propertyID := seedProperty(t, pool, propertyFixture{
		ownerID: ownerID, areaID: areaID, name: "Kost " + keyword, gender: "campur", lat: -6.3, lng: 106.8,
	})
	seedRoom(t, pool, propertyID, "Kamar "+keyword+" Tipe A", 2000000, 1)

	resp, body := discoveryGetJSON(t, srv, "/api/suggestion?keyword="+keyword)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, body)
	}
	var parsed struct {
		Suggestion struct {
			Areas []struct {
				GeocodeID int64 `json:"geocode_id"`
			} `json:"areas"`
			AreaSource string `json:"area_source"`
			Landings   []struct {
				ID int64 `json:"id"`
			} `json:"landings"`
			RoomTypes []struct {
				PropertyID int64 `json:"property_id"`
			} `json:"room_types"`
		} `json:"suggestion"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("parse body: %v", err)
	}
	if parsed.Suggestion.AreaSource != "db" {
		t.Fatalf("area_source = %q, want db", parsed.Suggestion.AreaSource)
	}
	if len(parsed.Suggestion.Areas) != 1 || parsed.Suggestion.Areas[0].GeocodeID != areaID {
		t.Fatalf("areas = %+v, want one matching %d", parsed.Suggestion.Areas, areaID)
	}
	if len(parsed.Suggestion.Landings) != 1 || parsed.Suggestion.Landings[0].ID != propertyID {
		t.Fatalf("landings = %+v, want one matching %d", parsed.Suggestion.Landings, propertyID)
	}
	if len(parsed.Suggestion.RoomTypes) != 1 || parsed.Suggestion.RoomTypes[0].PropertyID != propertyID {
		t.Fatalf("room_types = %+v, want one matching property %d", parsed.Suggestion.RoomTypes, propertyID)
	}
}

func TestSuggestionRejectsShortKeyword(t *testing.T) {
	srv := newTestServer(t)
	resp, _ := discoveryGetJSON(t, srv, "/api/suggestion?keyword=ab")
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", resp.StatusCode)
	}
}

func searchRoomsResult(t *testing.T, srv *httptest.Server, requestBody string) map[string]any {
	t.Helper()
	resp, body := discoveryPostJSON(t, srv, "/api/listings/search", requestBody)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, body)
	}
	var parsed map[string]any
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("parse body: %v", err)
	}
	return parsed
}

func TestSearchFixture(t *testing.T) {
	srv, pool := newTestServerAndPool(t)
	ownerID := seedOwner(t, pool)
	areaID := seedArea(t, pool, "Kota Search", "city", false, -6.2, 106.8)
	cheapProperty := seedProperty(t, pool, propertyFixture{
		ownerID: ownerID, areaID: areaID, name: "Kost Murah", gender: "campur", lat: -6.2, lng: 106.8,
	})
	seedRoom(t, pool, cheapProperty, "Kamar Murah", 1000000, 2)
	expensiveProperty := seedProperty(t, pool, propertyFixture{
		ownerID: ownerID, areaID: areaID, name: "Kost Mahal", gender: "putri", lat: -6.2, lng: 106.8,
	})
	seedRoom(t, pool, expensiveProperty, "Kamar Mahal", 5000000, 0)

	result := searchRoomsResult(t, srv, fmt.Sprintf(`{
		"filters": {"area_id": %d},
		"sorting": {"field": "price", "direction": "-"},
		"limit": 20, "offset": 0
	}`, areaID))

	if result["source"] != "db" {
		t.Fatalf("source = %v", result["source"])
	}
	rooms, _ := result["rooms"].([]any)
	if len(rooms) != 2 {
		t.Fatalf("expected 2 rooms, got %d: %+v", len(rooms), rooms)
	}
	first := rooms[0].(map[string]any)
	if first["price"].(float64) != 5000000 {
		t.Fatalf("expected highest price first (DESC), got %+v", first)
	}
	if first["rating_average"] != nil {
		t.Fatalf("rating_average should be null with no reviews, got %v", first["rating_average"])
	}
	if first["thumbnail_url"] != nil {
		t.Fatalf("thumbnail_url should be null with no photos, got %v", first["thumbnail_url"])
	}

	genderFiltered := searchRoomsResult(t, srv, `{"filters": {"gender": [2]}, "limit": 20, "offset": 0}`)
	rooms, _ = genderFiltered["rooms"].([]any)
	for _, raw := range rooms {
		room := raw.(map[string]any)
		if room["gender"] != "putri" {
			t.Fatalf("gender filter leaked a non-putri room: %+v", room)
		}
	}

	availableOnly := searchRoomsResult(t, srv, `{"filters": {"is_available": true}, "limit": 20, "offset": 0}`)
	rooms, _ = availableOnly["rooms"].([]any)
	for _, raw := range rooms {
		room := raw.(map[string]any)
		if room["available"].(float64) < 1 {
			t.Fatalf("is_available filter leaked a room with none available: %+v", room)
		}
	}

	apartemen := searchRoomsResult(t, srv, `{"filters": {"property_type": "apartemen"}, "limit": 20, "offset": 0}`)
	rooms, _ = apartemen["rooms"].([]any)
	if len(rooms) != 0 || apartemen["total"].(float64) != 0 {
		t.Fatalf("apartemen filter should return empty, got %+v", apartemen)
	}

	paged := searchRoomsResult(t, srv, fmt.Sprintf(`{"filters": {"area_id": %d}, "limit": 1, "offset": 0}`, areaID))
	if paged["has-more"] != true {
		t.Fatalf("expected has-more true with limit 1 and 2 matches, got %+v", paged)
	}
	if paged["next-offset"].(float64) != 1 {
		t.Fatalf("expected next-offset 1, got %+v", paged["next-offset"])
	}
}

func TestSearchRejectsUnknownAreaID(t *testing.T) {
	srv := newTestServer(t)
	resp, body := discoveryPostJSON(t, srv, "/api/listings/search", `{"filters": {"area_id": 999999999}}`)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, body)
	}
}
