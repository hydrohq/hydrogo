package mailer

import (
	"context"
	"net"
	"testing"
)

type fakeDNS struct{ txt map[string][]string }

func (f fakeDNS) LookupTXT(_ context.Context, name string) ([]string, error) { return f.txt[name], nil }
func (fakeDNS) LookupMX(context.Context, string) ([]*net.MX, error) {
	return []*net.MX{{Host: "mx.example.test."}}, nil
}

func TestDiagnoseDNSChecksConfiguredSenderDomainAndSelector(t *testing.T) {
	checks, err := DiagnoseDNS(context.Background(), `Hydro <mail@example.test>`, "selector1", fakeDNS{txt: map[string][]string{
		"example.test":                      {"v=spf1 include:mail.example.test -all"},
		"_dmarc.example.test":               {"v=DMARC1; p=reject"},
		"selector1._domainkey.example.test": {"v=DKIM1; p=abc"},
	}})
	if err != nil || len(checks) != 4 {
		t.Fatalf("checks=%+v err=%v", checks, err)
	}
	for _, check := range checks {
		if !check.OK {
			t.Errorf("%s should pass: %+v", check.Name, check)
		}
	}
}

func TestDiagnoseDNSRejectsMalformedSenderAndOffersGuidance(t *testing.T) {
	if _, err := DiagnoseDNS(context.Background(), "not-an-address", "", fakeDNS{}); err == nil {
		t.Fatal("malformed sender accepted")
	}
	checks, err := DiagnoseDNS(context.Background(), "mailer@example.test", "", fakeDNS{})
	if err != nil {
		t.Fatal(err)
	}
	for _, check := range checks[:3] {
		if check.OK || check.Guidance == "" {
			t.Errorf("missing guidance for %s: %+v", check.Name, check)
		}
	}
}
