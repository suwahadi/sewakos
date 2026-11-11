package owner

import (
	"context"
	"encoding/json"
	"log/slog"
	"math"
	"net/http"
	"net/netip"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/suwahadi/sewakos/backend/internal/audit"
	"github.com/suwahadi/sewakos/backend/internal/authctx"
	"github.com/suwahadi/sewakos/backend/internal/media"
	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
)

type Storage interface {
	PresignPut(ctx context.Context, key, contentType string, expiry time.Duration) (string, error)
	Size(ctx context.Context, key string) (int64, error)
	Get(ctx context.Context, key string) ([]byte, error)
	Put(ctx context.Context, key string, data []byte, contentType, cacheControl string) error
	Remove(ctx context.Context, key string) error
}

type ImageProcessor interface {
	Process(ctx context.Context, data []byte) (media.Image, error)
}

type TxBeginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

type Deps struct {
	Store          dbgen.Querier
	DB             TxBeginner
	Storage        Storage
	Images         ImageProcessor
	StorageBaseURL string
	UploadMaxBytes int64
	AppPublicURL   string
}

func photoURL(base, key string) string {
	return base + "/" + key
}

func parseID(r *http.Request, param string) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, param), 10, 64)
	return id, err == nil && id > 0
}

func decodeBody(r *http.Request) map[string]any {
	var body map[string]any
	_ = json.NewDecoder(r.Body).Decode(&body)
	if body == nil {
		body = map[string]any{}
	}
	return body
}

func wholeNumber(value any) (int64, bool) {
	n, ok := value.(float64)
	if !ok || n != math.Trunc(n) {
		return 0, false
	}
	return int64(n), true
}

func optionalText(value any) pgtype.Text {
	s, ok := value.(string)
	return pgtype.Text{String: s, Valid: ok && s != ""}
}

func nullableText(t pgtype.Text) any {
	if !t.Valid {
		return nil
	}
	return t.String
}

func nullableInt8(n pgtype.Int8) any {
	if !n.Valid {
		return nil
	}
	return n.Int64
}

func toPgTimestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

func toPgDate(t time.Time) pgtype.Date {
	return pgtype.Date{Time: t, Valid: true}
}

func recordAudit(ctx context.Context, store audit.Store, r *http.Request, user authctx.User, action, entityType string, entityID int64, before, after map[string]any) error {
	ownerID, role, userAgent := user.UserID, user.Role, r.UserAgent()
	params := audit.Params{
		ActorID: &ownerID, ActorRole: &role, Action: action, EntityType: entityType, EntityID: entityID,
		OwnerID: &ownerID, Before: before, After: after, UserAgent: &userAgent,
	}
	if ip, err := netip.ParseAddr(user.IP); err == nil {
		params.IP = &ip
	}
	_, err := audit.Record(ctx, store, params)
	return err
}

func queueStorageDeletion(ctx context.Context, store dbgen.Querier, keys ...string) {
	for _, key := range keys {
		if err := store.QueueStorageDeletion(ctx, key); err != nil {
			slog.Error("queue storage deletion", "object_key", key, "error", err)
		}
	}
}
