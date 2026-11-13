package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPaymentsValidation(t *testing.T) {
	cases := []struct {
		name    string
		p       Payments
		wantErr bool
	}{
		{"empty is valid", Payments{}, false},
		{"paytr needs key, salt and base_url", Payments{PayTR: PayTR{MerchantID: "1"}}, true},
		{"paytr complete", Payments{PayTR: PayTR{MerchantID: "1", MerchantKey: "k", MerchantSalt: "s", BaseURL: "https://gw.example"}}, false},
		{"paytr timeout minimum", Payments{PayTR: PayTR{TimeoutLimit: 5}}, false},
		{"paytr timeout maximum", Payments{PayTR: PayTR{TimeoutLimit: 29}}, false},
		{"paytr timeout below minimum", Payments{PayTR: PayTR{TimeoutLimit: 4}}, true},
		{"paytr timeout reaches reservation expiry", Payments{PayTR: PayTR{TimeoutLimit: 30}}, true},
		{"paytr timeout exceeds reservation expiry", Payments{PayTR: PayTR{TimeoutLimit: 31}}, true},
		{"iyzico needs secret, callback and base_url", Payments{Iyzico: Iyzico{APIKey: "a"}}, true},
		{"iyzico complete", Payments{Iyzico: Iyzico{APIKey: "a", SecretKey: "s", CallbackURL: "https://x/cb", BaseURL: "https://gw.example"}}, false},
		{"unknown default", Payments{Default: "stripe"}, true},
		{"havale default", Payments{Default: "havale"}, false},
	}
	for _, tc := range cases {
		if err := tc.p.validate(); (err != nil) != tc.wantErr {
			t.Errorf("%s: err = %v, wantErr %v", tc.name, err, tc.wantErr)
		}
	}
}

func TestCarriersValidation(t *testing.T) {
	cases := []struct {
		name    string
		c       Carriers
		wantErr bool
	}{
		{"empty is valid", Carriers{}, false},
		{"manual only", Carriers{Manual: ManualCarrier{Enabled: true}}, false},
		{"yurtici needs password and base_url", Carriers{Yurtici: YurticiCarrier{WSUserName: "u"}}, true},
		{"yurtici complete", Carriers{Yurtici: YurticiCarrier{WSUserName: "u", WSPassword: "p", BaseURL: "https://soap.example", Namespace: "urn:example"}}, false},
		{"http needs name, template and path", Carriers{HTTP: HTTPCarrier{URL: "https://c.example"}}, true},
		{"http complete", Carriers{HTTP: HTTPCarrier{URL: "https://c.example", Name: "X", BodyTemplate: "{}", TrackingPath: "a"}}, false},
		{"geliver default allowed", Carriers{Default: "geliver", Geliver: GeliverCarrier{Token: "tok"}}, false},
		{"geliver invalid base url", Carriers{Geliver: GeliverCarrier{Token: "tok", BaseURL: "not-a-url"}}, true},
		{"unknown default", Carriers{Default: "ups"}, true},
	}
	for _, tc := range cases {
		err := tc.c.validate()
		if (err != nil) != tc.wantErr {
			t.Errorf("%s: err = %v, wantErr %v", tc.name, err, tc.wantErr)
		}
	}
}

func TestLoadCarriersGeliverEnvOverrides(t *testing.T) {
	envMap := map[string]string{
		"HYDRO_DATABASE_DSN":                       "data/test.db",
		"HYDRO_CARRIERS_DEFAULT":                   "geliver",
		"HYDRO_CARRIERS_GELIVER_TOKEN":             "tok_env_123",
		"HYDRO_CARRIERS_GELIVER_SENDER_ADDRESS_ID": "addr_env_456",
		"HYDRO_CARRIERS_GELIVER_WEBHOOK_SECRET":    "wh_env_789",
		"HYDRO_CARRIERS_GELIVER_BASE_URL":          "https://sandbox.geliver.example",
		"HYDRO_CARRIERS_GELIVER_TEST_MODE":         "true",
	}
	lookup := func(key string) (string, bool) {
		v, ok := envMap[key]
		return v, ok
	}

	cfg, err := Load("", lookup, Overrides{})
	if err != nil {
		t.Fatalf("Load returned unexpected error: %v", err)
	}
	if cfg.Carriers.Default != "geliver" {
		t.Errorf("Carriers.Default = %q, want geliver", cfg.Carriers.Default)
	}
	if cfg.Carriers.Geliver.Token != "tok_env_123" ||
		cfg.Carriers.Geliver.SenderAddressID != "addr_env_456" ||
		cfg.Carriers.Geliver.WebhookSecret != "wh_env_789" ||
		cfg.Carriers.Geliver.BaseURL != "https://sandbox.geliver.example" ||
		!cfg.Carriers.Geliver.TestMode {
		t.Errorf("unexpected Geliver config from env: %+v", cfg.Carriers.Geliver)
	}

	envMap["HYDRO_CARRIERS_GELIVER_TEST_MODE"] = "not-a-bool"
	if _, err := Load("", lookup, Overrides{}); err == nil {
		t.Fatal("expected error for invalid HYDRO_CARRIERS_GELIVER_TEST_MODE, got nil")
	}
}

