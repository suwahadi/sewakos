package listing

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/suwahadi/sewakos/backend/internal/authctx"
	"github.com/suwahadi/sewakos/backend/internal/money"
	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
	"github.com/suwahadi/sewakos/backend/internal/response"
)

func parsePropertyID(r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "property"), 10, 64)
	return id, err == nil && id > 0
}

func unmarshalStringArray(raw []byte) []string {
	var out []string
	_ = json.Unmarshal(raw, &out)
	if out == nil {
		out = []string{}
	}
	return out
}

func unmarshalIntArray(raw []byte) []int {
	var out []int
	_ = json.Unmarshal(raw, &out)
	if out == nil {
		out = []int{}
	}
	return out
}

func DetailHandler(deps Deps, user *authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		propertyID, ok := parsePropertyID(r)
		if !ok {
			response.NotFoundHandler(w, r)
			return
		}
		ctx := r.Context()

		property, err := deps.Store.GetListingProperty(ctx, propertyID)
		if err != nil {
			response.NotFoundHandler(w, r)
			return
		}

		isFavorite := false
		if user != nil && user.Role == "tenant" {
			isFavorite, _ = deps.Store.ExistsFavorite(ctx, dbgen.ExistsFavoriteParams{
				UserID: user.UserID, PropertyID: propertyID,
			})
		}

		propertyPhotoRows, err := deps.Store.ListListingPropertyPhotos(ctx, propertyID)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		propertyPhotos := make([]map[string]any, len(propertyPhotoRows))
		for i, row := range propertyPhotoRows {
			propertyPhotos[i] = map[string]any{
				"id": row.ID, "url": photoURL(deps.StorageBaseURL, row.ObjectKey),
				"thumbnail_url": photoURL(deps.StorageBaseURL, row.ThumbKey),
				"caption":       nullableText(row.Caption),
			}
		}

		roomPhotoRows, err := deps.Store.ListListingRoomPhotos(ctx, propertyID)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		roomPhotosByRoom := map[int64][]map[string]any{}
		for _, row := range roomPhotoRows {
			roomID := row.RoomID.Int64
			roomPhotosByRoom[roomID] = append(roomPhotosByRoom[roomID], map[string]any{
				"id": row.ID, "url": photoURL(deps.StorageBaseURL, row.ObjectKey),
				"thumbnail_url": photoURL(deps.StorageBaseURL, row.ThumbKey),
				"caption":       nullableText(row.Caption),
			})
		}

		roomRows, err := deps.Store.ListListingRooms(ctx, propertyID)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		rooms := make([]map[string]any, len(roomRows))
		for i, row := range roomRows {
			photos := roomPhotosByRoom[row.ID]
			if photos == nil {
				photos = []map[string]any{}
			}
			rooms[i] = map[string]any{
				"id": row.ID, "name": row.Name,
				"price_monthly": row.PriceMonthly, "price_label": money.FormatRupiah(row.PriceMonthly),
				"available":                row.Available,
				"facilities":               unmarshalStringArray(row.Facilities),
				"monthly_duration_options": unmarshalIntArray(row.MonthlyDurationOptions),
				"photos":                   photos,
			}
		}

		var ratingAverage any
		if property.RatingCount > 0 {
			ratingAverage = property.RatingAverage
		}

		response.Success(w, http.StatusOK, "", map[string]any{
			"property": map[string]any{
				"id": property.ID, "name": property.Name, "slug": property.Slug,
				"gender": property.Gender, "description": property.Description,
				"address": property.Address, "latitude": property.Latitude, "longitude": property.Longitude,
				"is_active": property.IsActive, "is_flash_sale": property.IsFlashSale, "is_favorite": isFavorite,
				"owner": map[string]any{"name": property.OwnerName, "active_since": money.FormatTimestamp(property.OwnerCreatedAt.Time)},
				"area":  map[string]any{"name": property.AreaName},
				"photos": func() []map[string]any {
					if propertyPhotos == nil {
						return []map[string]any{}
					}
					return propertyPhotos
				}(),
				"rating": map[string]any{"average": ratingAverage, "count": property.RatingCount},
			},
			"rooms": rooms,
		})
	}
}
