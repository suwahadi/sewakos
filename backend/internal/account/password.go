package account

import (
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"

	"github.com/suwahadi/sewakos/backend/internal/auth"
	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
	"github.com/suwahadi/sewakos/backend/internal/response"
	"github.com/suwahadi/sewakos/backend/internal/validation"
)

// PutPasswordHandler revokes every *other* device's session, keeping the one
// making this request alive, unlike a forced reset (PasswordResetHandler)
// which revokes everything.
func PutPasswordHandler(deps Deps, user AuthenticatedUser) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		ctx := r.Context()

		v := validation.New()
		v.Field("current_password", body["current_password"]).Required().String()
		v.Field("password", body["password"]).Required().String().MinStringLen(8)
		if !v.Valid() {
			response.Fail(w, http.StatusUnprocessableEntity, response.MsgValidationFailed, v.Errors())
			return
		}

		current, _ := body["current_password"].(string)
		stored, err := deps.Store.GetUserPasswordByID(ctx, user.UserID)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if !stored.Valid || bcrypt.CompareHashAndPassword([]byte(stored.String), []byte(current)) != nil {
			response.Fail(w, http.StatusUnprocessableEntity, response.MsgValidationFailed, map[string][]string{
				"current_password": {"The current password is incorrect."},
			})
			return
		}

		newPassword, _ := body["password"].(string)
		hash, err := auth.HashPassword(newPassword)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if err := deps.Store.UpdateUserPassword(ctx, dbgen.UpdateUserPasswordParams{
			ID: user.UserID, Password: pgtype.Text{String: hash, Valid: true},
		}); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if err := deps.Store.DeleteOtherAccessTokensForUser(ctx, dbgen.DeleteOtherAccessTokensForUserParams{
			UserID: user.UserID, ID: user.TokenID,
		}); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}

		response.Success(w, http.StatusOK, "Password berhasil diubah.", nil)
	}
}
