package config

import "testing"

func TestLoadRefusesMidtransBaseURLInProduction(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("OTP_DEBUG_ECHO", "false")
	t.Setenv("MIDTRANS_BASE_URL", "http://127.0.0.1:8125")

	if _, err := Load(); err == nil {
		t.Fatal("Load accepted MIDTRANS_BASE_URL with APP_ENV=production")
	}
}

func TestLoadKeepsMidtransBaseURLOutsideProduction(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("MIDTRANS_BASE_URL", "http://127.0.0.1:8125")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.MidtransBaseURL != "http://127.0.0.1:8125" {
		t.Errorf("MidtransBaseURL = %q", cfg.MidtransBaseURL)
	}
}

func TestLoadFallsBackToServerKeyForSignatures(t *testing.T) {
	t.Setenv("MIDTRANS_SERVER_KEY", "server-key")
	t.Setenv("MIDTRANS_SIGNATURE_KEY", "")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.MidtransSignatureKey != "server-key" {
		t.Errorf("MidtransSignatureKey = %q, want the server key when unset", cfg.MidtransSignatureKey)
	}

	t.Setenv("MIDTRANS_SIGNATURE_KEY", "signature-secret")
	if cfg, err = Load(); err != nil || cfg.MidtransSignatureKey != "signature-secret" || cfg.MidtransServerKey != "server-key" {
		t.Errorf("Load with signature key = %+v, %v", cfg.MidtransSignatureKey, err)
	}
}

func TestLoadDefaultsToLogOTPProvider(t *testing.T) {
	t.Setenv("OTP_PROVIDER", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.OTPProvider != "log" {
		t.Errorf("OTPProvider = %q, want log", cfg.OTPProvider)
	}
}

func TestLoadRequiresFazpassKeys(t *testing.T) {
	t.Setenv("OTP_PROVIDER", "fazpass")
	t.Setenv("FAZPASS_MERCHANT_KEY", "merchant")
	t.Setenv("FAZPASS_GATEWAY_KEY", "")

	if _, err := Load(); err == nil {
		t.Fatal("Load accepted OTP_PROVIDER=fazpass without FAZPASS_GATEWAY_KEY")
	}

	t.Setenv("FAZPASS_GATEWAY_KEY", "gateway")
	if _, err := Load(); err != nil {
		t.Fatalf("Load with both Fazpass keys: %v", err)
	}
}

func TestLoadRejectsUnknownOTPProvider(t *testing.T) {
	t.Setenv("OTP_PROVIDER", "sms-gateway")

	if _, err := Load(); err == nil {
		t.Fatal("Load accepted an unknown OTP_PROVIDER")
	}
}
