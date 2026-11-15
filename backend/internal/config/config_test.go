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

func TestLoadDefaultsToSandboxWhatsAppAndNoSMS(t *testing.T) {
	t.Setenv("OTP_PROVIDER", "")
	t.Setenv("OTP_WHATSAPP_PROVIDER", "")
	t.Setenv("OTP_SMS_PROVIDER", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.OTPWhatsAppProvider != "log" || cfg.OTPSMSProvider != "off" {
		t.Errorf("providers = %q/%q, want log/off", cfg.OTPWhatsAppProvider, cfg.OTPSMSProvider)
	}
	if cfg.OTPSMSMaxPerDay != 50 {
		t.Errorf("OTPSMSMaxPerDay = %d, want 50", cfg.OTPSMSMaxPerDay)
	}
}

func TestLoadReadsTheLegacyOTPProviderForWhatsApp(t *testing.T) {
	t.Setenv("OTP_WHATSAPP_PROVIDER", "")
	t.Setenv("OTP_PROVIDER", "fazpass")
	t.Setenv("FAZPASS_MERCHANT_KEY", "merchant")
	t.Setenv("FAZPASS_GATEWAY_KEY", "gateway")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.OTPWhatsAppProvider != "fazpass" {
		t.Errorf("OTPWhatsAppProvider = %q, want fazpass from OTP_PROVIDER", cfg.OTPWhatsAppProvider)
	}
}

func TestLoadRequiresSMSGateCredentials(t *testing.T) {
	t.Setenv("OTP_SMS_PROVIDER", "smsgate")
	t.Setenv("SMSGATE_USERNAME", "user")
	t.Setenv("SMSGATE_PASSWORD", "")

	if _, err := Load(); err == nil {
		t.Fatal("Load accepted OTP_SMS_PROVIDER=smsgate without SMSGATE_PASSWORD")
	}

	t.Setenv("SMSGATE_PASSWORD", "password")
	t.Setenv("SMSGATE_SIM_NUMBER", "4")
	if _, err := Load(); err == nil {
		t.Fatal("Load accepted SMSGATE_SIM_NUMBER=4")
	}

	t.Setenv("SMSGATE_SIM_NUMBER", "2")
	if _, err := Load(); err != nil {
		t.Fatalf("Load with complete SMS gateway settings: %v", err)
	}
}

func TestLoadRequiresFazpassKeys(t *testing.T) {
	t.Setenv("OTP_WHATSAPP_PROVIDER", "fazpass")
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

func TestLoadRejectsUnknownOTPProviders(t *testing.T) {
	t.Setenv("OTP_WHATSAPP_PROVIDER", "sms-gateway")
	if _, err := Load(); err == nil {
		t.Fatal("Load accepted an unknown OTP_WHATSAPP_PROVIDER")
	}

	t.Setenv("OTP_WHATSAPP_PROVIDER", "log")
	t.Setenv("OTP_SMS_PROVIDER", "carrier-pigeon")
	if _, err := Load(); err == nil {
		t.Fatal("Load accepted an unknown OTP_SMS_PROVIDER")
	}
}