func TestLoadInvoicingBizimHesapEnvOverrides(t *testing.T) {
	envMap := map[string]string{
		"HYDRO_DATABASE_DSN":                      "data/test.db",
		"HYDRO_INVOICING_BIZIMHESAP_FIRM_ID":      "firm_env_1",
		"HYDRO_INVOICING_BIZIMHESAP_ENDPOINT_URL": "https://invoicing.example.test/addinvoice",
		"HYDRO_INVOICING_BIZIMHESAP_API_KEY":      "api_key_1",
		"HYDRO_INVOICING_BIZIMHESAP_TOKEN":        "token_1",
	}
	lookup := func(key string) (string, bool) {
		v, ok := envMap[key]
		return v, ok
	}

	cfg, err := Load("", lookup, Overrides{})
	if err != nil {
		t.Fatalf("Load returned unexpected error: %v", err)
	}
	if cfg.Invoicing.AutoIssueOnPaid {
		t.Fatal("AutoIssueOnPaid must default to false")
	}
	if cfg.Invoicing.BizimHesap.FirmID != "firm_env_1" ||
		cfg.Invoicing.BizimHesap.EndpointURL != "https://invoicing.example.test/addinvoice" ||
		cfg.Invoicing.BizimHesap.APIKey != "api_key_1" ||
		cfg.Invoicing.BizimHesap.Token != "token_1" {
		t.Fatalf("unexpected BizimHesap config from env: %+v", cfg.Invoicing.BizimHesap)
	}

	envMap["HYDRO_INVOICING_AUTO_ISSUE_ON_PAID"] = "true"
	cfg, err = Load("", lookup, Overrides{})
	if err != nil || !cfg.Invoicing.AutoIssueOnPaid {
		t.Fatalf("expected AutoIssueOnPaid=true with env override, got %v err=%v", cfg.Invoicing.AutoIssueOnPaid, err)
	}

	// Missing endpoint_url with firm_id set must fail validation
	delete(envMap, "HYDRO_INVOICING_BIZIMHESAP_ENDPOINT_URL")
	if _, err := Load("", lookup, Overrides{}); err == nil {
		t.Fatal("expected validation error when firm_id is set without endpoint_url")
	}
}

func TestLoadPaymentsHavaleAndCommerceEnvOverrides(t *testing.T) {
	envMap := map[string]string{
		"HYDRO_DATABASE_DSN":                            "data/test.db",
		"HYDRO_HTTP_PUBLIC_ORIGIN":                      "https://beyond.example.com",
		"HYDRO_SECURE_COOKIES":                          "true",
		"HYDRO_SECURITY_PUBLIC_CONTENT_RATE_PER_MINUTE": "12000",
		"HYDRO_COMMERCE_MAX_PENDING_ORDERS_PER_IP":      "150",
		"HYDRO_PAYMENTS_DEFAULT":                        "havale",
		"HYDRO_PAYMENTS_HAVALE_IBAN":                    "TR330006100519786457841326",
		"HYDRO_PAYMENTS_HAVALE_BANK_NAME":               "Ziraat Bankası",
		"HYDRO_PAYMENTS_HAVALE_ACCOUNT_HOLDER":          "BEYOND DERİ A.Ş.",
		"HYDRO_PAYMENTS_HAVALE_WHATSAPP":                "+905551112233",
		"HYDRO_PAYMENTS_HAVALE_DISCOUNT_PERCENT":        "7.5",
	}
	lookup := func(key string) (string, bool) {
		v, ok := envMap[key]
		return v, ok
	}

	cfg, err := Load("", lookup, Overrides{})
	if err != nil {
		t.Fatalf("Load returned unexpected error: %v", err)
	}
	if cfg.HTTP.PublicOrigin != "https://beyond.example.com" {
		t.Errorf("HTTP.PublicOrigin = %q, want https://beyond.example.com", cfg.HTTP.PublicOrigin)
	}
	if cfg.Security.PublicContentRatePerMinute != 12000 {
		t.Errorf("Security.PublicContentRatePerMinute = %d, want 12000", cfg.Security.PublicContentRatePerMinute)
	}
	if cfg.Commerce.MaxPendingOrdersPerIP != 150 {
		t.Errorf("Commerce.MaxPendingOrdersPerIP = %d, want 150", cfg.Commerce.MaxPendingOrdersPerIP)
	}
	if cfg.Payments.Default != "havale" ||
		cfg.Payments.Havale.IBAN != "TR330006100519786457841326" ||
		cfg.Payments.Havale.BankName != "Ziraat Bankası" ||
		cfg.Payments.Havale.AccountHolder != "BEYOND DERİ A.Ş." ||
		cfg.Payments.Havale.WhatsApp != "+905551112233" ||
		cfg.Payments.Havale.DiscountPercent != 7.5 {
		t.Errorf("unexpected Havale config from env: %+v", cfg.Payments)
	}

	envMap["HYDRO_COMMERCE_MAX_PENDING_ORDERS_PER_IP"] = "abc"
	if _, err := Load("", lookup, Overrides{}); err == nil {
		t.Fatal("expected error for invalid HYDRO_COMMERCE_MAX_PENDING_ORDERS_PER_IP")
	}
}

