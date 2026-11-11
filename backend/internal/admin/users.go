package admin

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"

	"github.com/suwahadi/sewakos/backend/internal/authctx"
	"github.com/suwahadi/sewakos/backend/internal/money"
	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
	"github.com/suwahadi/sewakos/backend/internal/response"
	"github.com/suwahadi/sewakos/backend/internal/validation"
)

func userBody(u dbgen.AdminLockUserRow) map[string]any {
	return map[string]any{
		"id": u.ID, "name": u.Name, "role": u.Role, "phone_number": u.Phone, "email": nullableText(u.Email),
		"is_verified": u.PhoneVerifiedAt.Valid, "suspended_at": nullableTimestamp(u.SuspendedAt),
		"created_at": money.FormatTimestamp(u.CreatedAt.Time),
	}
}

func ListUsersHandler(deps Deps, _ authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := queryText(r, "q")
		v := validation.New()
		v.Field("q", r.URL.Query().Get("q")).Optional().MinStringLen(3)
		if !v.Valid() {
			response.Fail(w, http.StatusUnprocessableEntity, response.MsgValidationFailed, v.Errors())
			return
		}
		page, limit := pageParam(r), limitParam(r)
		rows, err := deps.Store.AdminListUsers(r.Context(), dbgen.AdminListUsersParams{
			Role: queryText(r, "role"), Q: q, RowLimit: limit, RowOffset: (page - 1) * limit,
		})
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		users := make([]map[string]any, len(rows))
		for i, row := range rows {
			users[i] = userBody(dbgen.AdminLockUserRow(row))
		}
		response.Success(w, http.StatusOK, "", map[string]any{"users": users, "current_page": page})
	}
}

func GetUserHandler(deps Deps, _ authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := parseID(r, "user")
		if !ok {
			response.NotFoundHandler(w, r)
			return
		}
		ctx := r.Context()
		tx, err := deps.DB.Begin(ctx)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		defer func() { _ = tx.Rollback(ctx) }()
		user, err := dbgen.New(tx).AdminLockUser(ctx, userID)
		if errors.Is(err, pgx.ErrNoRows) {
			response.NotFoundHandler(w, r)
			return
		}
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		response.Success(w, http.StatusOK, "", map[string]any{"user": userBody(user)})
	}
}

func SuspendUserHandler(deps Deps, actor authctx.User) http.HandlerFunc {
	return setSuspension(deps, actor, true)
}

func UnsuspendUserHandler(deps Deps, actor authctx.User) http.HandlerFunc {
	return setSuspension(deps, actor, false)
}

func setSuspension(deps Deps, actor authctx.User, suspend bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := parseID(r, "user")
		if !ok {
			response.NotFoundHandler(w, r)
			return
		}
		ctx := r.Context()
		tx, err := deps.DB.Begin(ctx)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		defer func() { _ = tx.Rollback(ctx) }()
		queries := dbgen.New(tx)

		user, err := queries.AdminLockUser(ctx, userID)
		if errors.Is(err, pgx.ErrNoRows) {
			response.NotFoundHandler(w, r)
			return
		}
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if suspend && user.Role == "admin" {
			response.Fail(w, http.StatusBadRequest, "Admin tidak dapat dinonaktifkan.", nil)
			return
		}
		if err := queries.SetUserSuspended(ctx, dbgen.SetUserSuspendedParams{ID: user.ID, Suspended: suspend}); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		action, message := "user.unsuspend", "Pengguna diaktifkan kembali."
		if suspend {
			action, message = "user.suspend", "Pengguna dinonaktifkan."
			if err := queries.DeleteAllAccessTokensForUser(ctx, user.ID); err != nil {
				response.InternalErrorHandler(w, r)
				return
			}
		}
		if err := recordAudit(ctx, queries, r, actor, action, "user", user.ID, nil,
			map[string]any{"suspended": user.SuspendedAt.Valid}, map[string]any{"suspended": suspend}); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if err := tx.Commit(ctx); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		response.Success(w, http.StatusOK, message, nil)
	}
}
