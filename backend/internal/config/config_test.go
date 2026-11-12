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
