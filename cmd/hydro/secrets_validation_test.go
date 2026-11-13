package main

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/hydrohq/hydrogo/internal/config"
)

func TestValidateSecretsInProduction(t *testing.T) {
	validCfg := config.Config{
		Payments: config.Payments{
			PayTR: config.PayTR{
				MerchantID:   "m-1",
				MerchantKey:  "valid-merchant-key",
				MerchantSalt: "valid-merchant-salt",
				BaseURL:      "https://paytr.example.com",
			},
		},
		Webhooks: []config.Webhook{
			{URL: "https://hook.example.com", Secret: "valid-hook-secret"},
		},
	}

	validLookup := func(key string) (string, bool) {
		switch key {
		case "HYDRO_JWT_SECRET":
			return "super-secure-random-jwt-secret-at-least-32-chars", true
		case "RECOVER_SECRET":
			return "super-secure-recover-secret-value", true
		case "SHOPIER_API_KEY":
			return "shopier-key-123", true
		case "SHOPIER_API_SECRET":
			return "shopier-secret-456", true
		}
		return "", false
	}

	// 1. All valid secrets in production: should pass
	if err := validateSecrets(validCfg, validLookup, true, nil); err != nil {
		t.Fatalf("expected valid secrets to pass in production, got: %v", err)
	}

	// 2. Missing or CHANGE_ME HYDRO_JWT_SECRET in production: must fail
	badJWTLookup := func(key string) (string, bool) {
		if key == "HYDRO_JWT_SECRET" {
			return "CHANGE_ME", true
		}
		return validLookup(key)
	}
	if err := validateSecrets(validCfg, badJWTLookup, true, nil); err == nil || !strings.Contains(err.Error(), "HYDRO_JWT_SECRET") {
		t.Fatalf("expected error for CHANGE_ME HYDRO_JWT_SECRET, got: %v", err)
	}

	// 3. Missing RECOVER_SECRET in production: must fail
	badRecoverLookup := func(key string) (string, bool) {
		if key == "RECOVER_SECRET" {
			return "", true
		}
		return validLookup(key)
	}
	if err := validateSecrets(validCfg, badRecoverLookup, true, nil); err == nil || !strings.Contains(err.Error(), "RECOVER_SECRET") {
		t.Fatalf("expected error for empty RECOVER_SECRET, got: %v", err)
	}

	// 4. Insecure webhook secret in production: must fail
	badWebhookCfg := validCfg
	badWebhookCfg.Webhooks = []config.Webhook{
		{URL: "https://hook.example.com", Secret: "change_me"},
	}
	if err := validateSecrets(badWebhookCfg, validLookup, true, nil); err == nil || !strings.Contains(err.Error(), "webhooks[0]") {
		t.Fatalf("expected error for CHANGE_ME webhook secret, got: %v", err)
	}

	// 5. Insecure PayTR merchant_key in production: must fail
	badPayTRCfg := validCfg
	badPayTRCfg.Payments.PayTR.MerchantKey = "CHANGE_ME"
	if err := validateSecrets(badPayTRCfg, validLookup, true, nil); err == nil || !strings.Contains(err.Error(), "payments.paytr.merchant_key") {
		t.Fatalf("expected error for CHANGE_ME paytr merchant_key, got: %v", err)
	}

	// 6. Havale enabled in production without HYDRO_PAYMENTS_HAVALE_IBAN env var: must fail
	havaleCfg := validCfg
	havaleCfg.Payments.Havale = config.Havale{
		IBAN:          "TR330006100519786457841326",
		BankName:      "Ziraat Bankası",
		AccountHolder: "BEYOND DERİ A.Ş.",
	}
	if err := validateSecrets(havaleCfg, validLookup, true, nil); err == nil || !strings.Contains(err.Error(), "HYDRO_PAYMENTS_HAVALE_IBAN") {
		t.Fatalf("expected error when Havale enabled without HYDRO_PAYMENTS_HAVALE_IBAN, got: %v", err)
	}

	// 7. Havale with placeholder IBAN in production: must fail
	placeholderHavaleCfg := validCfg
	placeholderHavaleCfg.Payments.Havale = config.Havale{
		IBAN:          "TR00 0000 0000 0000 0000 0000 00",
		BankName:      "TODO Banka",
		AccountHolder: "BEYOND",
	}
	havaleLookup := func(key string) (string, bool) {
		if key == "HYDRO_PAYMENTS_HAVALE_IBAN" {
			return "TR00 0000 0000 0000 0000 0000 00", true
		}
		return validLookup(key)
	}
	if err := validateSecrets(placeholderHavaleCfg, havaleLookup, true, nil); err == nil || !strings.Contains(err.Error(), "payments.havale.iban") {
		t.Fatalf("expected error for placeholder Havale IBAN, got: %v", err)
	}

	// 8. Havale with env var and valid fields in production: must pass
	validHavaleLookup := func(key string) (string, bool) {
		if key == "HYDRO_PAYMENTS_HAVALE_IBAN" {
			return "TR330006100519786457841326", true
		}
		return validLookup(key)
	}
	if err := validateSecrets(havaleCfg, validHavaleLookup, true, nil); err != nil {
		t.Fatalf("expected valid env-backed Havale config to pass in production, got: %v", err)
	}

	// 9. Analytics local-dev-key in production: must fail
	badAnalyticsCfg := validCfg
	badAnalyticsCfg.Analytics = config.Analytics{
		NoCookieEnabled: true,
		HashKey:         "beyond-analytics-hash-secret-2026-local-dev-key",
	}
	if err := validateSecrets(badAnalyticsCfg, validLookup, true, nil); err == nil || !strings.Contains(err.Error(), "analytics.hash_key") {
		t.Fatalf("expected error for local-dev analytics.hash_key, got: %v", err)
	}
}

func TestValidateSecretsInDevelopmentWarnsInsteadOfFailing(t *testing.T) {
	insecureCfg := config.Config{
		Payments: config.Payments{
			PayTR: config.PayTR{
				MerchantID:   "m-1",
				MerchantKey:  "CHANGE_ME",
				MerchantSalt: "",
				BaseURL:      "https://paytr.example.com",
			},
		},
		Webhooks: []config.Webhook{
			{URL: "https://hook.example.com", Secret: "CHANGE_ME"},
		},
	}

	emptyLookup := func(key string) (string, bool) {
		return "", false
	}

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	// In development (isProd = false), must return nil error and log warnings
	err := validateSecrets(insecureCfg, emptyLookup, false, logger)
	if err != nil {
		t.Fatalf("expected development environment to allow startup with warnings, got error: %v", err)
	}

	logOutput := buf.String()
	if !strings.Contains(logOutput, "HYDRO_JWT_SECRET") || !strings.Contains(logOutput, "RECOVER_SECRET") {
		t.Fatalf("expected warning log about insecure secrets, got: %s", logOutput)
	}
}
