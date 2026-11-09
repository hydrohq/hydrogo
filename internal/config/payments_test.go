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
		{"yurtici complete", Carriers{Yurtici: YurticiCarrier{WSUserName: "u", WSPassword: "p", BaseURL: "https://soap.example"}}, false},
		{"unknown default", Carriers{Default: "ups"}, true},
	}
	for _, tc := range cases {
		err := tc.c.validate()
		if (err != nil) != tc.wantErr {
			t.Errorf("%s: err = %v, wantErr %v", tc.name, err, tc.wantErr)
		}
	}
}
