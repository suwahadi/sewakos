package contract

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/suwahadi/sewakos/backend/internal/booking"
	"github.com/suwahadi/sewakos/backend/internal/config"
	"github.com/suwahadi/sewakos/backend/internal/platform/storage"
	"github.com/suwahadi/sewakos/backend/internal/worker"
)

func testStorageConfig(cfg config.Config) storage.Config {
	return storage.Config{
		Endpoint: cfg.StorageEndpoint, Region: cfg.StorageRegion, Bucket: cfg.StorageBucket,
		AccessKey: cfg.StorageAccessKey, SecretKey: cfg.StorageSecretKey, UseSSL: cfg.StorageUseSSL,
	}
}

func requireStorage(t *testing.T) *storage.Client {
	t.Helper()
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.StorageEndpoint == "" {
		t.Skip("STORAGE_ENDPOINT not set, skipping storage contract test")
	}
	client, err := storage.New(testStorageConfig(cfg))
	if err != nil {
		t.Fatalf("create storage client: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if _, err := client.List(ctx, "uploads/no-such-prefix/"); err != nil {
		t.Skipf("storage not reachable, skipping storage contract test: %v", err)
	}
	return client
}

func newOwner(t *testing.T, srv *httptest.Server, pool *pgxpool.Pool) (ownerID int64, token string) {
	t.Helper()
	phone, token := registerAndVerify(t, srv)
	if _, err := pool.Exec(context.Background(), `UPDATE users SET role = 'owner' WHERE phone = $1`, phone); err != nil {
		t.Fatalf("make owner: %v", err)
	}
	return currentUserID(t, srv, token), token
}

func newOwnerProperty(t *testing.T, pool *pgxpool.Pool, ownerID int64) int64 {
	t.Helper()
	areaID := seedArea(t, pool, "Kota Foto", "city", false, -6.2, 106.8)
	return seedProperty(t, pool, propertyFixture{ownerID: ownerID, areaID: areaID, name: "Kost Foto", gender: "campur", lat: -6.2, lng: 106.8})
}

func photoJPEG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2400, 1200))
	for y := 0; y < 1200; y += 4 {
		for x := 0; x < 2400; x += 4 {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 90, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func presignUpload(t *testing.T, srv *httptest.Server, token, contentType string, size int) (url, objectKey string) {
	t.Helper()
	resp, body := authedRequest(t, srv, http.MethodPost, "/api/owner/uploads/presign", token,
		fmt.Sprintf(`{"content_type":%q,"size":%d}`, contentType, size))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("presign: status = %d, body = %s", resp.StatusCode, body)
	}
	var parsed struct {
		Upload struct {
			URL       string `json:"url"`
			ObjectKey string `json:"object_key"`
		} `json:"upload"`
	}
	mustUnmarshal(t, body, &parsed)
	return parsed.Upload.URL, parsed.Upload.ObjectKey
}

func putObject(t *testing.T, url, contentType string, data []byte) int {
	t.Helper()
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", contentType)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("put object: %v", err)
	}
	resp.Body.Close()
	return resp.StatusCode
}

func uploadPhoto(t *testing.T, srv *httptest.Server, token string, data []byte) string {
	t.Helper()
	url, key := presignUpload(t, srv, token, "image/jpeg", len(data))
	if status := putObject(t, url, "image/jpeg", data); status != http.StatusOK {
		t.Fatalf("put presigned object: status = %d", status)
	}
	return key
}

