// Package abtest implements GET /config/ab-test/{experiment}?visitor_id=
// (API.md §7, SPEC-GO.md §9.4): a deterministic variant assignment with no
// storage — the same visitor_id always resolves to the same variant.
package abtest

import (
	"encoding/json"
	"hash/crc32"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/suwahadi/sewakos/backend/internal/response"
	"github.com/suwahadi/sewakos/backend/internal/validation"
)

// experiments lists every known experiment and its variants, in the fixed
// order SPEC-GO.md §9.4 requires: changing the order moves visitors to a
// different variant, so entries are only ever appended, never reordered.
var experiments = map[string][]string{
	"discovery-communication": {"control", "chat-first", "booking-first"},
	"home-hero":               {"image", "video"},
}

func Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		experiment := chi.URLParam(r, "experiment")
		variants, ok := experiments[experiment]
		if !ok {
			response.NotFoundHandler(w, r)
			return
		}

		visitorID := r.URL.Query().Get("visitor_id")
		if visitorID == "" {
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if v, ok := body["visitor_id"].(string); ok {
				visitorID = v
			}
		}

		v := validation.New()
		var visitorIDValue any
		if visitorID != "" {
			visitorIDValue = visitorID
		}
		v.Field("visitor_id", visitorIDValue).Required().String()
		if !v.Valid() {
			response.Fail(w, http.StatusUnprocessableEntity, response.MsgValidationFailed, v.Errors())
			return
		}

		index := crc32.ChecksumIEEE([]byte(experiment+"|"+visitorID)) % uint32(len(variants))
		response.Success(w, http.StatusOK, "", map[string]any{
			"experiment": experiment, "variant": variants[index],
		})
	}
}
