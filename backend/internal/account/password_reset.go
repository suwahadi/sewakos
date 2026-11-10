package account

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/suwahadi/sewakos/backend/internal/auth"
	"github.com/suwahadi/sewakos/backend/internal/otp"
	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
	"github.com/suwahadi/sewakos/backend/internal/platform/ratelimit"
	"github.com/suwahadi/sewakos/backend/internal/response"
	"github.com/suwahadi/sewakos/backend/internal/validation"
)

const msgOTPInvalid = "Kode verifikasi salah atau kedaluwarsa."

// PasswordResetHandler consumes a code issued by POST /auth/otp/request with
// for=reset_password directly — otp/check explicitly refuses to verify
// reset_password codes (SPEC-GO.md §6.4).
func PasswordResetHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		ctx := r.Context()

		v := validation.New()
		v.Field("destination", body["destination"]).Required().Regex(auth.PhoneRegex)
		v.Field("code", body["code"]).Required().String()
		v.Field("password", body["password"]).Required().String().MinStringLen(8)
		if !v.Valid() {
			response.Fail(w, http.StatusUnprocessableEntity, response.MsgValidationFailed, v.Errors())
			return
		}

		destination, _ := body["destination"].(string)
		code, _ := body["code"].(string)
		password, _ := body["password"].(string)

		if !ratelimit.EnforceHTTP(ctx, w, deps.Redis, "otp", destination, int64(deps.MaxPerMinute), time.Minute) {
			return
		}

		ok, err := deps.OTP.Check(ctx, otp.CodeKey("reset_password", destination), code)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if !ok {
			response.Fail(w, http.StatusBadRequest, msgOTPInvalid, nil)
			return
		}

		user, err := deps.Store.GetUserByPhone(ctx, destination)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}

		hash, err := auth.HashPassword(password)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if err := deps.Store.UpdateUserPasswordAndVerify(ctx, dbgen.UpdateUserPasswordAndVerifyParams{
			ID: user.ID, Password: pgtype.Text{String: hash, Valid: true},
		}); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if err := deps.Store.DeleteAllAccessTokensForUser(ctx, user.ID); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}

		response.Success(w, http.StatusOK, "Password berhasil diubah. Silakan login kembali.", nil)
	}
}