func TestOwnerPresignFixture(t *testing.T) {
	requireStorage(t)
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv, pool := newTestServerAndPool(t)
	ownerID, token := newOwner(t, srv, pool)

	resp, body := authedRequest(t, srv, http.MethodPost, "/api/owner/uploads/presign", token, `{"content_type":"image/jpeg","size":345678}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, body)
	}
	want := `{
		"status": true,
		"meta": {"response_code": 200, "code": 200, "severity": "OK", "message": "OK"},
		"upload": {
			"method": "PUT", "url": "", "headers": {"Content-Type": "image/jpeg"},
			"object_key": "", "expires_at": ""
		}
	}`
	keyPattern := fmt.Sprintf(`uploads/%d/[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[0-9a-f]{4}-[0-9a-f]{12}\.jpg`, ownerID)
	AssertJSONMatches(t, body, []byte(want), map[string]*regexp.Regexp{
		"$.upload.url":        regexp.MustCompile(`^https?://[^/]+/[^/]+/` + keyPattern + `\?.*X-Amz-Signature=`),
		"$.upload.object_key": regexp.MustCompile(`^` + keyPattern + `$`),
		"$.upload.expires_at": regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\+00:00$`),
	})
}

func TestOwnerPhotoLifecycle(t *testing.T) {
	objectStore := requireStorage(t)
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv, pool := newTestServerAndPool(t)
	ownerID, token := newOwner(t, srv, pool)
	propertyID := newOwnerProperty(t, pool, ownerID)
	ctx := context.Background()

	original := photoJPEG(t)
	url, uploadKey := presignUpload(t, srv, token, "image/jpeg", len(original))
	if status := putObject(t, url, "image/png", original); status != http.StatusForbidden {
		t.Errorf("PUT with a different Content-Type: status = %d, want 403 (header is signed)", status)
	}
	if status := putObject(t, url, "image/jpeg", original); status != http.StatusOK {
		t.Fatalf("PUT presigned object: status = %d", status)
	}

	resp, body := authedRequest(t, srv, http.MethodPost, fmt.Sprintf("/api/owner/properties/%d/photos", propertyID), token,
		fmt.Sprintf(`{"object_key":%q,"room_id":null,"caption":"Tampak depan"}`, uploadKey))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("add photo: status = %d, body = %s", resp.StatusCode, body)
	}
	want := `{
		"status": true,
		"meta": {"response_code": 200, "code": 200, "severity": "OK", "message": "Foto ditambahkan."},
		"photo": {"id": 0, "url": "", "thumbnail_url": "", "caption": "Tampak depan", "room_id": null, "sort_order": 1}
	}`
	photoKey := fmt.Sprintf(`/properties/%d/[0-9a-f-]{36}`, propertyID)
	AssertJSONMatches(t, body, []byte(want), map[string]*regexp.Regexp{
		"$.photo.id":            regexp.MustCompile(`^\d+$`),
		"$.photo.url":           regexp.MustCompile(photoKey + `\.webp$`),
		"$.photo.thumbnail_url": regexp.MustCompile(photoKey + `_thumb\.webp$`),
	})

	var photoID int64
	var largeKey, thumbKey string
	var width, height int
	if err := pool.QueryRow(ctx, `SELECT id, object_key, thumb_key, width, height FROM property_photos WHERE property_id = $1`, propertyID).
		Scan(&photoID, &largeKey, &thumbKey, &width, &height); err != nil {
		t.Fatalf("load photo row: %v", err)
	}
	if width != 1600 || height != 800 {
		t.Errorf("stored size = %dx%d, want 1600x800", width, height)
	}
	if _, err := objectStore.Size(ctx, uploadKey); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("original upload still present (err = %v)", err)
	}
	for _, key := range []string{largeKey, thumbKey} {
		data, err := objectStore.Get(ctx, key)
		if err != nil {
			t.Fatalf("get %s: %v", key, err)
		}
		if !bytes.HasPrefix(data, []byte("RIFF")) || !bytes.Equal(data[8:12], []byte("WEBP")) {
			t.Errorf("%s is not WebP", key)
		}
	}

	resp, body = authedRequest(t, srv, http.MethodPatch, fmt.Sprintf("/api/owner/photos/%d", photoID), token, `{"caption":"Kamar mandi","sort_order":3}`)
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), `"Foto diperbarui."`) {
		t.Fatalf("update photo: status = %d, body = %s", resp.StatusCode, body)
	}
	var caption string
	var sortOrder int
	if err := pool.QueryRow(ctx, `SELECT caption, sort_order FROM property_photos WHERE id = $1`, photoID).Scan(&caption, &sortOrder); err != nil {
		t.Fatal(err)
	}
	if caption != "Kamar mandi" || sortOrder != 3 {
		t.Errorf("after update caption = %q, sort_order = %d", caption, sortOrder)
	}

	resp, body = authedRequest(t, srv, http.MethodDelete, fmt.Sprintf("/api/owner/photos/%d", photoID), token, "")
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), `"Foto dihapus."`) {
		t.Fatalf("delete photo: status = %d, body = %s", resp.StatusCode, body)
	}
	var queued int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM storage_deletions WHERE object_key IN ($1, $2)`, largeKey, thumbKey).Scan(&queued); err != nil {
		t.Fatal(err)
	}
	if queued != 2 {
		t.Fatalf("storage_deletions rows for the photo = %d, want 2", queued)
	}
	var audits int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM audit_logs WHERE entity_type = 'property_photo' AND entity_id = $1 AND actor_id = $2`, photoID, ownerID).Scan(&audits); err != nil {
		t.Fatal(err)
	}
	if audits != 3 {
		t.Errorf("audit rows = %d, want 3 (create, update, delete)", audits)
	}

	worker.PurgeStorage(ctx, worker.Deps{Booking: booking.Deps{DB: pool}, Storage: objectStore}, discardLogger())
	for _, key := range []string{largeKey, thumbKey} {
		if _, err := objectStore.Size(ctx, key); !errors.Is(err, storage.ErrNotFound) {
			t.Errorf("%s not purged (err = %v)", key, err)
		}
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM storage_deletions WHERE object_key IN ($1, $2)`, largeKey, thumbKey).Scan(&queued); err != nil {
		t.Fatal(err)
	}
	if queued != 0 {
		t.Errorf("storage_deletions rows left after purge = %d, want 0", queued)
	}
}

