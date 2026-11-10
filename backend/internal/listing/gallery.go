package listing

import (
	"net/http"

	"github.com/suwahadi/sewakos/backend/internal/response"
)

func GalleryHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		propertyID, ok := parsePropertyID(r)
		if !ok {
			response.NotFoundHandler(w, r)
			return
		}
		ctx := r.Context()

		if _, err := deps.Store.GetListingProperty(ctx, propertyID); err != nil {
			response.NotFoundHandler(w, r)
			return
		}

		roomRows, err := deps.Store.ListGalleryRooms(ctx, propertyID)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		photoRows, err := deps.Store.ListListingRoomPhotos(ctx, propertyID)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		photosByRoom := map[int64][]map[string]any{}
		for _, row := range photoRows {
			roomID := row.RoomID.Int64
			photosByRoom[roomID] = append(photosByRoom[roomID], map[string]any{
				"id": row.ID, "url": photoURL(deps.StorageBaseURL, row.ObjectKey),
				"thumbnail_url": photoURL(deps.StorageBaseURL, row.ThumbKey),
				"caption":       nullableText(row.Caption),
			})
		}

		gallery := make([]map[string]any, len(roomRows))
		for i, row := range roomRows {
			photos := photosByRoom[row.ID]
			if photos == nil {
				photos = []map[string]any{}
			}
			gallery[i] = map[string]any{
				"room_id": row.ID, "name": row.Name,
				"facilities": unmarshalStringArray(row.Facilities),
				"photos":     photos,
			}
		}

		response.Success(w, http.StatusOK, "", map[string]any{"gallery": gallery})
	}
}