func TestHTTPSOriginRejectsPlaceholderHavaleAndDevAnalyticsKey(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("modules: [core, commerce, payments, analytics]\n"), 0o600); err != nil {
		t.Fatalf("write temp config: %v", err)
	}

	baseEnv := func(overrides map[string]string) LookupEnv {
		m := map[string]string{
			"HYDRO_DATABASE_DSN":       "data/test.db",
			"HYDRO_HTTP_PUBLIC_ORIGIN": "https://shop.example.com",
			"HYDRO_SECURE_COOKIES":     "true",
		}
		for k, v := range overrides {
			m[k] = v
		}
		return func(key string) (string, bool) {
			v, ok := m[key]
			return v, ok
		}
	}

	if _, err := Load(cfgPath, baseEnv(map[string]string{
		"HYDRO_PAYMENTS_HAVALE_IBAN":           "TR00 0000 0000 0000 0000 0000 00",
		"HYDRO_PAYMENTS_HAVALE_BANK_NAME":      "Garanti",
		"HYDRO_PAYMENTS_HAVALE_ACCOUNT_HOLDER": "BEYOND",
	}), Overrides{}); err == nil || !strings.Contains(err.Error(), "payments.havale.iban") {
		t.Fatalf("expected error for placeholder IBAN under HTTPS, got: %v", err)
	}

	if _, err := Load(cfgPath, baseEnv(map[string]string{
		"HYDRO_PAYMENTS_HAVALE_IBAN":           "TR330006100519786457841326",
		"HYDRO_PAYMENTS_HAVALE_BANK_NAME":      "TODO Banka",
		"HYDRO_PAYMENTS_HAVALE_ACCOUNT_HOLDER": "BEYOND",
	}), Overrides{}); err == nil || !strings.Contains(err.Error(), "payments.havale.bank_name") {
		t.Fatalf("expected error for placeholder bank_name under HTTPS, got: %v", err)
	}

	if _, err := Load(cfgPath, baseEnv(map[string]string{
		"HYDRO_ANALYTICS_NO_COOKIE_ENABLED": "true",
		"HYDRO_ANALYTICS_HASH_KEY":          "beyond-analytics-hash-secret-2026-local-dev-key",
	}), Overrides{}); err == nil || !strings.Contains(err.Error(), "analytics.hash_key") {
		t.Fatalf("expected error for local-dev analytics hash key under HTTPS, got: %v", err)
	}

	if _, err := Load(cfgPath, baseEnv(map[string]string{
		"HYDRO_PAYMENTS_HAVALE_IBAN":           "TR330006100519786457841326",
		"HYDRO_PAYMENTS_HAVALE_BANK_NAME":      "Ziraat Bankası",
		"HYDRO_PAYMENTS_HAVALE_ACCOUNT_HOLDER": "BEYOND DERİ A.Ş.",
		"HYDRO_ANALYTICS_NO_COOKIE_ENABLED":    "true",
		"HYDRO_ANALYTICS_HASH_KEY":             "a1b2c3d4e5f60718293a4b5c6d7e8f9012345678",
	}), Overrides{}); err != nil {
		t.Fatalf("expected valid HTTPS Havale + Analytics config to pass, got: %v", err)
	}
}
