//go:build loadtest

package contract

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/suwahadi/sewakos/backend/internal/money"
)

const (
	loadPropertyCount = 10_000
	loadAreaCount     = 10
	loadWorkers       = 10
)

var loadVisitor atomic.Uint32

func loadRequest(srv *httptest.Server, method, path, token, body string) (int, []byte, time.Duration, error) {
	req, err := http.NewRequest(method, srv.URL+path, bytes.NewBufferString(body))
	if err != nil {
		return 0, nil, 0, err
	}
	n := loadVisitor.Add(1)
	req.Header.Set("x-api-client", "KOSAPI WEB:WEB")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-For", fmt.Sprintf("100.%d.%d.%d", 64+(n>>16)%64, (n>>8)%256, n%256))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	start := time.Now()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, nil, 0, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	return resp.StatusCode, data, time.Since(start), err
}

type loadResult struct {
	name      string
	latencies []time.Duration
	failures  []string
}

func (r loadResult) percentile(p float64) time.Duration {
	sorted := slices.Clone(r.latencies)
	slices.Sort(sorted)
	if len(sorted) == 0 {
		return 0
	}
	return sorted[min(len(sorted)-1, int(float64(len(sorted))*p))]
}

func runLoad(name string, requests, workers int, call func(i int) (int, []byte, time.Duration, error)) loadResult {
	result := loadResult{name: name}
	var mu sync.Mutex
	jobs := make(chan int)
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				status, body, elapsed, err := call(i)
				mu.Lock()
				if err != nil || status != http.StatusOK {
					result.failures = append(result.failures, fmt.Sprintf("status %d err %v body %.200s", status, err, body))
				} else {
					result.latencies = append(result.latencies, elapsed)
				}
				mu.Unlock()
			}
		}()
	}
	for i := range requests {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	return result
}

