package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
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

const msgOTPInvalid = "Kode verifikasi salah atau kedaluwarsa."

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
}

func otpCodeKey(forType, destination string) string {
	return "otp:code:" + forType + ":" + destination
}

func otpCooldownKey(destination string) string {
	return "otp:cooldown:" + destination
}

// checkOTPRateLimit enforces the shared "otp" limiter (2/minute per
// destination by default, API.md §1.5): request and check draw from the same
// counter. Redis being unreachable fails open, matching RateLimitAPI.
func checkOTPRateLimit(ctx context.Context, rdb *redis.Client, destination string, limit int) (ratelimit.Result, bool) {
	result, err := ratelimit.Allow(ctx, rdb, "otp", destination, int64(limit), time.Minute)
	if err != nil {
		return ratelimit.Result{Allowed: true}, true
	}
	return result, false
}

func writeOTPRateLimited(w http.ResponseWriter, limit int, result ratelimit.Result) {
	w.Header().Set("Retry-After", strconv.Itoa(int(result.RetryAfter.Seconds())))
	w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
	w.Header().Set("X-RateLimit-Remaining", strconv.FormatInt(result.Remaining, 10))
	response.Fail(w, http.StatusTooManyRequests, response.MsgRateLimited, nil)
}

// OTPRequestHandler implements POST /auth/otp/request (API.md §2.2, SPEC-GO.md §6.3).
func OTPRequestHandler(deps OTPDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		ctx := r.Context()

		v := validation.New()
		v.Field("for", body["for"]).Required().In(OTPForTenantVerification, OTPForLogin, OTPForResetPassword)
		v.Field("via", body["via"]).Required().In("0", "1")
		v.Field("destination", body["destination"]).Required().Regex(PhoneRegex)
		if !v.Valid() {
			response.Fail(w, http.StatusUnprocessableEntity, response.MsgValidationFailed, v.Errors())
			return
		}

		forType, _ := body["for"].(string)
		via, _ := body["via"].(string)
		destination, _ := body["destination"].(string)

		if result, skip := checkOTPRateLimit(ctx, deps.Redis, destination, deps.MaxPerMinute); !skip && !result.Allowed {
			writeOTPRateLimited(w, deps.MaxPerMinute, result)
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

		channel := otp.ChannelSMS
		if via == "1" {
			channel = otp.ChannelWhatsApp
		}

		result, err := deps.Service.Request(ctx, otpCodeKey(forType, destination), otpCooldownKey(destination), channel, destination)
		if errors.Is(err, otp.ErrCooldown) {
			const msg = "Silakan tunggu sebelum meminta kode baru."
			response.Fail(w, http.StatusUnprocessableEntity, msg, map[string][]string{"destination": {msg}})
			return
		}
		if err != nil {
			response.InternalErrorHandler(w, r)
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

		if result, skip := checkOTPRateLimit(ctx, deps.Redis, destination, deps.MaxPerMinute); !skip && !result.Allowed {
			writeOTPRateLimited(w, deps.MaxPerMinute, result)
			return
		}

		// reset_password codes are only consumable by POST /auth/password/reset
		// (SPEC-GO.md §6.4) — otp/check must neither verify nor delete them.
		if forType == OTPForResetPassword {
			response.Fail(w, http.StatusBadRequest, msgOTPInvalid, nil)
			return
		}

		ok, err := deps.Service.Check(ctx, otpCodeKey(forType, destination), code)
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
