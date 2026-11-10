package account

import (
	"net/http"

	"github.com/suwahadi/sewakos/backend/internal/response"
)

// AuthenticatedUser is the authenticated caller, extracted from request
// context by the router (this package doesn't depend on internal/server, so
// it carries the two fields handlers need instead of server.AuthUser itself).
type AuthenticatedUser struct {
	UserID  int64
	TokenID int64
}

// LogoutHandler revokes only the token this request used, leaving every
// other device's session intact.
func LogoutHandler(deps Deps, user AuthenticatedUser) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := deps.Store.DeleteAccessToken(r.Context(), user.TokenID); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		response.Success(w, http.StatusOK, "Berhasil keluar.", nil)
	}
}
