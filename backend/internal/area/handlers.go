package area

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
	"github.com/suwahadi/sewakos/backend/internal/response"
)

func ListHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		areaType := chi.URLParam(r, "type")
		if !validTypes[areaType] {
			response.NotFoundHandler(w, r)
			return
		}

		var isTop pgtype.Bool
		if r.URL.Query().Get("top") == "true" {
			isTop = pgtype.Bool{Bool: true, Valid: true}
		}

		rows, err := deps.Store.ListAreasByType(r.Context(), dbgen.ListAreasByTypeParams{
			Type: areaType, IsTop: isTop,
		})
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}

		areas := make([]map[string]any, len(rows))
		for i, row := range rows {
			areas[i] = areaResponse(row)
		}
		response.Success(w, http.StatusOK, "", map[string]any{"areas": areas})
	}
}
