package favorite

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/suwahadi/sewakos/backend/internal/authctx"
	"github.com/suwahadi/sewakos/backend/internal/money"
	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
	"github.com/suwahadi/sewakos/backend/internal/response"
)

const (
	defaultLimit = 20
	maxLimit     = 50
)

type Deps struct {
	Store          dbgen.Querier
	StorageBaseURL string
}

func parsePropertyID(r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "property"), 10, 64)
	return id, err == nil && id > 0
}

func pageParam(r *http.Request) int32 {
	n, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || n < 1 {
		return 1
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

func AddHandler(deps Deps, user authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		propertyID, ok := parsePropertyID(r)
		if !ok {
			response.NotFoundHandler(w, r)
			return
		}
		ctx := r.Context()
		listed, err := deps.Store.ExistsListedProperty(ctx, propertyID)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if !listed {
			response.NotFoundHandler(w, r)
			return
		}
		if err := deps.Store.AddFavorite(ctx, dbgen.AddFavoriteParams{UserID: user.UserID, PropertyID: propertyID}); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		response.Success(w, http.StatusOK, "Ditambahkan ke favorit.", nil)
	}
}

func RemoveHandler(deps Deps, user authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		propertyID, ok := parsePropertyID(r)
		if !ok {
			response.NotFoundHandler(w, r)
			return
		}
		ctx := r.Context()
		exists, err := deps.Store.ExistsProperty(ctx, propertyID)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if !exists {
			response.NotFoundHandler(w, r)
			return
		}
		if err := deps.Store.RemoveFavorite(ctx, dbgen.RemoveFavoriteParams{UserID: user.UserID, PropertyID: propertyID}); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		response.Success(w, http.StatusOK, "Dihapus dari favorit.", nil)
	}
}

func ListHandler(deps Deps, user authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, limit := pageParam(r), limitParam(r)
		rows, err := deps.Store.ListFavorites(r.Context(), dbgen.ListFavoritesParams{
			UserID: user.UserID, RowLimit: limit, RowOffset: (page - 1) * limit,
		})
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		favorites := make([]map[string]any, len(rows))
		for i, row := range rows {
			var thumbnailURL, minPrice, minPriceLabel, ratingAverage any
			if row.ThumbnailKey != "" {
				thumbnailURL = deps.StorageBaseURL + "/" + row.ThumbnailKey
			}
			if row.HasMonthlyPrice {
				minPrice, minPriceLabel = row.MinPriceMonthly, money.FormatRupiah(row.MinPriceMonthly)
			}
			if row.RatingCount > 0 {
				ratingAverage = row.RatingAverage
			}
			favorites[i] = map[string]any{
				"id": row.ID, "name": row.Name, "gender": row.Gender,
				"area":              map[string]any{"name": row.AreaName},
				"thumbnail_url":     thumbnailURL,
				"min_price_monthly": minPrice, "min_price_monthly_label": minPriceLabel,
				"rating_average": ratingAverage, "is_available": row.IsAvailable,
			}
		}
		response.Success(w, http.StatusOK, "", map[string]any{"favorites": favorites, "current_page": page})
	}
}
