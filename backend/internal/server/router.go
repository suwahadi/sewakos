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
	"github.com/suwahadi/sewakos/backend/internal/admin"
	"github.com/suwahadi/sewakos/backend/internal/area"
	"github.com/suwahadi/sewakos/backend/internal/auth"
	"github.com/suwahadi/sewakos/backend/internal/authctx"
	"github.com/suwahadi/sewakos/backend/internal/booking"
	"github.com/suwahadi/sewakos/backend/internal/config"
	"github.com/suwahadi/sewakos/backend/internal/listing"
	"github.com/suwahadi/sewakos/backend/internal/midtrans"
	"github.com/suwahadi/sewakos/backend/internal/notification"
	"github.com/suwahadi/sewakos/backend/internal/otp"
	"github.com/suwahadi/sewakos/backend/internal/owner"
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
	Storage  owner.Storage
	Images   owner.ImageProcessor
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
		midtransClient = midtrans.NewHTTPClient(deps.Config.MidtransServerKey, deps.Config.MidtransIsProduction, deps.Config.MidtransBaseURL)
	}
	adminFeeCache := settings.NewAdminFeeCache(queries)
	bookingDeps := booking.Deps{
		Store: queries, DB: deps.DB, AdminFee: adminFeeCache,
		Midtrans:             midtransClient,
		MidtransServerKey:    deps.Config.MidtransServerKey,
		PaymentExpiryMinutes: deps.Config.PaymentExpiryMinutes,
		MidtransFinishURL:    deps.Config.MidtransFinishURL,
		AppPublicURL:         deps.Config.AppPublicURL,
	}

	ownerDeps := owner.Deps{
		Store: queries, DB: deps.DB, Storage: deps.Storage, Images: deps.Images,
		StorageBaseURL: deps.Config.StoragePublicBaseURL, UploadMaxBytes: deps.Config.UploadMaxBytes,
		AppPublicURL: deps.Config.AppPublicURL,
	}

	adminDeps := admin.Deps{
		Store: queries, DB: deps.DB, AdminFee: adminFeeCache,
		StorageBaseURL: deps.Config.StoragePublicBaseURL, AppPublicURL: deps.Config.AppPublicURL,
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
			authed.Get("/invoices", withAuthedUser(bookingDeps, booking.InvoiceListHandler))
			authed.Get("/invoices/{invoice}", withAuthedUser(bookingDeps, booking.InvoiceDetailHandler))

			authed.Group(func(tenant chi.Router) {
				tenant.Use(RequireRole("tenant"))
				tenant.Post("/booking/submit", withAuthedUser(bookingDeps, booking.SubmitHandler))
				tenant.Post("/booking/{booking}/checkout", withAuthedUser(bookingDeps, booking.CheckoutHandler))
				tenant.Post("/booking/{booking}/cancel", withAuthedUser(bookingDeps, booking.CancelHandler))
				tenant.Put("/booking/{booking}/refund-account", withAuthedUser(bookingDeps, booking.RefundAccountHandler))
				tenant.Post("/invoices/{invoice}/checkout", withAuthedUser(bookingDeps, booking.InvoiceCheckoutHandler))
			})

			authed.Group(func(adminRoutes chi.Router) {
				adminRoutes.Use(RequireRole("admin"))
				adminRoutes.Get("/admin/areas", withAuthedUser(adminDeps, admin.ListAreasHandler))
				adminRoutes.Post("/admin/areas", withAuthedUser(adminDeps, admin.CreateAreaHandler))
				adminRoutes.Patch("/admin/areas/{area}", withAuthedUser(adminDeps, admin.UpdateAreaHandler))
				adminRoutes.Delete("/admin/areas/{area}", withAuthedUser(adminDeps, admin.DeleteAreaHandler))
				adminRoutes.Get("/admin/users", withAuthedUser(adminDeps, admin.ListUsersHandler))
				adminRoutes.Get("/admin/users/{user}", withAuthedUser(adminDeps, admin.GetUserHandler))
				adminRoutes.Post("/admin/users/{user}/suspend", withAuthedUser(adminDeps, admin.SuspendUserHandler))
				adminRoutes.Post("/admin/users/{user}/unsuspend", withAuthedUser(adminDeps, admin.UnsuspendUserHandler))
				adminRoutes.Get("/admin/properties", withAuthedUser(adminDeps, admin.ListPropertiesHandler))
				adminRoutes.Get("/admin/properties/{property}", withAuthedUser(adminDeps, admin.GetPropertyHandler))
				adminRoutes.Post("/admin/properties/{property}/approve", withAuthedUser(adminDeps, admin.ApprovePropertyHandler))
				adminRoutes.Post("/admin/properties/{property}/reject", withAuthedUser(adminDeps, admin.RejectPropertyHandler))
				adminRoutes.Post("/admin/properties/{property}/suspend", withAuthedUser(adminDeps, admin.SuspendPropertyHandler))
				adminRoutes.Get("/admin/bookings", withAuthedUser(adminDeps, admin.ListBookingsHandler))
				adminRoutes.Get("/admin/bookings/{booking}", withAuthedUser(adminDeps, admin.GetBookingHandler))
				adminRoutes.Get("/admin/invoices", withAuthedUser(adminDeps, admin.ListInvoicesHandler))
				adminRoutes.Get("/admin/refunds", withAuthedUser(adminDeps, admin.ListRefundsHandler))
				adminRoutes.Post("/admin/refunds/{refund}/complete", withAuthedUser(adminDeps, admin.CompleteRefundHandler))
				adminRoutes.Get("/admin/payouts", withAuthedUser(adminDeps, admin.ListPayoutsHandler))
				adminRoutes.Post("/admin/payouts/{payout}/complete", withAuthedUser(adminDeps, admin.CompletePayoutHandler))
				adminRoutes.Get("/admin/settings", withAuthedUser(adminDeps, admin.GetSettingsHandler))
				adminRoutes.Patch("/admin/settings", withAuthedUser(adminDeps, admin.PatchSettingsHandler))
				adminRoutes.Post("/admin/notifications/broadcast", withAuthedUser(adminDeps, admin.BroadcastHandler))
			})

			authed.Group(func(ownerRoutes chi.Router) {
				ownerRoutes.Use(RequireRole("owner"))
				ownerRoutes.Get("/owner/properties", withAuthedUser(ownerDeps, owner.ListPropertiesHandler))
				ownerRoutes.Post("/owner/properties", withAuthedUser(ownerDeps, owner.CreatePropertyHandler))
				ownerRoutes.Get("/owner/properties/{property}", withAuthedUser(ownerDeps, owner.GetPropertyHandler))
				ownerRoutes.Patch("/owner/properties/{property}", withAuthedUser(ownerDeps, owner.UpdatePropertyHandler))
				ownerRoutes.Post("/owner/properties/{property}/submit", withAuthedUser(ownerDeps, owner.SubmitPropertyHandler))
				ownerRoutes.Get("/owner/properties/{property}/rooms", withAuthedUser(ownerDeps, owner.ListRoomsHandler))
				ownerRoutes.Post("/owner/properties/{property}/rooms", withAuthedUser(ownerDeps, owner.CreateRoomHandler))
				ownerRoutes.Patch("/owner/rooms/{room}", withAuthedUser(ownerDeps, owner.UpdateRoomHandler))
				ownerRoutes.Delete("/owner/rooms/{room}", withAuthedUser(ownerDeps, owner.DeleteRoomHandler))
				ownerRoutes.Get("/owner/bookings", withAuthedUser(ownerDeps, owner.ListBookingsHandler))
				ownerRoutes.Get("/owner/bookings/{booking}", withAuthedUser(ownerDeps, owner.GetBookingHandler))
				ownerRoutes.Post("/owner/bookings/{booking}/approve", withAuthedUser(ownerDeps, owner.ApproveBookingHandler))
				ownerRoutes.Post("/owner/bookings/{booking}/reject", withAuthedUser(ownerDeps, owner.RejectBookingHandler))
				ownerRoutes.Get("/owner/invoices", withAuthedUser(ownerDeps, owner.ListInvoicesHandler))
				ownerRoutes.Get("/owner/bank-account", withAuthedUser(ownerDeps, owner.GetBankAccountHandler))
				ownerRoutes.Put("/owner/bank-account", withAuthedUser(ownerDeps, owner.PutBankAccountHandler))
				ownerRoutes.Get("/owner/payouts", withAuthedUser(ownerDeps, owner.ListPayoutsHandler))
				ownerRoutes.Post("/owner/uploads/presign", withAuthedUser(ownerDeps, owner.PresignHandler))
				ownerRoutes.Post("/owner/properties/{property}/photos", withAuthedUser(ownerDeps, owner.AddPhotoHandler))
				ownerRoutes.Patch("/owner/photos/{photo}", withAuthedUser(ownerDeps, owner.UpdatePhotoHandler))
				ownerRoutes.Delete("/owner/photos/{photo}", withAuthedUser(ownerDeps, owner.DeletePhotoHandler))
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
		handler(deps, authctx.User{UserID: user.ID, TokenID: user.TokenID, Role: user.Role, IP: clientIP(r)})(w, r)
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
