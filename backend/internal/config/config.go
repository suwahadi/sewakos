// Package config loads the application's environment variables, per SPEC-GO.md §3.
package config

import (
	"fmt"
	"net/netip"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	AppEnv      string
	HTTPAddr    string
	DatabaseURL string
	RedisURL    string

	APIClientID        string
	CORSAllowedOrigins []string
	TrustedProxies     []netip.Prefix

	OTPLength             int
	OTPTTLSeconds         int
	OTPResendAfterSeconds int
	OTPMaxPerMinute       int
	OTPDebugEcho          bool

	MidtransServerKey    string
	MidtransSignatureKey string
	MidtransClientKey    string
	MidtransIsProduction bool
	MidtransFinishURL    string
	MidtransBaseURL      string

	PaymentExpiryMinutes       int
	OwnerResponseHours         int
	OwnerReminderBeforeMinutes int

	StorageEndpoint      string
	StorageRegion        string
	StorageBucket        string
	StorageAccessKey     string
	StorageSecretKey     string
	StorageUseSSL        bool
	StoragePublicBaseURL string

	UploadMaxBytes      int64
	ImageMaxPixels      int64
	ImageLargePx        int
	ImageThumbPx        int
	ImageLargeQuality   int
	ImageThumbQuality   int
	ImageMaxConcurrency int

	AreaRadiusKM           float64
	RenewalIssueDaysBefore int
	RenewalGraceDays       int

	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	EmailFrom    string
	AppPublicURL string
}

