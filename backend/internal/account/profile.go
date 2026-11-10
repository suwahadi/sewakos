package account

import (
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/suwahadi/sewakos/backend/internal/authctx"
	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
	"github.com/suwahadi/sewakos/backend/internal/response"
	"github.com/suwahadi/sewakos/backend/internal/validation"
)

func GetMeHandler(deps Deps, user authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		row, err := deps.Store.GetUserByID(r.Context(), user.UserID)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		response.Success(w, http.StatusOK, "", map[string]any{
			"user": userResponse(row.ID, row.Name, row.Role, row.Phone, row.Email, row.PhoneVerifiedAt.Valid),
		})
	}
}

// PatchMeHandler's fields are both optional, but dbgen.UpdateUserProfileParams
// always sets both columns, so an omitted field is filled in from the user's
// current value before saving.
func PatchMeHandler(deps Deps, user authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		ctx := r.Context()

		current, err := deps.Store.GetUserByID(ctx, user.UserID)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}

		_, emailProvided := body["email"]
		emailValue, _ := body["email"].(string)
		emailTaken := false
		if emailProvided && emailValue != "" && emailValue != current.Email.String {
			emailTaken, _ = deps.Store.ExistsUserByEmail(ctx, pgtype.Text{String: emailValue, Valid: true})
		}

		v := validation.New()
		v.Field("name", body["name"]).Optional().String().MaxStringLen(255)
		if emailProvided && body["email"] != nil {
			v.Field("email", body["email"]).Email().Unique(emailTaken)
		}
		if !v.Valid() {
			response.Fail(w, http.StatusUnprocessableEntity, response.MsgValidationFailed, v.Errors())
			return
		}

		name := current.Name
		if n, ok := body["name"].(string); ok && n != "" {
			name = n
		}
		email := current.Email
		if emailProvided {
			if emailValue == "" || body["email"] == nil {
				email = pgtype.Text{}
			} else {
				email = pgtype.Text{String: emailValue, Valid: true}
			}
		}

		if err := deps.Store.UpdateUserProfile(ctx, dbgen.UpdateUserProfileParams{
			ID: user.UserID, Name: name, Email: email,
		}); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}

		response.Success(w, http.StatusOK, "Profil diperbarui.", map[string]any{
			"user": userResponse(current.ID, name, current.Role, current.Phone, email, current.PhoneVerifiedAt.Valid),
		})
	}
}