func TestOwnerPhotoRejections(t *testing.T) {
	objectStore := requireStorage(t)
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv, pool := newTestServerAndPool(t)
	ownerID, token := newOwner(t, srv, pool)
	propertyID := newOwnerProperty(t, pool, ownerID)
	otherOwnerID, _ := newOwner(t, srv, pool)
	otherPropertyID := newOwnerProperty(t, pool, otherOwnerID)
	otherRoomID := seedRoom(t, pool, otherPropertyID, "Kamar Lain", 1500000, 1)
	photosPath := fmt.Sprintf("/api/owner/properties/%d/photos", propertyID)

	t.Run("tenant is forbidden", func(t *testing.T) {
		_, tenantToken := registerAndVerify(t, srv)
		resp, body := authedRequest(t, srv, http.MethodPost, "/api/owner/uploads/presign", tenantToken, `{"content_type":"image/jpeg","size":10}`)
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("status = %d, body = %s", resp.StatusCode, body)
		}
	})

	t.Run("presign validation", func(t *testing.T) {
		resp, body := authedRequest(t, srv, http.MethodPost, "/api/owner/uploads/presign", token, `{"content_type":"image/gif","size":0}`)
		want := `{
			"status": false,
			"meta": {"response_code": 422, "code": 422, "severity": "WARNING", "message": "Validasi gagal."},
			"messages": {
				"content_type": ["The selected content type is invalid."],
				"size": ["The size field must be between 1 and 10485760."]
			}
		}`
		if resp.StatusCode != http.StatusUnprocessableEntity {
			t.Fatalf("status = %d", resp.StatusCode)
		}
		AssertJSONMatches(t, body, []byte(want), nil)
	})

	t.Run("another owner's property is not found", func(t *testing.T) {
		key := uploadPhoto(t, srv, token, photoJPEG(t))
		resp, _ := authedRequest(t, srv, http.MethodPost, fmt.Sprintf("/api/owner/properties/%d/photos", otherPropertyID), token, fmt.Sprintf(`{"object_key":%q}`, key))
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("status = %d, want 404", resp.StatusCode)
		}
	})

	t.Run("room of another property", func(t *testing.T) {
		resp, body := authedRequest(t, srv, http.MethodPost, photosPath, token, fmt.Sprintf(`{"object_key":"uploads/%d/x.jpg","room_id":%d}`, ownerID, otherRoomID))
		if resp.StatusCode != http.StatusUnprocessableEntity || !strings.Contains(string(body), "The selected room id is invalid.") {
			t.Errorf("status = %d, body = %s", resp.StatusCode, body)
		}
	})

	for name, objectKey := range map[string]string{
		"key outside the owner's upload prefix": fmt.Sprintf("uploads/%d/x.jpg", otherOwnerID),
		"key that was never uploaded":           fmt.Sprintf("uploads/%d/missing.jpg", ownerID),
	} {
		t.Run(name, func(t *testing.T) {
			resp, body := authedRequest(t, srv, http.MethodPost, photosPath, token, fmt.Sprintf(`{"object_key":%q}`, objectKey))
			if resp.StatusCode != http.StatusBadRequest || !strings.Contains(string(body), "File foto tidak ditemukan atau tidak valid.") {
				t.Errorf("status = %d, body = %s", resp.StatusCode, body)
			}
		})
	}

	t.Run("corrupt file", func(t *testing.T) {
		corrupt := append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, bytes.Repeat([]byte{7}, 256)...)
		key := uploadPhoto(t, srv, token, corrupt)
		resp, body := authedRequest(t, srv, http.MethodPost, photosPath, token, fmt.Sprintf(`{"object_key":%q}`, key))
		if resp.StatusCode != http.StatusBadRequest || !strings.Contains(string(body), "File foto tidak dapat diproses.") {
			t.Errorf("status = %d, body = %s", resp.StatusCode, body)
		}
		_ = objectStore.Remove(context.Background(), key)
	})

	t.Run("photo limit", func(t *testing.T) {
		fullPropertyID := newOwnerProperty(t, pool, ownerID)
		for range 30 {
			seedPropertyPhoto(t, pool, fullPropertyID)
		}
		resp, body := authedRequest(t, srv, http.MethodPost, fmt.Sprintf("/api/owner/properties/%d/photos", fullPropertyID), token, fmt.Sprintf(`{"object_key":"uploads/%d/x.jpg"}`, ownerID))
		if resp.StatusCode != http.StatusBadRequest || !strings.Contains(string(body), "Jumlah foto sudah mencapai batas.") {
			t.Errorf("status = %d, body = %s", resp.StatusCode, body)
		}
	})
}

