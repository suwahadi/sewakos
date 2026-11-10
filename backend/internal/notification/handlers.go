package notification

import (
	"context"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/suwahadi/sewakos/backend/internal/authctx"
	"github.com/suwahadi/sewakos/backend/internal/money"
	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
	"github.com/suwahadi/sewakos/backend/internal/response"
)

const (
	defaultPage  = 1
	defaultLimit = 20
	maxLimit     = 50
)

// ReadStore is the subset of dbgen.Querier the read-side handlers need.
type ReadStore interface {
	ListUserNotifications(ctx context.Context, arg dbgen.ListUserNotificationsParams) ([]dbgen.ListUserNotificationsRow, error)
	CountUserNotifications(ctx context.Context, arg dbgen.CountUserNotificationsParams) (int64, error)
	CountUnreadNotifications(ctx context.Context, userID int64) (int64, error)
	GetUserNotification(ctx context.Context, id int64) (dbgen.GetUserNotificationRow, error)
	MarkUserNotificationRead(ctx context.Context, id int64) error
}

// Deps are the read-side handlers' dependencies.
type Deps struct {
	Store ReadStore
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
	if n > maxLimit {
		return maxLimit
	}
	return int32(n)
}

// ListHandler implements GET /notifications?category=&page=&limit=.
func ListHandler(deps Deps, user authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		page := pageParam(r)
		limit := limitParam(r)
		var category pgtype.Text
		if c := r.URL.Query().Get("category"); c != "" {
			category = pgtype.Text{String: c, Valid: true}
		}

		rows, err := deps.Store.ListUserNotifications(ctx, dbgen.ListUserNotificationsParams{
			UserID: user.UserID, Limit: limit, Offset: (page - 1) * limit, Category: category,
		})
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}

		items := make([]map[string]any, len(rows))
		for i, row := range rows {
			var link any
			if row.Link.Valid {
				link = row.Link.String
			}
			var readAt any
			if row.ReadAt.Valid {
				readAt = money.FormatTimestamp(row.ReadAt.Time)
			}
			items[i] = map[string]any{
				"id":         row.ID,
				"category":   row.Category,
				"title":      row.Title,
				"body":       row.Body,
				"link":       link,
				"read_at":    readAt,
				"created_at": money.FormatTimestamp(row.CreatedAt.Time),
			}
		}

		response.Success(w, http.StatusOK, "", map[string]any{
			"notifications": items,
			"current_page":  page,
		})
	}
}

// CounterHandler implements GET /notifications/counter.
func CounterHandler(deps Deps, user authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		unread, err := deps.Store.CountUnreadNotifications(r.Context(), user.UserID)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		response.Success(w, http.StatusOK, "", map[string]any{"unread": unread})
	}
}

// MarkReadHandler implements PATCH /notifications/{id}/read. Already-read is
// left untouched (idempotent), not re-stamped (API.md §6, SPEC-GO.md §9.3).
func MarkReadHandler(deps Deps, user authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			response.NotFoundHandler(w, r)
			return
		}

		row, err := deps.Store.GetUserNotification(ctx, id)
		if err != nil {
			response.NotFoundHandler(w, r)
			return
		}
		if row.UserID != user.UserID {
			response.Fail(w, http.StatusForbidden, response.MsgAccessDenied, nil)
			return
		}

		if err := deps.Store.MarkUserNotificationRead(ctx, id); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		response.Success(w, http.StatusOK, "", nil)
	}
}
