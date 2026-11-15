package auth

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/suwahadi/sewakos/backend/internal/otp"
	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
	"github.com/suwahadi/sewakos/backend/internal/platform/ratelimit"
	"github.com/suwahadi/sewakos/backend/internal/response"
	"github.com/suwahadi/sewakos/backend/internal/validation"
)

// OTP "for" values (API.md §2.2-2.3).
const (
	OTPForTenantVerification = "tenant_verification"
	OTPForLogin              = "login"
	OTPForResetPassword      = "reset_password"
)

const (
	msgOTPInvalid     = "Kode verifikasi salah atau kedaluwarsa."
	msgOTPCooldown    = "Silakan tunggu sebelum meminta kode baru."
	msgOTPSendLimit   = "Batas pengiriman kode untuk nomor ini sudah tercapai. Silakan coba lagi nanti."
	msgOTPSendFailed  = "Kode verifikasi gagal dikirim. Silakan coba lagi dalam 1 menit."
	msgOTPChannelFull = "Pengiriman kode sedang tidak tersedia. Silakan coba lagi nanti."
)

var msgOTPSwitchChannel = map[otp.Channel]string{
	otp.ChannelWhatsApp: "Pengiriman kode lewat SMS sedang tidak tersedia. Silakan kirim lewat WhatsApp.",
	otp.ChannelSMS:      "Pengiriman kode lewat WhatsApp sedang tidak tersedia. Silakan kirim lewat SMS.",
}

func WriteOTPRequestError(w http.ResponseWriter, r *http.Request, logger *slog.Logger, channel, fallback otp.Channel, err error) {
	switchMessage, canSwitch := msgOTPSwitchChannel[fallback]
	if errors.Is(err, otp.ErrSendFailed) && logger != nil {
		logger.Error("otp send failed", "channel", channel, "error", err)
	}
	switch {
	case errors.Is(err, otp.ErrCooldown):
		response.Fail(w, http.StatusUnprocessableEntity, msgOTPCooldown, map[string][]string{"destination": {msgOTPCooldown}})
	case errors.Is(err, otp.ErrSendLimit):
		response.Fail(w, http.StatusTooManyRequests, msgOTPSendLimit, nil)
	case errors.Is(err, otp.ErrChannelLimit) && canSwitch:
		response.Fail(w, http.StatusTooManyRequests, switchMessage, map[string][]string{"via": {switchMessage}})
	case errors.Is(err, otp.ErrChannelLimit):
		response.Fail(w, http.StatusTooManyRequests, msgOTPChannelFull, nil)
	case errors.Is(err, otp.ErrSendFailed) && errors.Is(err, otp.ErrCooldownReleased) && canSwitch:
		response.Fail(w, http.StatusServiceUnavailable, switchMessage, map[string][]string{"via": {switchMessage}})
	case errors.Is(err, otp.ErrSendFailed):
		response.Fail(w, http.StatusServiceUnavailable, msgOTPSendFailed, nil)
	default:
		if logger != nil {
			logger.Error("otp request failed", "error", err)
		}
		response.InternalErrorHandler(w, r)
	}
}

