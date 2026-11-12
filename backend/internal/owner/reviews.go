package owner

import (
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/suwahadi/sewakos/backend/internal/authctx"
	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
	"github.com/suwahadi/sewakos/backend/internal/response"
	"github.com/suwahadi/sewakos/backend/internal/review"
)

func ListReviewsHandler(deps Deps, user authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, limit := pageParam(r), limitParam(r)
		propertyID, parseErr := strconv.ParseInt(r.URL.Query().Get("property_id"), 10, 64)
		rows, err := deps.Store.ListOwnerReviews(r.Context(), dbgen.ListOwnerReviewsParams{
			OwnerID: user.UserID, PropertyID: pgtype.Int8{Int64: propertyID, Valid: parseErr == nil},
			RowLimit: limit, RowOffset: (page - 1) * limit,
		})
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		reviews := make([]map[string]any, len(rows))
		for i, row := range rows {
			item := review.Body(row.ID, row.Rating, row.Comment, review.MaskTenantName(row.TenantName), row.RoomName, row.CreatedAt.Time)
			item["property_id"] = row.PropertyID
			item["is_hidden"] = row.IsHidden
			reviews[i] = item
		}
		response.Success(w, http.StatusOK, "", map[string]any{"reviews": reviews, "current_page": page})
	}
}
