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