func remoteIP(r *http.Request) string {
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

// OTPUserStore is the subset of dbgen.Querier the OTP handlers need.
type OTPUserStore interface {
	GetUserByPhone(ctx context.Context, phone string) (dbgen.GetUserByPhoneRow, error)
	VerifyUserPhone(ctx context.Context, id int64) error
}

// OTPDeps bundles the OTP request/check handlers' shared dependencies.
type OTPDeps struct {
	Service      *otp.Service
	Store        OTPUserStore
	Tokens       *Service
	Redis        *redis.Client
	MaxPerMinute int
	DebugEcho    bool
	Logger       *slog.Logger
}

// OTPRequestHandler implements POST /auth/otp/request (API.md §2.2, SPEC-GO.md §6.3).
func OTPRequestHandler(deps OTPDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		ctx := r.Context()

		v := validation.New()
		v.Field("for", body["for"]).Required().In(OTPForTenantVerification, OTPForLogin, OTPForResetPassword)
		v.Field("via", body["via"]).Optional().In("0", "1")
		v.Field("destination", body["destination"]).Required().Regex(PhoneRegex)
		if !v.Valid() {
			response.Fail(w, http.StatusUnprocessableEntity, response.MsgValidationFailed, v.Errors())
			return
		}

		forType, _ := body["for"].(string)
		destination, _ := body["destination"].(string)
		channel, fallback := otp.ChannelWhatsApp, otp.ChannelSMS
		if via, _ := body["via"].(string); via == "0" && deps.Service.HasChannel(otp.ChannelSMS) {
			channel, fallback = otp.ChannelSMS, otp.ChannelWhatsApp
		}
		if !deps.Service.HasChannel(fallback) {
			fallback = ""
		}

		if !ratelimit.EnforceHTTP(ctx, w, deps.Redis, "otp", destination, int64(deps.MaxPerMinute), time.Minute) {
			return
		}

		user, err := deps.Store.GetUserByPhone(ctx, destination)
		userExists := err == nil

		switch forType {
		case OTPForTenantVerification:
			if !userExists || user.PhoneVerifiedAt.Valid {
				response.Fail(w, http.StatusNotFound, response.MsgNotFound, nil)
				return
			}
		case OTPForResetPassword:
			if !userExists {
				response.Fail(w, http.StatusNotFound, response.MsgNotFound, nil)
				return
			}
			if user.SuspendedAt.Valid {
				response.Fail(w, http.StatusForbidden, response.MsgAccountSuspended, nil)
				return
			}
		case OTPForLogin:
			if !userExists {
				response.Fail(w, http.StatusNotFound, response.MsgNotFound, nil)
				return
			}
			if !user.PhoneVerifiedAt.Valid {
				const msg = "Nomor handphone belum terverifikasi."
				response.Fail(w, http.StatusUnprocessableEntity, msg, map[string][]string{"destination": {msg}})
				return
			}
		}

		result, err := deps.Service.Request(ctx, otp.RequestParams{
			Channel: channel, CodeKey: otp.CodeKey(forType, destination), CooldownKey: otp.CooldownKey(destination),
			Destination: destination, ClientIP: remoteIP(r),
		})
		if err != nil {
			WriteOTPRequestError(w, r, deps.Logger, channel, fallback, err)
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

// OTPCheckHandler implements POST /auth/otp/check (API.md §2.3, SPEC-GO.md §6.4).
func OTPCheckHandler(deps OTPDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		ctx := r.Context()

		v := validation.New()
		v.Field("for", body["for"]).Required().In(OTPForTenantVerification, OTPForLogin, OTPForResetPassword)
		v.Field("destination", body["destination"]).Required().Regex(PhoneRegex)
		v.Field("code", body["code"]).Required().String()
		if !v.Valid() {
			response.Fail(w, http.StatusUnprocessableEntity, response.MsgValidationFailed, v.Errors())
			return
		}

		forType, _ := body["for"].(string)
		destination, _ := body["destination"].(string)
		code, _ := body["code"].(string)

		if !ratelimit.EnforceHTTP(ctx, w, deps.Redis, "otp", destination, int64(deps.MaxPerMinute), time.Minute) {
			return
		}

		// reset_password codes are only consumable by POST /auth/password/reset
		// (SPEC-GO.md §6.4) — otp/check must neither verify nor delete them.
		if forType == OTPForResetPassword {
			response.Fail(w, http.StatusBadRequest, msgOTPInvalid, nil)
			return
		}

		ok, err := deps.Service.Check(ctx, otp.CodeKey(forType, destination), code)
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

		if forType == OTPForTenantVerification {
			if err := deps.Store.VerifyUserPhone(ctx, user.ID); err != nil {
				response.InternalErrorHandler(w, r)
				return
			}
		}

		token, err := deps.Tokens.Issue(ctx, user.ID)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}

		response.Success(w, http.StatusOK, "Nomor handphone terverifikasi.", map[string]any{
			"verified": true,
			"token":    token,
			"user": map[string]any{
				"id":   user.ID,
				"name": user.Name,
				"role": user.Role,
			},
		})
	}
}

func OTPChannelsHandler(deps OTPDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		response.Success(w, http.StatusOK, "", map[string]any{"channels": deps.Service.Channels()})
	}
}