func seedLoadCatalog(t *testing.T, pool *pgxpool.Pool) (areaIDs, propertyIDs, roomIDs []int64) {
	t.Helper()
	ctx := context.Background()
	var existing int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM properties WHERE slug LIKE 'load-%'`).Scan(&existing); err != nil {
		t.Fatal(err)
	}
	if existing < loadPropertyCount {
		ownerID := seedOwner(t, pool)
		for i := range loadAreaCount {
			areaIDs = append(areaIDs, seedArea(t, pool, fmt.Sprintf("Kota Beban %d", i), "city", i < 3, -6.2, 106.8))
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO properties (owner_id, area_id, name, slug, gender, latitude, longitude, is_active, moderation_status, is_flash_sale, flash_sale_discount)
			SELECT $1, ($2::bigint[])[1 + g % array_length($2::bigint[], 1)], 'Kost Beban ' || g, 'load-' || g || '-' || $3::bigint,
				(ARRAY['campur', 'putra', 'putri'])[1 + g % 3], -6.2 + random() * 0.3, 106.7 + random() * 0.3, true, 'approved', g % 7 = 0, 100000
			FROM generate_series(1, $4::int) g`, ownerID, areaIDs, time.Now().UnixNano(), loadPropertyCount-existing); err != nil {
			t.Fatalf("seed properties: %v", err)
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO rooms (property_id, name, price_monthly, available)
			SELECT p.id, 'Kamar ' || k, 800000 + (p.id % 40) * 50000 + k * 100000, 1 + (p.id % 4)
			FROM properties p CROSS JOIN generate_series(1, 2) k
			WHERE p.slug LIKE 'load-%' AND NOT EXISTS (SELECT 1 FROM rooms r WHERE r.property_id = p.id)`); err != nil {
			t.Fatalf("seed rooms: %v", err)
		}
		if _, err := pool.Exec(ctx, `ANALYZE properties; ANALYZE rooms`); err != nil {
			t.Fatal(err)
		}
	}
	collect := func(query string) []int64 {
		rows, err := pool.Query(ctx, query)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		var ids []int64
		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err != nil {
				t.Fatal(err)
			}
			ids = append(ids, id)
		}
		return ids
	}
	areaIDs = collect(`SELECT DISTINCT area_id FROM properties WHERE slug LIKE 'load-%'`)
	propertyIDs = collect(`SELECT id FROM properties WHERE slug LIKE 'load-%' ORDER BY random() LIMIT 1000`)
	roomIDs = collect(`SELECT r.id FROM rooms r JOIN properties p ON p.id = r.property_id WHERE p.slug LIKE 'load-%' AND r.available > 0 ORDER BY random() LIMIT 200`)
	return areaIDs, propertyIDs, roomIDs
}

// Inserts 10,000 properties, so TEST_DATABASE_URL must point at a dedicated database.
func TestLoadReadCheckoutAndUpload(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv, pool := newTestServerAndPool(t)
	areaIDs, propertyIDs, roomIDs := seedLoadCatalog(t, pool)

	search := runLoad("search", 400, loadWorkers, func(i int) (int, []byte, time.Duration, error) {
		body := fmt.Sprintf(`{"filters":{"area_id":%d,"rent_type":3,"is_available":%t},"sorting":{"field":"price","direction":"%s"},"limit":20,"offset":%d}`,
			areaIDs[rand.IntN(len(areaIDs))], i%2 == 0, []string{"+", "-"}[i%2], rand.IntN(5)*20)
		return loadRequest(srv, http.MethodPost, "/api/listings/search", "", body)
	})
	unfiltered := runLoad("search without area (API only)", 200, loadWorkers, func(i int) (int, []byte, time.Duration, error) {
		return loadRequest(srv, http.MethodPost, "/api/listings/search", "", fmt.Sprintf(`{"limit":20,"offset":%d}`, rand.IntN(10)*20))
	})
	home := runLoad("home flash sale search", 200, loadWorkers, func(int) (int, []byte, time.Duration, error) {
		return loadRequest(srv, http.MethodPost, "/api/listings/search", "", `{"filters":{"rent_type":3,"flash_sale":true},"limit":6,"offset":0}`)
	})
	detail := runLoad("detail", 400, loadWorkers, func(int) (int, []byte, time.Duration, error) {
		return loadRequest(srv, http.MethodGet, fmt.Sprintf("/api/listings/%d", propertyIDs[rand.IntN(len(propertyIDs))]), "", "")
	})

	_, tenantToken := bookingRegisterAndVerify(t, srv)
	startDate := money.TodayWIB().Format(time.DateOnly)
	bookingIDs := make([]int64, 0, 60)
	for _, roomID := range roomIDs[:60] {
		status, body, _, err := loadRequest(srv, http.MethodPost, "/api/booking/submit", tenantToken,
			fmt.Sprintf(`{"room_id":%d,"rent_type":"monthly","start_date":%q}`, roomID, startDate))
		if err != nil || status != http.StatusOK {
			t.Fatalf("submit: status %d err %v body %s", status, err, body)
		}
		var submitted struct {
			Booking struct {
				ID int64 `json:"id"`
			} `json:"booking"`
		}
		mustUnmarshal(t, body, &submitted)
		bookingIDs = append(bookingIDs, submitted.Booking.ID)
	}
	checkout := runLoad("checkout", len(bookingIDs), loadWorkers, func(i int) (int, []byte, time.Duration, error) {
		return loadRequest(srv, http.MethodPost, fmt.Sprintf("/api/booking/%d/checkout", bookingIDs[i]), tenantToken, "")
	})

	ownerID, ownerToken := newOwner(t, srv, pool)
	photoProperties := []int64{newOwnerProperty(t, pool, ownerID), newOwnerProperty(t, pool, ownerID), newOwnerProperty(t, pool, ownerID)}
	photo := photoJPEG(t)
	upload := runLoad("photo upload (presign + PUT + confirm)", 12, 4, func(i int) (int, []byte, time.Duration, error) {
		start := time.Now()
		status, body, _, err := loadRequest(srv, http.MethodPost, "/api/owner/uploads/presign", ownerToken,
			fmt.Sprintf(`{"content_type":"image/jpeg","size":%d}`, len(photo)))
		if err != nil || status != http.StatusOK {
			return status, body, 0, err
		}
		var presigned struct {
			Upload struct {
				URL       string `json:"url"`
				ObjectKey string `json:"object_key"`
			} `json:"upload"`
		}
		if err := json.Unmarshal(body, &presigned); err != nil {
			return 0, body, 0, err
		}
		put, err := http.NewRequest(http.MethodPut, presigned.Upload.URL, bytes.NewReader(photo))
		if err != nil {
			return 0, nil, 0, err
		}
		put.Header.Set("Content-Type", "image/jpeg")
		stored, err := http.DefaultClient.Do(put)
		if err != nil {
			return 0, nil, 0, err
		}
		stored.Body.Close()
		if stored.StatusCode != http.StatusOK {
			return stored.StatusCode, nil, 0, nil
		}
		status, body, _, err = loadRequest(srv, http.MethodPost, fmt.Sprintf("/api/owner/properties/%d/photos", photoProperties[i%len(photoProperties)]),
			ownerToken, fmt.Sprintf(`{"object_key":%q}`, presigned.Upload.ObjectKey))
		return status, body, time.Since(start), err
	})

	limits := map[string]time.Duration{"search": 300 * time.Millisecond, "home flash sale search": 300 * time.Millisecond,
		"detail": 300 * time.Millisecond, "checkout": 800 * time.Millisecond}
	for _, r := range []loadResult{search, unfiltered, home, detail, checkout, upload} {
		t.Logf("%-40s n=%d failures=%d p50=%v p95=%v max=%v", r.name, len(r.latencies), len(r.failures),
			r.percentile(0.50).Round(time.Millisecond), r.percentile(0.95).Round(time.Millisecond), r.percentile(1).Round(time.Millisecond))
		if len(r.failures) > 0 {
			t.Errorf("%s: %d failed requests, first: %s", r.name, len(r.failures), r.failures[0])
		}
		if limit, ok := limits[r.name]; ok && r.percentile(0.95) > limit {
			t.Errorf("%s: p95 %v exceeds the PRD §7 target %v", r.name, r.percentile(0.95), limit)
		}
	}
}
