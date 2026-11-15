package account

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/suwahadi/sewakos/backend/internal/auth"
	"github.com/suwahadi/sewakos/backend/internal/authctx"
	"github.com/suwahadi/sewakos/backend/internal/otp"
	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
	"github.com/suwahadi/sewakos/backend/internal/platform/ratelimit"
	"github.com/suwahadi/sewakos/backend/internal/response"
	"github.com/suwahadi/sewakos/backend/internal/validation"
)

func PhoneRequestHandler(deps Deps, user authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		ctx := r.Context()

		phone, _ := body["phone_number"].(string)
		taken := false
		if phone != "" {
			taken, _ = deps.Store.ExistsUserByPhone(ctx, phone)
		}

		v := validation.New()
		v.Field("phone_number", body["phone_number"]).Required().Regex(auth.PhoneRegex).Unique(taken)
		if !v.Valid() {
			response.Fail(w, http.StatusUnprocessableEntity, response.MsgValidationFailed, v.Errors())
			return
		}

		if !ratelimit.EnforceHTTP(ctx, w, deps.Redis, "otp", phone, int64(deps.MaxPerMinute), time.Minute) {
			return
		}

		result, err := deps.OTP.Request(ctx, otp.RequestParams{
			Channel: otp.ChannelWhatsApp, CodeKey: otp.ChangePhoneCodeKey(user.UserID, phone), CooldownKey: otp.CooldownKey(phone),
			Destination: phone, ClientIP: user.IP,
		})
		if err != nil {
			auth.WriteOTPRequestError(w, r, deps.Logger, otp.ChannelWhatsApp, err)
			return
		}

		data := map[string]any{
			"verification_code": map[string]any{
				"can_resend_at":            result.CanResendAt,
				"can_resend_after_seconds": result.CanResendAfterSeconds,
			},
			"data": nil,
		}
		if deps.DebugEcho {
			data["debug_code"] = result.Code
		}
		response.Success(w, http.StatusOK, "Kode verifikasi telah dikirim.", data)
	}
}

func PhoneVerifyHandler(deps Deps, user authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		ctx := r.Context()

		v := validation.New()
		v.Field("phone_number", body["phone_number"]).Required().Regex(auth.PhoneRegex)
		v.Field("code", body["code"]).Required().String()
		if !v.Valid() {
			response.Fail(w, http.StatusUnprocessableEntity, response.MsgValidationFailed, v.Errors())
			return
		}

		phone, _ := body["phone_number"].(string)
		code, _ := body["code"].(string)

		if !ratelimit.EnforceHTTP(ctx, w, deps.Redis, "otp", phone, int64(deps.MaxPerMinute), time.Minute) {
			return
		}

		ok, err := deps.OTP.Check(ctx, otp.ChangePhoneCodeKey(user.UserID, phone), code)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if !ok {
			response.Fail(w, http.StatusBadRequest, msgOTPInvalid, nil)
			return
		}

		if err := deps.Store.UpdateUserPhone(ctx, dbgen.UpdateUserPhoneParams{ID: user.UserID, Phone: phone}); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}

		current, err := deps.Store.GetUserByID(ctx, user.UserID)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}

		response.Success(w, http.StatusOK, "Nomor handphone diperbarui.", map[string]any{
			"user": userResponse(current.ID, current.Name, current.Role, current.Phone, current.Email, current.PhoneVerifiedAt.Valid),
		})
	}
}
