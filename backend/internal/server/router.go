// Package server builds the HTTP router and middleware chain for cmd/api.
package server

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/suwahadi/sewakos/backend/internal/account"
	"github.com/suwahadi/sewakos/backend/internal/auth"
	"github.com/suwahadi/sewakos/backend/internal/authctx"
	"github.com/suwahadi/sewakos/backend/internal/config"
	"github.com/suwahadi/sewakos/backend/internal/otp"
	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
	"github.com/suwahadi/sewakos/backend/internal/response"
)

// Deps are the shared dependencies every route handler may need.
type Deps struct {
	Config config.Config
	DB     *pgxpool.Pool
	Redis  *redis.Client
	Logger *slog.Logger
}

// NewRouter assembles the full middleware chain (SPEC-GO.md §5.1) and mounts
// routes under /api. Route groups are added package by package as each one is
// implemented (auth, area, search, ... in later tasks); for now it only wires
// the middleware stack and the foundational envelope-based error handlers.
func NewRouter(deps Deps) chi.Router {
	r := chi.NewRouter()

	r.Use(RequestID)
	r.Use(Recoverer(deps.Logger))
	r.Use(CORS(deps.Config.CORSAllowedOrigins))

	r.NotFound(response.NotFoundHandler)
	r.MethodNotAllowed(response.MethodNotAllowedHandler)

	queries := dbgen.New(deps.DB)
	tokens := auth.NewService(queries)
	login := auth.NewLoginService(queries, tokens)
	register := auth.NewRegisterService(queries)
	otpService := otp.NewService(deps.Redis, otp.LogSender{Logger: deps.Logger}, otp.Config{
		CodeLength:  deps.Config.OTPLength,
		TTL:         time.Duration(deps.Config.OTPTTLSeconds) * time.Second,
		ResendAfter: time.Duration(deps.Config.OTPResendAfterSeconds) * time.Second,
	})
	otpDeps := auth.OTPDeps{
		Service:      otpService,
		Store:        queries,
		Tokens:       tokens,
		Redis:        deps.Redis,
		MaxPerMinute: deps.Config.OTPMaxPerMinute,
		DebugEcho:    deps.Config.OTPDebugEcho,
	}
	accountDeps := account.Deps{
		Store:        queries,
		OTP:          otpService,
		Redis:        deps.Redis,
		MaxPerMinute: deps.Config.OTPMaxPerMinute,
		DebugEcho:    deps.Config.OTPDebugEcho,
	}

	r.Route("/api", func(api chi.Router) {
		api.Use(ClientCheck(deps.Config.APIClientID))
		api.Use(RateLimitAPI(deps.Redis))

		api.Post("/auth/register", auth.RegisterHandler(register, queries))
		api.Post("/auth/otp/request", auth.OTPRequestHandler(otpDeps))
		api.Post("/auth/otp/check", auth.OTPCheckHandler(otpDeps))
		api.Post("/auth/login", auth.LoginHandler(login))
		api.Post("/auth/password/reset", account.PasswordResetHandler(accountDeps))

		api.Group(func(authed chi.Router) {
			authed.Use(RequireAuth(tokens))

			authed.Post("/auth/logout", withAuthedUser(accountDeps, account.LogoutHandler))
			authed.Get("/me", withAuthedUser(accountDeps, account.GetMeHandler))
			authed.Patch("/me", withAuthedUser(accountDeps, account.PatchMeHandler))
			authed.Put("/me/password", withAuthedUser(accountDeps, account.PutPasswordHandler))
			authed.Post("/me/phone/request", withAuthedUser(accountDeps, account.PhoneRequestHandler))
			authed.Post("/me/phone/verify", withAuthedUser(accountDeps, account.PhoneVerifyHandler))
		})
		// Remaining endpoint groups (notifications, area, search, booking,
		// owner, admin, ...) mount here in later tasks (B2.3+), each behind
		// the auth/role middleware it needs.
	})

	return r
}

// withAuthedUser adapts a feature package's handler (account, notification,
// ...) to a chi route. Those packages don't depend on this one (server mounts
// their routes, so the reverse import would cycle), so this reads the
// already-authenticated user RequireAuth put in context and hands it over as
// authctx.User instead.
func withAuthedUser[Deps any](deps Deps, handler func(Deps, authctx.User) http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, _ := UserFromContext(r.Context())
		handler(deps, authctx.User{UserID: user.ID, TokenID: user.TokenID})(w, r)
	}
}
