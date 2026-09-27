package mailer

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"sort"
	"strings"
	"time"
)

// MXSender delivers through a recipient domain's MX records. It is intended
// only for explicitly configured deployments without a submission provider.
// MX delivery requires outbound port 25 and recipient-domain acceptance; the
// durable spool remains the fallback when that path is unavailable.
type MXSender struct {
	From      string
	Resolver  *net.Resolver
	Dialer    *net.Dialer
	Timeout   time.Duration
	LookupMX  func(context.Context, string) ([]*net.MX, error)
	DeliverMX func(context.Context, string, string, Message) error
}

func (s MXSender) Send(ctx context.Context, message Message) error {
	if ctx == nil {
		ctx = context.Background()
	}
	operationCtx, cancel := s.withTimeout(ctx)
	defer cancel()
	if err := validateMessage(message); err != nil {
		return err
	}
	from, err := mail.ParseAddress(strings.TrimSpace(s.From))
	if err != nil || from.Address != strings.TrimSpace(s.From) || strings.ContainsAny(s.From, "\r\n") {
		return fmt.Errorf("%w: valid configured sender address is required", ErrInvalidMessage)
	}
	to, err := mail.ParseAddress(message.To)
	if err != nil {
		return fmt.Errorf("%w: invalid recipient", ErrInvalidMessage)
	}
	domain := strings.ToLower(strings.TrimSpace(to.Address[strings.LastIndex(to.Address, "@")+1:]))
	lookup := s.LookupMX
	if lookup == nil {
		resolver := s.Resolver
		if resolver == nil {
			resolver = net.DefaultResolver
		}
		lookup = resolver.LookupMX
	}
	records, err := lookup(operationCtx, domain)
	if err != nil {
		return fmt.Errorf("mail recipient MX lookup: %w", err)
	}
	if len(records) == 0 {
		return errors.New("mail recipient domain has no usable MX records")
	}
	records = append([]*net.MX(nil), records...)
	sort.Slice(records, func(i, j int) bool { return records[i].Pref < records[j].Pref })
	if len(records) > 32 {
		records = records[:32]
	}
	dialer := s.Dialer
	if dialer == nil {
		dialer = &net.Dialer{}
	}
	var failures []string
	for _, record := range records {
		if record == nil || strings.TrimSpace(record.Host) == "" || strings.ContainsAny(record.Host, "\r\n/\\") {
			continue
		}
		host := strings.TrimSuffix(record.Host, ".")
		var dialErr error
		if s.DeliverMX != nil {
			dialErr = s.DeliverMX(operationCtx, host, from.Address, message)
		} else {
			conn, err := dialer.DialContext(operationCtx, "tcp", net.JoinHostPort(host, "25"))
			dialErr = err
			if dialErr == nil {
				dialErr = s.sendOnConnection(operationCtx, conn, host, from.Address, message)
			}
		}
		if dialErr == nil {
			return nil
		}
		failures = append(failures, host+": "+dialErr.Error())
	}
	if len(failures) == 0 {
		return errors.New("mail recipient domain has no usable MX records")
	}
	return fmt.Errorf("mail direct MX delivery failed: %s", strings.Join(failures, "; "))
}

func (s MXSender) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	timeout := s.Timeout
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	return context.WithTimeout(ctx, timeout)
}

func (s MXSender) sendOnConnection(ctx context.Context, conn net.Conn, host, from string, message Message) error {
	defer conn.Close()
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer client.Close()
	if ok, _ := client.Extension("STARTTLS"); !ok {
		return errors.New("recipient MX does not offer STARTTLS")
	}
	if err := client.StartTLS(&tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}); err != nil {
		return err
	}
	if err := client.Mail(from); err != nil {
		return err
	}
	if err := client.Rcpt(message.To); err != nil {
		return err
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := fmt.Fprint(writer, formatMessage(s.From, message)); err != nil {
		_ = writer.Close()
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return client.Quit()
}
