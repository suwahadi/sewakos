package admin

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/netip"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/suwahadi/sewakos/backend/internal/audit"
	"github.com/suwahadi/sewakos/backend/internal/authctx"
	"github.com/suwahadi/sewakos/backend/internal/money"
	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
)

const (
	defaultPage  = 1
	defaultLimit = 20
	maxLimit     = 50
)

type TxBeginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

type AdminFeeCache interface {
	Invalidate()
}

type RatingCache interface {
	Invalidate(ctx context.Context, propertyID int64)
}

type Deps struct {
	Store          dbgen.Querier
	DB             TxBeginner
	AdminFee       AdminFeeCache
	Ratings        RatingCache
	StorageBaseURL string
	AppPublicURL   string
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

func pageParam(r *http.Request) int32 {
	n, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || n < 1 {
		return defaultPage
	}
	return int32(n)
}

func limitParam(r *http.Request) int32 {
	n, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil || n < 1 {
		return defaultLimit
	}
	return int32(min(n, maxLimit))
}

func queryText(r *http.Request, key string) pgtype.Text {
	value := r.URL.Query().Get(key)
	return pgtype.Text{String: value, Valid: value != ""}
}

func queryID(r *http.Request, key string) pgtype.Int8 {
	n, err := strconv.ParseInt(r.URL.Query().Get(key), 10, 64)
	return pgtype.Int8{Int64: n, Valid: err == nil}
}

func wholeNumber(value any) (int64, bool) {
	n, ok := value.(float64)
	if !ok || n != math.Trunc(n) {
		return 0, false
	}
	return int64(n), true
}

func nullableText(t pgtype.Text) any {
	if !t.Valid {
		return nil
	}
	return t.String
}

func nullableTimestamp(t pgtype.Timestamptz) any {
	if !t.Valid {
		return nil
	}
	return money.FormatExpiresAt(t.Time)
}

func recordAudit(ctx context.Context, store audit.Store, r *http.Request, user authctx.User, action, entityType string, entityID int64, ownerID *int64, before, after map[string]any) error {
	actorID, role, userAgent := user.UserID, user.Role, r.UserAgent()
	params := audit.Params{
		ActorID: &actorID, ActorRole: &role, Action: action, EntityType: entityType, EntityID: entityID,
		OwnerID: ownerID, Before: before, After: after, UserAgent: &userAgent,
	}
	if ip, err := netip.ParseAddr(user.IP); err == nil {
		params.IP = &ip
	}
	_, err := audit.Record(ctx, store, params)
	return err
}
