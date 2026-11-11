package contract

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/suwahadi/sewakos/backend/internal/demoseed"
	"github.com/suwahadi/sewakos/backend/internal/media"
)

func TestDemoSeedCreatesListedPropertiesWithProcessedPhotos(t *testing.T) {
	srv, pool := newTestServerAndPool(t)
	store := requireStorage(t)
	ctx := context.Background()

	photo := photoJPEG(t)
	var downloads atomic.Int32
	unsplash := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		downloads.Add(1)
		if !strings.HasPrefix(r.URL.Path, "/photo-") || r.URL.Query().Get("fm") != "jpg" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write(photo)
	}))
	t.Cleanup(unsplash.Close)

	deps := demoseed.Deps{
		DB: pool, Storage: store, Logger: discardLogger(), HTTP: unsplash.Client(), PhotoBaseURL: unsplash.URL,
		Images: media.NewProcessor(media.Config{MaxPixels: 40_000_000, LargePx: 1600, ThumbPx: 400, LargeQuality: 80, ThumbQuality: 75, Concurrency: 2}),
	}
	prefix := fmt.Sprintf("demo-test-%d", time.Now().UnixNano())
	created, err := demoseed.Run(ctx, deps, demoseed.Options{Count: 3, SlugPrefix: prefix})
	if err != nil || created != 3 {
		t.Fatalf("first run: created %d, err %v", created, err)
	}
	if downloads.Load() != 27 {
		t.Errorf("photo downloads = %d, want one per Unsplash source (27)", downloads.Load())
	}
	if created, err := demoseed.Run(ctx, deps, demoseed.Options{Count: 3, SlugPrefix: prefix}); err != nil || created != 0 {
		t.Errorf("second run: created %d, err %v, want 0 (idempotent)", created, err)
	}

	var propertyID int64
	var status, ownerRole string
	var hasBank bool
	if err := pool.QueryRow(ctx, `
		SELECT p.id, p.moderation_status, o.role, EXISTS (SELECT 1 FROM owner_bank_accounts b WHERE b.user_id = o.id)
		FROM properties p JOIN users o ON o.id = p.owner_id WHERE p.slug = $1`, prefix+"-001").Scan(&propertyID, &status, &ownerRole, &hasBank); err != nil {
		t.Fatal(err)
	}
	if status != "approved" || ownerRole != "owner" || !hasBank {
		t.Errorf("property status %q, owner role %q, bank account %v", status, ownerRole, hasBank)
	}

	var rooms, propertyPhotos, roomPhotos int
	var firstCaption, largeKey, thumbKey string
	var facilities []byte
	if err := pool.QueryRow(ctx, `
		SELECT (SELECT count(*) FROM rooms WHERE property_id = $1),
			(SELECT count(*) FROM property_photos WHERE property_id = $1 AND room_id IS NULL),
			(SELECT count(*) FROM property_photos WHERE property_id = $1 AND room_id IS NOT NULL),
			(SELECT caption FROM property_photos WHERE property_id = $1 AND room_id IS NULL ORDER BY sort_order LIMIT 1),
			(SELECT object_key FROM property_photos WHERE property_id = $1 ORDER BY sort_order LIMIT 1),
			(SELECT thumb_key FROM property_photos WHERE property_id = $1 ORDER BY sort_order LIMIT 1),
			(SELECT facilities FROM rooms WHERE property_id = $1 ORDER BY id LIMIT 1)`, propertyID).
		Scan(&rooms, &propertyPhotos, &roomPhotos, &firstCaption, &largeKey, &thumbKey, &facilities); err != nil {
		t.Fatal(err)
	}
	var facilityList []string
	if err := json.Unmarshal(facilities, &facilityList); err != nil || len(facilityList) == 0 {
		t.Errorf("facilities %s: %v", facilities, err)
	}
	if rooms < 1 || propertyPhotos != 4 || roomPhotos < rooms || firstCaption != "Tampak depan" {
		t.Errorf("rooms %d, property photos %d, room photos %d, first caption %q", rooms, propertyPhotos, roomPhotos, firstCaption)
	}
	for _, key := range []string{largeKey, thumbKey} {
		if !strings.HasPrefix(key, fmt.Sprintf("properties/%d/", propertyID)) || !strings.HasSuffix(key, ".webp") {
			t.Errorf("object key %q", key)
		}
		if size, err := store.Size(ctx, key); err != nil || size == 0 {
			t.Errorf("stored object %s: size %d, err %v", key, size, err)
		}
	}

	resp, body := bookingAuthedRequest(t, srv, http.MethodGet, fmt.Sprintf("/api/listings/%d", propertyID), "", "")
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), largeKey) {
		t.Errorf("listing detail: status %d, body %.300s", resp.StatusCode, body)
	}
}
