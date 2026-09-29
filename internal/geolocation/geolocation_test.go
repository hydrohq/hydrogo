package geolocation

import (
	"net/http/httptest"
	"net/netip"
	"testing"
)

type databaseStub struct{ location Location }

func (d databaseStub) Lookup(netip.Addr) (Location, bool) { return d.location, true }

func TestResolveClassifiesPrivateAndNormalizesDatabaseResults(t *testing.T) {
	db := databaseStub{location: Location{CountryCode: " tr ", City: " Istanbul "}}
	resolver := Resolver{Database: db}
	if got := resolver.Resolve(netip.MustParseAddr("192.168.1.5")); !got.Local {
		t.Fatalf("private location = %+v", got)
	}
	if got := resolver.Resolve(netip.MustParseAddr("8.8.8.8")); got.CountryCode != "TR" || got.City != "Istanbul" {
		t.Fatalf("database location = %+v", got)
	}
}

func TestResolveRequestTrustsProxyHeadersOnlyFromConfiguredPeers(t *testing.T) {
	prefixes, err := ParseTrustedProxyRanges([]string{"192.0.2.0/24"})
	if err != nil {
		t.Fatal(err)
	}
	resolver := Resolver{Database: databaseStub{location: Location{CountryCode: "US", City: "Test City"}}, TrustedProxyRanges: prefixes}
	request := httptest.NewRequest("GET", "/", nil)
	request.RemoteAddr = "203.0.113.5:1234"
	request.Header.Set("CF-Connecting-IP", "8.8.8.8")
	request.Header.Set("CF-IPCountry", "TR")
	if got := resolver.ResolveRequest(request); got.CountryCode != "US" {
		t.Fatalf("untrusted headers affected location: %+v", got)
	}
	request.RemoteAddr = "192.0.2.10:1234"
	if got := resolver.ResolveRequest(request); got.CountryCode != "US" || got.City != "Test City" {
		t.Fatalf("trusted client lookup = %+v", got)
	}
	request.Header.Set("CF-Connecting-IP", "bad")
	if got := (Resolver{TrustedProxyRanges: prefixes}).ResolveRequest(request); got.CountryCode != "TR" {
		t.Fatalf("trusted country fallback = %+v", got)
	}
}

func TestProxyRangesAndCountryCodeValidation(t *testing.T) {
	if _, err := ParseTrustedProxyRanges([]string{"not-a-range"}); err == nil {
		t.Fatal("invalid proxy CIDR accepted")
	}
	if got := normalizeCountryCode(" t1 "); got != "" {
		t.Fatalf("reserved code = %q", got)
	}
	if got := CountryFlag("TR"); got != "🇹🇷" {
		t.Fatalf("flag = %q", got)
	}
}
