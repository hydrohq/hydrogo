package config

import "testing"

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
