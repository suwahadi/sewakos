package search

import (
	"net/http"

	"github.com/suwahadi/sewakos/backend/internal/response"
	"github.com/suwahadi/sewakos/backend/internal/validation"
)

func SuggestionHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		keyword := r.URL.Query().Get("keyword")

		v := validation.New()
		var keywordValue any
		if keyword != "" {
			keywordValue = keyword
		}
		v.Field("keyword", keywordValue).Required().String().MinStringLen(3)
		if !v.Valid() {
			response.Fail(w, http.StatusUnprocessableEntity, response.MsgValidationFailed, v.Errors())
			return
		}

		ctx := r.Context()

		areaRows, err := deps.Store.SuggestAreas(ctx, keyword)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		landingRows, err := deps.Store.SuggestLandings(ctx, keyword)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		roomTypeRows, err := deps.Store.SuggestRoomTypes(ctx, keyword)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}

		areas := make([]map[string]any, len(areaRows))
		for i, row := range areaRows {
			areas[i] = map[string]any{
				"title": row.Name, "area": row.Name,
				"latitude": row.Latitude, "longitude": row.Longitude,
				"geocode_id": row.ID, "type": "area",
			}
		}
		landings := make([]map[string]any, len(landingRows))
		for i, row := range landingRows {
			landings[i] = map[string]any{
				"id": row.ID, "title": row.Name, "slug": row.Slug, "gender": row.Gender,
			}
		}
		roomTypes := make([]map[string]any, len(roomTypeRows))
		for i, row := range roomTypeRows {
			roomTypes[i] = map[string]any{
				"id": row.ID, "property_id": row.PropertyID, "title": row.Name, "slug": row.Slug,
			}
		}

		response.Success(w, http.StatusOK, "", map[string]any{
			"suggestion": map[string]any{
				"areas": areas, "area_source": "db",
				"landings": landings, "room_types": roomTypes,
			},
		})
	}
}
