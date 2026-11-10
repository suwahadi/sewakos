package account

import (
	"net/http"

	"github.com/suwahadi/sewakos/backend/internal/authctx"
	"github.com/suwahadi/sewakos/backend/internal/response"
)

// LogoutHandler revokes only the token this request used, leaving every
// other device's session intact.
func LogoutHandler(deps Deps, user authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := deps.Store.DeleteAccessToken(r.Context(), user.TokenID); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		response.Success(w, http.StatusOK, "Berhasil keluar.", nil)
	}
}