type failingObjectStore struct{ err error }

func (s failingObjectStore) Remove(context.Context, string) error { return s.err }
func (s failingObjectStore) List(context.Context, string) ([]storage.Object, error) {
	return nil, s.err
}

func TestPurgeStorageBacksOffWhenStorageFails(t *testing.T) {
	_, pool := newTestServerAndPool(t)
	ctx := context.Background()
	key := fmt.Sprintf("properties/0/retry-%d.webp", time.Now().UnixNano())
	var id int64
	if err := pool.QueryRow(ctx, `INSERT INTO storage_deletions (object_key) VALUES ($1) RETURNING id`, key).Scan(&id); err != nil {
		t.Fatal(err)
	}

	worker.PurgeStorage(ctx, worker.Deps{Booking: booking.Deps{DB: pool}, Storage: failingObjectStore{err: errors.New("storage down")}}, discardLogger())

	var attempts int
	var lastError string
	var delaySeconds float64
	if err := pool.QueryRow(ctx, `SELECT attempts, last_error, EXTRACT(EPOCH FROM next_attempt_at - now())::float8 FROM storage_deletions WHERE id = $1`, id).
		Scan(&attempts, &lastError, &delaySeconds); err != nil {
		t.Fatalf("load row: %v", err)
	}
	if attempts != 1 || lastError != "storage down" {
		t.Errorf("attempts = %d, last_error = %q", attempts, lastError)
	}
	if delaySeconds < 110 || delaySeconds > 130 {
		t.Errorf("next attempt in %.0fs, want about 120s", delaySeconds)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM storage_deletions WHERE id = $1`, id); err != nil {
		t.Fatal(err)
	}
}

func TestStorageHousekeepingJobs(t *testing.T) {
	objectStore := requireStorage(t)
	_, pool := newTestServerAndPool(t)
	ctx := context.Background()
	deps := worker.Deps{Booking: booking.Deps{DB: pool}, Storage: objectStore}
	suffix := time.Now().UnixNano()
	staleUpload := fmt.Sprintf("uploads/0/stale-%d.jpg", suffix)
	orphan := fmt.Sprintf("properties/0/orphan-%d.webp", suffix)
	for _, key := range []string{staleUpload, orphan} {
		if err := objectStore.Put(ctx, key, []byte("x"), "application/octet-stream", ""); err != nil {
			t.Fatalf("put %s: %v", key, err)
		}
	}

	worker.CleanupUploads(ctx, deps, discardLogger(), time.Now())
	if _, err := objectStore.Size(ctx, staleUpload); err != nil {
		t.Errorf("fresh upload removed too early: %v", err)
	}
	worker.CleanupUploads(ctx, deps, discardLogger(), time.Now().Add(25*time.Hour))
	if _, err := objectStore.Size(ctx, staleUpload); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("upload older than 24h kept (err = %v)", err)
	}

	worker.ReconcileStorage(ctx, deps, discardLogger(), time.Now().Add(25*time.Hour))
	var queued int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM storage_deletions WHERE object_key = $1`, orphan).Scan(&queued); err != nil {
		t.Fatal(err)
	}
	if queued != 1 {
		t.Errorf("orphan queued %d times, want 1", queued)
	}
	_ = objectStore.Remove(ctx, orphan)
	if _, err := pool.Exec(ctx, `DELETE FROM storage_deletions WHERE object_key = $1`, orphan); err != nil {
		t.Fatal(err)
	}
}