// Load reads configuration from the environment, applying SPEC-GO.md §3's defaults.
// It does not validate values beyond what Load itself needs (e.g. DATABASE_URL and
// REDIS_URL are required and have no default); callers that need them validate.
func Load() (Config, error) {
	cfg := Config{
		AppEnv:      getString("APP_ENV", "development"),
		HTTPAddr:    getString("HTTP_ADDR", ":8121"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		RedisURL:    os.Getenv("REDIS_URL"),

		APIClientID:        getString("API_CLIENT_ID", "KOSAPI WEB:WEB"),
		CORSAllowedOrigins: splitNonEmpty(os.Getenv("CORS_ALLOWED_ORIGINS")),

		MidtransServerKey: os.Getenv("MIDTRANS_SERVER_KEY"),
		MidtransClientKey: os.Getenv("MIDTRANS_CLIENT_KEY"),
		MidtransFinishURL: os.Getenv("MIDTRANS_FINISH_URL"),
		MidtransBaseURL:   os.Getenv("MIDTRANS_BASE_URL"),

		StorageEndpoint:      os.Getenv("STORAGE_ENDPOINT"),
		StorageRegion:        getString("STORAGE_REGION", "us-east-1"),
		StorageBucket:        os.Getenv("STORAGE_BUCKET"),
		StorageAccessKey:     os.Getenv("STORAGE_ACCESS_KEY"),
		StorageSecretKey:     os.Getenv("STORAGE_SECRET_KEY"),
		StoragePublicBaseURL: os.Getenv("STORAGE_PUBLIC_BASE_URL"),

		SMTPHost:     os.Getenv("SMTP_HOST"),
		SMTPUsername: os.Getenv("SMTP_USERNAME"),
		SMTPPassword: os.Getenv("SMTP_PASSWORD"),
		EmailFrom:    os.Getenv("EMAIL_FROM"),
		AppPublicURL: os.Getenv("APP_PUBLIC_URL"),
	}

	var err error
	if cfg.OTPLength, err = getInt("OTP_LENGTH", 4); err != nil {
		return Config{}, err
	}
	if cfg.OTPTTLSeconds, err = getInt("OTP_TTL_SECONDS", 300); err != nil {
		return Config{}, err
	}
	if cfg.OTPResendAfterSeconds, err = getInt("OTP_RESEND_AFTER_SECONDS", 60); err != nil {
		return Config{}, err
	}
	if cfg.OTPMaxPerMinute, err = getInt("OTP_MAX_PER_MINUTE", 2); err != nil {
		return Config{}, err
	}
	if cfg.OTPDebugEcho, err = getBool("OTP_DEBUG_ECHO", false); err != nil {
		return Config{}, err
	}
	if cfg.MidtransIsProduction, err = getBool("MIDTRANS_IS_PRODUCTION", false); err != nil {
		return Config{}, err
	}
	if cfg.PaymentExpiryMinutes, err = getInt("PAYMENT_EXPIRY_MINUTES", 15); err != nil {
		return Config{}, err
	}
	if cfg.OwnerResponseHours, err = getInt("OWNER_RESPONSE_HOURS", 24); err != nil {
		return Config{}, err
	}
	if cfg.OwnerReminderBeforeMinutes, err = getInt("OWNER_REMINDER_BEFORE_MINUTES", 120); err != nil {
		return Config{}, err
	}
	if cfg.StorageUseSSL, err = getBool("STORAGE_USE_SSL", true); err != nil {
		return Config{}, err
	}
	if cfg.UploadMaxBytes, err = getInt64("UPLOAD_MAX_BYTES", 10485760); err != nil {
		return Config{}, err
	}
	if cfg.ImageMaxPixels, err = getInt64("IMAGE_MAX_PIXELS", 40000000); err != nil {
		return Config{}, err
	}
	if cfg.ImageLargePx, err = getInt("IMAGE_LARGE_PX", 1600); err != nil {
		return Config{}, err
	}
	if cfg.ImageThumbPx, err = getInt("IMAGE_THUMB_PX", 400); err != nil {
		return Config{}, err
	}
	if cfg.ImageLargeQuality, err = getInt("IMAGE_LARGE_QUALITY", 80); err != nil {
		return Config{}, err
	}
	if cfg.ImageThumbQuality, err = getInt("IMAGE_THUMB_QUALITY", 75); err != nil {
		return Config{}, err
	}
	defaultConcurrency := 4
	if cfg.ImageMaxConcurrency, err = getInt("IMAGE_MAX_CONCURRENCY", defaultConcurrency); err != nil {
		return Config{}, err
	}
	if cfg.AreaRadiusKM, err = getFloat("AREA_RADIUS_KM", 3); err != nil {
		return Config{}, err
	}
	if cfg.RenewalIssueDaysBefore, err = getInt("RENEWAL_ISSUE_DAYS_BEFORE", 3); err != nil {
		return Config{}, err
	}
	if cfg.RenewalGraceDays, err = getInt("RENEWAL_GRACE_DAYS", 3); err != nil {
		return Config{}, err
	}
	if cfg.SMTPPort, err = getInt("SMTP_PORT", 0); err != nil {
		return Config{}, err
	}
	if cfg.TrustedProxies, err = getPrefixes("TRUSTED_PROXIES"); err != nil {
		return Config{}, err
	}
	cfg.MidtransSignatureKey = getString("MIDTRANS_SIGNATURE_KEY", cfg.MidtransServerKey)

	// OTP_DEBUG_ECHO must never leak real codes in production (API.md §2.2, SPEC-GO.md §3).
	if cfg.AppEnv == "production" && cfg.OTPDebugEcho {
		return Config{}, fmt.Errorf("OTP_DEBUG_ECHO must be false when APP_ENV=production")
	}
	// A base URL override points payments at a fake Midtrans that settles anything it is asked about.
	if cfg.AppEnv == "production" && cfg.MidtransBaseURL != "" {
		return Config{}, fmt.Errorf("MIDTRANS_BASE_URL must be empty when APP_ENV=production")
	}

	return cfg, nil
}

func getString(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getInt(key string, def int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("parse %s=%q: %w", key, v, err)
	}
	return n, nil
}

func getInt64(key string, def int64) (int64, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse %s=%q: %w", key, v, err)
	}
	return n, nil
}

func getFloat(key string, def float64) (float64, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	n, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, fmt.Errorf("parse %s=%q: %w", key, v, err)
	}
	return n, nil
}

func getBool(key string, def bool) (bool, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false, fmt.Errorf("parse %s=%q: %w", key, v, err)
	}
	return b, nil
}

func getPrefixes(key string) ([]netip.Prefix, error) {
	var prefixes []netip.Prefix
	for _, entry := range splitNonEmpty(os.Getenv(key)) {
		if addr, err := netip.ParseAddr(entry); err == nil {
			prefixes = append(prefixes, netip.PrefixFrom(addr, addr.BitLen()))
			continue
		}
		prefix, err := netip.ParsePrefix(entry)
		if err != nil {
			return nil, fmt.Errorf("%s: invalid IP or CIDR %q", key, entry)
		}
		prefixes = append(prefixes, prefix.Masked())
	}
	return prefixes, nil
}

func splitNonEmpty(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
