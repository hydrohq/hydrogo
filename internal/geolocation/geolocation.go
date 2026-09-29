// Package geolocation resolves coarse request locations from an operator-managed
// offline database and explicitly trusted proxy metadata.
package geolocation

import (
	"errors"
	"net/http"
	"net/netip"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/requestmeta"
	"github.com/oschwald/maxminddb-golang/v2"
)

type Location struct {
	CountryCode string
	City        string
	Local       bool
}

type Database interface {
	Lookup(netip.Addr) (Location, bool)
}

type Resolver struct {
	Database           Database
	TrustedProxyRanges []netip.Prefix
}

func ParseTrustedProxyRanges(values []string) ([]netip.Prefix, error) {
	if len(values) > 256 {
		return nil, ErrInvalidProxyRange
	}
	prefixes := make([]netip.Prefix, 0, len(values))
	for _, value := range values {
		prefix, err := netip.ParsePrefix(strings.TrimSpace(value))
		if err != nil {
			return nil, ErrInvalidProxyRange
		}
		prefixes = append(prefixes, prefix.Masked())
	}
	return prefixes, nil
}

var ErrInvalidProxyRange = errors.New("invalid trusted proxy range")

func (r Resolver) Resolve(ip netip.Addr) Location {
	ip = ip.Unmap()
	if !ip.IsValid() {
		return Location{}
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || !ip.IsGlobalUnicast() {
		return Location{Local: true}
	}
	if r.Database == nil {
		return Location{}
	}
	location, ok := r.Database.Lookup(ip)
	if !ok {
		return Location{}
	}
	location.CountryCode = normalizeCountryCode(location.CountryCode)
	location.City = strings.TrimSpace(location.City)
	if len(location.City) > 128 {
		location.City = location.City[:128]
	}
	if location.Local || location.CountryCode == "" {
		return Location{}
	}
	return location
}

func (r Resolver) IsTrustedProxy(ip netip.Addr) bool {
	ip = ip.Unmap()
	if !ip.IsValid() {
		return false
	}
	for _, prefix := range r.TrustedProxyRanges {
		if prefix.Contains(ip) {
			return true
		}
	}
	return false
}

func (r Resolver) ResolveRequest(request *http.Request) Location {
	if request == nil {
		return Location{}
	}
	remote, err := netip.ParseAddr(requestmeta.ClientIP(request))
	trusted := err == nil && r.IsTrustedProxy(remote)
	clientIP := remote
	if trusted {
		clientIP = netip.Addr{}
		if forwarded, parseErr := netip.ParseAddr(strings.TrimSpace(request.Header.Get("CF-Connecting-IP"))); parseErr == nil {
			clientIP = forwarded
		}
	}
	location := r.Resolve(clientIP)
	if location.Local || location.CountryCode != "" || !trusted {
		return location
	}
	location.CountryCode = normalizeCountryCode(request.Header.Get("CF-IPCountry"))
	return location
}

func normalizeCountryCode(code string) string {
	code = strings.ToUpper(strings.TrimSpace(code))
	if len(code) != 2 || code[0] < 'A' || code[0] > 'Z' || code[1] < 'A' || code[1] > 'Z' || code == "XX" || code == "T1" {
		return ""
	}
	return code
}

func CountryFlag(code string) string {
	code = normalizeCountryCode(code)
	if code == "" {
		return ""
	}
	return string([]rune{rune(0x1F1E6 + uint32(code[0]-'A')), rune(0x1F1E6 + uint32(code[1]-'A'))})
}

type MaxMindDatabase struct{ reader *maxminddb.Reader }

func OpenMaxMind(path string) (*MaxMindDatabase, error) {
	reader, err := maxminddb.Open(path)
	if err != nil {
		return nil, err
	}
	if err := reader.Verify(); err != nil {
		_ = reader.Close()
		return nil, err
	}
	return &MaxMindDatabase{reader: reader}, nil
}

func (db *MaxMindDatabase) Close() error {
	if db == nil || db.reader == nil {
		return nil
	}
	return db.reader.Close()
}

func (db *MaxMindDatabase) Lookup(ip netip.Addr) (Location, bool) {
	if db == nil || db.reader == nil || !ip.IsValid() {
		return Location{}, false
	}
	var record struct {
		Country struct {
			ISOCode string            `maxminddb:"iso_code"`
			Names   map[string]string `maxminddb:"names"`
		} `maxminddb:"country"`
		RegisteredCountry struct {
			ISOCode string `maxminddb:"iso_code"`
		} `maxminddb:"registered_country"`
		City struct {
			Names map[string]string `maxminddb:"names"`
		} `maxminddb:"city"`
	}
	result := db.reader.Lookup(ip)
	if result.Decode(&record) != nil || !result.Found() {
		return Location{}, false
	}
	code := record.Country.ISOCode
	if code == "" {
		code = record.RegisteredCountry.ISOCode
	}
	city := record.City.Names["en"]
	if city == "" {
		city = record.City.Names["tr"]
	}
	location := Location{CountryCode: normalizeCountryCode(code), City: city}
	return location, location.CountryCode != ""
}
