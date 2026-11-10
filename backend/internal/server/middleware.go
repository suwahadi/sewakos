package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/suwahadi/sewakos/backend/internal/auth"
	"github.com/suwahadi/sewakos/backend/internal/platform/ratelimit"
	"github.com/suwahadi/sewakos/backend/internal/response"
)

type ctxKey int

const (
	ctxKeyRequestID ctxKey = iota
	ctxKeyUser
)

// RequestID assigns a random request id to every request, exposed both as a
// response header and in the request context for logging.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := newRequestID()
		w.Header().Set("X-Request-Id", id)
		ctx := context.WithValue(r.Context(), ctxKeyRequestID, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func newRequestID() string {
	buf := make([]byte, 16)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}

// RequestIDFromContext returns the request id set by RequestID, or "" if absent.
func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(ctxKeyRequestID).(string)
	return id
}

// Recoverer turns a panic in a later handler into a 500 envelope. The panic
// detail is logged, never returned to the client (SPEC-GO.md §5.3).
func Recoverer(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logger.Error("panic recovered",
						"request_id", RequestIDFromContext(r.Context()),
						"panic", rec,
						"path", r.URL.Path,
					)
					response.InternalErrorHandler(w, r)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// CORS answers preflight requests and sets CORS headers, only when origins is
// non-empty (SPEC-GO.md §5.1: an empty CORS_ALLOWED_ORIGINS means no CORS at all).
func CORS(origins []string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(origins))
	for _, o := range origins {
		allowed[o] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		if len(allowed) == 0 {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if _, ok := allowed[origin]; ok {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Headers", "x-api-client, Content-Type, Authorization")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				w.Header().Set("Vary", "Origin")
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ClientCheck enforces the x-api-client header (SPEC-GO.md §5.1). The Midtrans
// webhook is mounted outside the /api group that carries this middleware, so it
// never needs an explicit exception here.
func ClientCheck(expected string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("x-api-client") != expected {
				response.Fail(w, http.StatusUnauthorized, response.MsgUnknownClient, nil)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RateLimitAPI enforces the "api" limiter: 60 requests/minute per IP
// (SPEC-GO.md §5.1, API.md §1.5). The "otp" limiter (2/minute per destination
// phone number) shares the same ratelimit.Allow primitive but isn't HTTP
// middleware — the OTP service calls it directly, keyed by destination
// rather than by request (B2.1, since it has no endpoint to attach to yet).
func RateLimitAPI(rdb *redis.Client) func(http.Handler) http.Handler {
	const limit = 60
	const window = time.Minute
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			result, err := ratelimit.Allow(r.Context(), rdb, "api", clientIP(r), limit, window)
			if err != nil {
				// Redis being unavailable must not take the whole API down; fail open.
				next.ServeHTTP(w, r)
				return
			}

			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
			w.Header().Set("X-RateLimit-Remaining", strconv.FormatInt(result.Remaining, 10))

			if !result.Allowed {
				w.Header().Set("Retry-After", strconv.Itoa(int(result.RetryAfter.Seconds())))
				response.Fail(w, http.StatusTooManyRequests, response.MsgRateLimited, nil)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func clientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		if parts := strings.SplitN(fwd, ",", 2); len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	host := r.RemoteAddr
	if i := strings.LastIndex(host, ":"); i != -1 {
		host = host[:i]
	}
	return host
}

// AuthUser is the authenticated user carried in context by RequireAuth/OptionalAuth.
type AuthUser struct {
	ID   int64
	Role string
}

// UserFromContext returns the authenticated user, or (AuthUser{}, false) if the
// request carried no (valid) token.
func UserFromContext(ctx context.Context) (AuthUser, bool) {
	u, ok := ctx.Value(ctxKeyUser).(AuthUser)
	return u, ok
}

// Verifier is the subset of auth.Service that middleware needs, so this package
// doesn't depend on auth's concrete storage.
type Verifier interface {
	Verify(ctx context.Context, token string) (auth.VerifiedUser, error)
}

// RequireAuth enforces Bearer auth on [auth], /owner/*, and /admin/* routes
// (SPEC-GO.md §5.1). A suspended account's tokens are already deleted at
// suspend time, so an invalid/missing token and a suspended account are the
// only two outcomes this middleware itself needs to distinguish.
func RequireAuth(verifier Verifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, err := authenticate(r, verifier)
			if err != nil {
				response.Fail(w, http.StatusUnauthorized, response.MsgInvalidSession, nil)
				return
			}
			ctx := context.WithValue(r.Context(), ctxKeyUser, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalAuth attaches the user to context when the request carries a valid
// token, but never rejects the request otherwise (used by GET /listings/{id}
// for is_favorite, API.md §4.1).
func OptionalAuth(verifier Verifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if user, err := authenticate(r, verifier); err == nil {
				r = r.WithContext(context.WithValue(r.Context(), ctxKeyUser, user))
			}
			next.ServeHTTP(w, r)
		})
	}
}

func authenticate(r *http.Request, verifier Verifier) (AuthUser, error) {
	authz := r.Header.Get("Authorization")
	token, ok := strings.CutPrefix(authz, "Bearer ")
	if !ok || token == "" {
		return AuthUser{}, auth.ErrInvalidToken
	}
	verified, err := verifier.Verify(r.Context(), token)
	if err != nil {
		return AuthUser{}, err
	}
	return AuthUser{ID: verified.ID, Role: verified.Role}, nil
}

// RequireRole rejects any authenticated user whose role isn't in roles
// (SPEC-GO.md §5.1 step 7); it must run after RequireAuth.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := UserFromContext(r.Context())
			if !ok {
				response.Fail(w, http.StatusUnauthorized, response.MsgInvalidSession, nil)
				return
			}
			if _, ok := allowed[user.Role]; !ok {
				response.Fail(w, http.StatusForbidden, response.MsgAccessDenied, nil)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
