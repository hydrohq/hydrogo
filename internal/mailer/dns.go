package mailer

import (
	"context"
	"errors"
	"net"
	"net/mail"
	"strings"
	"time"
)

type DNSRecords interface {
	LookupTXT(context.Context, string) ([]string, error)
	LookupMX(context.Context, string) ([]*net.MX, error)
}

type DNSCheck struct {
	Name, Detail, Guidance string
	OK                     bool
}

type netDNSRecords struct{}

func (netDNSRecords) LookupTXT(ctx context.Context, name string) ([]string, error) {
	return net.DefaultResolver.LookupTXT(ctx, name)
}
func (netDNSRecords) LookupMX(ctx context.Context, name string) ([]*net.MX, error) {
	return net.DefaultResolver.LookupMX(ctx, name)
}

func DiagnoseDNS(ctx context.Context, from, selector string, records DNSRecords) ([]DNSCheck, error) {
	address, err := mail.ParseAddress(strings.TrimSpace(from))
	if err != nil {
		return nil, errors.New("configured sender address is invalid")
	}
	parts := strings.Split(address.Address, "@")
	if len(parts) != 2 || parts[1] == "" {
		return nil, errors.New("configured sender domain is invalid")
	}
	domain := strings.ToLower(strings.TrimSuffix(parts[1], "."))
	if records == nil {
		records = netDNSRecords{}
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	spf, _ := records.LookupTXT(ctx, domain)
	checks := []DNSCheck{{Name: "SPF", Guidance: "Add a TXT record beginning with v=spf1 at " + domain}}
	for _, value := range spf {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(value)), "v=spf1") {
			checks[0].OK, checks[0].Detail, checks[0].Guidance = true, "SPF record found", ""
			break
		}
	}
	dmarc, _ := records.LookupTXT(ctx, "_dmarc."+domain)
	checks = append(checks, DNSCheck{Name: "DMARC", Guidance: "Add a TXT record at _dmarc." + domain + " beginning with v=DMARC1"})
	for _, value := range dmarc {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(value)), "v=dmarc1") {
			checks[1].OK, checks[1].Detail, checks[1].Guidance = true, "DMARC record found", ""
			break
		}
	}
	selector = strings.TrimSpace(selector)
	if selector == "" || len(selector) > 63 || strings.ContainsAny(selector, ". \t\r\n") {
		selector = "default"
	}
	dkim, _ := records.LookupTXT(ctx, selector+"._domainkey."+domain)
	checks = append(checks, DNSCheck{Name: "DKIM", Guidance: "Publish the matching DKIM public key at " + selector + "._domainkey." + domain})
	for _, value := range dkim {
		if strings.Contains(strings.ToLower(value), "v=dkim1") || strings.Contains(strings.ToLower(value), "p=") {
			checks[2].OK, checks[2].Detail, checks[2].Guidance = true, "DKIM key found", ""
			break
		}
	}
	_, mxErr := records.LookupMX(ctx, domain)
	checks = append(checks, DNSCheck{Name: "MX", OK: mxErr == nil, Detail: "MX records found"})
	if mxErr != nil {
		checks[3].Guidance = "Verify that " + domain + " publishes the mail provider's MX records"
	}
	return checks, nil
}
