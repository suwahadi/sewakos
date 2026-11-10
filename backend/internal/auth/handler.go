package auth

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/suwahadi/sewakos/backend/internal/response"
	"github.com/suwahadi/sewakos/backend/internal/validation"
)

// LoginHandler implements POST /auth/login (API.md §2.4). A malformed JSON
// body is treated as an empty one (SPEC-GO.md §5.4), so a decode error here
// simply leaves body empty rather than failing the request outright.
func LoginHandler(svc *LoginService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)

		v := validation.New()
		v.Field("phone_number", body["phone_number"]).Required()
		v.Field("password", body["password"]).Required()
		if !v.Valid() {
			response.Fail(w, http.StatusUnprocessableEntity, response.MsgValidationFailed, v.Errors())
			return
		}

		phone, _ := body["phone_number"].(string)
		password, _ := body["password"].(string)

		result, err := svc.Login(r.Context(), phone, password)
		switch {
		case errors.Is(err, ErrLoginFailed):
			response.FailWithNullMessages(w, http.StatusUnauthorized, response.MsgLoginFailed)
			return
		case errors.Is(err, ErrAccountSuspended):
			response.Fail(w, http.StatusForbidden, response.MsgAccountSuspended, nil)
			return
		case err != nil:
			response.InternalErrorHandler(w, r)
			return
		}

		response.Success(w, http.StatusOK, "Login berhasil.", map[string]any{
			"token": result.Token,
			"user": map[string]any{
				"id":          result.UserID,
				"name":        result.Name,
				"role":        result.Role,
				"is_verified": result.IsVerified,
			},
		})
	}
}
