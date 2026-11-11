// Package server builds the HTTP router and middleware chain for cmd/api.
package server

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/suwahadi/sewakos/backend/internal/abtest"
	"github.com/suwahadi/sewakos/backend/internal/account"
	"github.com/suwahadi/sewakos/backend/internal/area"
	"github.com/suwahadi/sewakos/backend/internal/auth"
	"github.com/suwahadi/sewakos/backend/internal/authctx"
	"github.com/suwahadi/sewakos/backend/internal/booking"
	"github.com/suwahadi/sewakos/backend/internal/config"
	"github.com/suwahadi/sewakos/backend/internal/listing"
	"github.com/suwahadi/sewakos/backend/internal/midtrans"
	"github.com/suwahadi/sewakos/backend/internal/notification"
	"github.com/suwahadi/sewakos/backend/internal/otp"
	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
	"github.com/suwahadi/sewakos/backend/internal/response"
	"github.com/suwahadi/sewakos/backend/internal/search"
	"github.com/suwahadi/sewakos/backend/internal/settings"
)

// Deps are the shared dependencies every route handler may need. Midtrans is
// optional: tests set it to a mock so no suite run reaches the real Midtrans
// sandbox; cmd/api leaves it nil and gets the real HTTP client.
type Deps struct {
	Config   config.Config
	DB       *pgxpool.Pool
	Redis    *redis.Client
	Logger   *slog.Logger
	Midtrans midtrans.Client
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
	notificationDeps := notification.Deps{Store: queries}
	areaDeps := area.Deps{Store: queries}
	searchDeps := search.Deps{
		Store: queries, AreaRadiusKM: deps.Config.AreaRadiusKM,
		StorageBaseURL: deps.Config.StoragePublicBaseURL,
	}
	listingDeps := listing.Deps{Store: queries, DB: deps.DB, StorageBaseURL: deps.Config.StoragePublicBaseURL}
	midtransClient := deps.Midtrans
	if midtransClient == nil {
		midtransClient = midtrans.NewHTTPClient(deps.Config.MidtransServerKey, deps.Config.MidtransIsProduction)
	}
	bookingDeps := booking.Deps{
		Store: queries, DB: deps.DB, AdminFee: settings.NewAdminFeeCache(queries),
		Midtrans:             midtransClient,
		MidtransServerKey:    deps.Config.MidtransServerKey,
		PaymentExpiryMinutes: deps.Config.PaymentExpiryMinutes,
		MidtransFinishURL:    deps.Config.MidtransFinishURL,
		AppPublicURL:         deps.Config.AppPublicURL,
	}

	r.Post("/api/payments/midtrans/notification", booking.WebhookHandler(bookingDeps))

	r.Route("/api", func(api chi.Router) {
		api.Use(ClientCheck(deps.Config.APIClientID))
		api.Use(RateLimitAPI(deps.Redis))

		api.Post("/auth/register", auth.RegisterHandler(register, queries))
		api.Post("/auth/otp/request", auth.OTPRequestHandler(otpDeps))
		api.Post("/auth/otp/check", auth.OTPCheckHandler(otpDeps))
		api.Post("/auth/login", auth.LoginHandler(login))
		api.Post("/auth/password/reset", account.PasswordResetHandler(accountDeps))

		api.Get("/area/{type}", area.ListHandler(areaDeps))
		api.Get("/suggestion", search.SuggestionHandler(searchDeps))
		api.Post("/listings/search", search.SearchHandler(searchDeps))
		api.Post("/listings/{property}/view", listing.ViewHandler(listingDeps))
		api.Get("/listings/{property}/gallery", listing.GalleryHandler(listingDeps))
		api.Get("/config/ab-test/{experiment}", abtest.Handler())
		api.Get("/booking/draft-price/{room}", booking.DraftPriceHandler(bookingDeps))

		api.Group(func(optional chi.Router) {
			optional.Use(OptionalAuth(tokens))
			optional.Get("/listings/{property}", withOptionalAuthedUser(listingDeps, listing.DetailHandler))
		})

		api.Group(func(authed chi.Router) {
			authed.Use(RequireAuth(tokens))

			authed.Post("/auth/logout", withAuthedUser(accountDeps, account.LogoutHandler))
			authed.Get("/me", withAuthedUser(accountDeps, account.GetMeHandler))
			authed.Patch("/me", withAuthedUser(accountDeps, account.PatchMeHandler))
			authed.Put("/me/password", withAuthedUser(accountDeps, account.PutPasswordHandler))
			authed.Post("/me/phone/request", withAuthedUser(accountDeps, account.PhoneRequestHandler))
			authed.Post("/me/phone/verify", withAuthedUser(accountDeps, account.PhoneVerifyHandler))

			authed.Get("/notifications", withAuthedUser(notificationDeps, notification.ListHandler))
			authed.Get("/notifications/counter", withAuthedUser(notificationDeps, notification.CounterHandler))
			authed.Patch("/notifications/{id}/read", withAuthedUser(notificationDeps, notification.MarkReadHandler))

			authed.Get("/booking/my", withAuthedUser(bookingDeps, booking.ListHandler))
			authed.Get("/booking/{booking}", withAuthedUser(bookingDeps, booking.DetailHandler))

			authed.Group(func(tenant chi.Router) {
				tenant.Use(RequireRole("tenant"))
				tenant.Post("/booking/submit", withAuthedUser(bookingDeps, booking.SubmitHandler))
				tenant.Post("/booking/{booking}/checkout", withAuthedUser(bookingDeps, booking.CheckoutHandler))
			})
		})
		// Remaining endpoint groups (area, search, booking, owner, admin, ...)
		// mount here in later tasks, each behind the auth/role middleware it
		// needs.
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
		handler(deps, authctx.User{UserID: user.ID, TokenID: user.TokenID, Role: user.Role})(w, r)
	}
}

// withOptionalAuthedUser is withAuthedUser for an OptionalAuth route (e.g.
// GET /listings/{property}'s is_favorite, API.md §4.1): the handler gets a
// nil *authctx.User instead of a 401 when the request carried no valid token.
func withOptionalAuthedUser[Deps any](deps Deps, handler func(Deps, *authctx.User) http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var user *authctx.User
		if u, ok := UserFromContext(r.Context()); ok {
			user = &authctx.User{UserID: u.ID, TokenID: u.TokenID, Role: u.Role}
		}
		handler(deps, user)(w, r)
	}
}
