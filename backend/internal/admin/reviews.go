package admin

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/suwahadi/sewakos/backend/internal/authctx"
	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
	"github.com/suwahadi/sewakos/backend/internal/response"
	"github.com/suwahadi/sewakos/backend/internal/review"
	"github.com/suwahadi/sewakos/backend/internal/validation"
)

func queryBool(r *http.Request, key string) pgtype.Bool {
	value, err := strconv.ParseBool(r.URL.Query().Get(key))
	return pgtype.Bool{Bool: value, Valid: err == nil}
}

func ListReviewsHandler(deps Deps, _ authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, limit := pageParam(r), limitParam(r)
		rows, err := deps.Store.AdminListReviews(r.Context(), dbgen.AdminListReviewsParams{
			IsHidden: queryBool(r, "is_hidden"), PropertyID: queryID(r, "property_id"),
			RowLimit: limit, RowOffset: (page - 1) * limit,
		})
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		reviews := make([]map[string]any, len(rows))
		for i, row := range rows {
			item := review.Body(row.ID, row.Rating, row.Comment, row.TenantName, row.RoomName, row.CreatedAt.Time)
			item["property_id"] = row.PropertyID
			item["is_hidden"] = row.IsHidden
			item["hidden_reason"] = nullableText(row.HiddenReason)
			reviews[i] = item
		}
		response.Success(w, http.StatusOK, "", map[string]any{"reviews": reviews, "current_page": page})
	}
}

func setReviewHidden(deps Deps, user authctx.User, w http.ResponseWriter, r *http.Request, hidden bool, reason pgtype.Text, action, message string) {
	reviewID, ok := parseID(r, "review")
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
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()
	queries := dbgen.New(tx)

	current, err := queries.LockReviewForModeration(ctx, reviewID)
	if errors.Is(err, pgx.ErrNoRows) {
		response.NotFoundHandler(w, r)
		return
	}
	if err != nil {
		response.InternalErrorHandler(w, r)
		return
	}
	if err := queries.SetReviewHidden(ctx, dbgen.SetReviewHiddenParams{ID: reviewID, IsHidden: hidden, HiddenReason: reason}); err != nil {
		response.InternalErrorHandler(w, r)
		return
	}
	if err := recordAudit(ctx, queries, r, user, action, "review", reviewID, &current.OwnerID,
		map[string]any{"is_hidden": current.IsHidden, "hidden_reason": nullableText(current.HiddenReason)},
		map[string]any{"is_hidden": hidden, "hidden_reason": nullableText(reason)}); err != nil {
		response.InternalErrorHandler(w, r)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		response.InternalErrorHandler(w, r)
		return
	}
	committed = true
	deps.Ratings.Invalidate(ctx, current.PropertyID)

	response.Success(w, http.StatusOK, message, nil)
}

func HideReviewHandler(deps Deps, user authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body := decodeBody(r)
		v := validation.New()
		v.Field("reason", body["reason"]).Required().String().MaxStringLen(500)
		if !v.Valid() {
			response.Fail(w, http.StatusUnprocessableEntity, response.MsgValidationFailed, v.Errors())
			return
		}
		reason, _ := body["reason"].(string)
		setReviewHidden(deps, user, w, r, true, pgtype.Text{String: reason, Valid: true}, "review.hide", "Ulasan disembunyikan.")
	}
}

func UnhideReviewHandler(deps Deps, user authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setReviewHidden(deps, user, w, r, false, pgtype.Text{}, "review.unhide", "Ulasan ditampilkan kembali.")
	}
}
