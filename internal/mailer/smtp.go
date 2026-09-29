// Package mailer provides bounded outbound notification adapters.
package mailer

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/guards"
)

type Message struct {
	To          string
	Subject     string
	Body        string
	HTMLBody    string
	Attachments []Attachment
}

type Attachment struct {
	Filename, ContentType string
	Data                  []byte
}

type Sender interface {
	Send(context.Context, Message) error
}

var (
	ErrSenderRequired = errors.New("notification sender is required")
	ErrInvalidMessage = errors.New("invalid notification message")
)

// NotificationSender adapts password-reset and invitation events to the
// application notifier interfaces without logging or persisting raw tokens.
type NotificationSender struct{ Sender Sender }

func (n NotificationSender) SendPasswordReset(ctx context.Context, email, token string) error {
	if guards.IsNil(n.Sender) {
		return ErrSenderRequired
	}
	return n.Sender.Send(ctx, Message{To: email, Subject: "HydroGo password reset", Body: "Password reset token: " + token})
}

func (n NotificationSender) SendInvite(ctx context.Context, email, token string) error {
	if guards.IsNil(n.Sender) {
		return ErrSenderRequired
	}
	return n.Sender.Send(ctx, Message{To: email, Subject: "HydroGo invitation", Body: "Invitation token: " + token})
}

type SMTPConfig struct {
	Host         string
	Port         int
	From         string
	Username     string
	Password     string
	StartTLS     bool
	Timeout      time.Duration
	MaxPerSecond int
	MaxPerHour   int
}

type SMTP struct {
	config      SMTPConfig
	sendLimiter sendLimiter
}

type PingResult struct {
	Stage         string
	TLSVersion    string
	Authenticated bool
	Elapsed       time.Duration
}

func NewSMTP(config SMTPConfig) (*SMTP, error) {
	if strings.TrimSpace(config.Host) == "" || strings.TrimSpace(config.From) == "" {
		return nil, errors.New("SMTP host and from are required")
	}
	if config.Port <= 0 || config.Port > 65535 {
		return nil, errors.New("SMTP port is invalid")
	}
	if strings.ContainsAny(config.From, "\r\n") {
		return nil, errors.New("SMTP from contains a newline")
	}
	if config.Username == "" && config.Password != "" {
		return nil, errors.New("SMTP password requires username")
	}
	if config.Username != "" && !config.StartTLS {
		return nil, errors.New("SMTP AUTH requires STARTTLS")
	}
	if config.MaxPerSecond < 0 || config.MaxPerSecond > 3600 || config.MaxPerHour < 0 || config.MaxPerHour > 100_000 {
		return nil, errors.New("SMTP rate limits are invalid")
	}
	if config.MaxPerSecond == 0 {
		config.MaxPerSecond = 5
	}
	if config.MaxPerHour == 0 {
		config.MaxPerHour = 100
	}
	return &SMTP{config: config, sendLimiter: sendLimiter{
		perSecond: newTokenBucket(float64(config.MaxPerSecond), float64(config.MaxPerSecond)),
		perHour:   newTokenBucket(float64(config.MaxPerHour)/3600, 1),
	}}, nil
}

// Ping checks SMTP greeting, configured TLS, and credentials without sending mail.
func (s *SMTP) Ping(ctx context.Context) (result PingResult, err error) {
	if s == nil {
		return result, errors.New("SMTP sender is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	timeout := s.config.Timeout
	if timeout <= 0 || timeout > time.Second {
		timeout = time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	started := time.Now()
	defer func() { result.Elapsed = time.Since(started) }()
	result.Stage = "connect"
	conn, err := (&net.Dialer{Timeout: timeout}).DialContext(ctx, "tcp", net.JoinHostPort(s.config.Host, strconv.Itoa(s.config.Port)))
	if err != nil {
		return result, err
	}
	defer conn.Close()
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}
	result.Stage = "greeting"
	client, err := smtp.NewClient(conn, s.config.Host)
	if err != nil {
		return result, err
	}
	defer client.Close()
	if s.config.StartTLS {
		result.Stage = "tls"
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return result, errors.New("STARTTLS is unavailable")
		}
		if err := client.StartTLS(&tls.Config{ServerName: s.config.Host, MinVersion: tls.VersionTLS12}); err != nil {
			return result, err
		}
		if state, ok := client.TLSConnectionState(); ok {
			result.TLSVersion = tlsVersionName(state.Version)
		}
	}
	if s.config.Username != "" {
		result.Stage = "authentication"
		ok, authMechanisms := client.Extension("AUTH")
		if !ok {
			return result, errors.New("SMTP AUTH is unavailable")
		}
		var auth smtp.Auth
		switch {
		case strings.Contains(strings.ToUpper(authMechanisms), "PLAIN"):
			auth = smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Host)
		case strings.Contains(strings.ToUpper(authMechanisms), "LOGIN"):
			auth = &smtpLoginAuth{username: s.config.Username, password: s.config.Password}
		default:
			return result, errors.New("SMTP AUTH LOGIN/PLAIN is unavailable")
		}
		if err := client.Auth(auth); err != nil {
			return result, err
		}
		result.Authenticated = true
	}
	result.Stage = "complete"
	if err := client.Quit(); err != nil {
		return result, err
	}
	return result, nil
}

type smtpLoginAuth struct {
	username, password string
	step               int
}

func (a *smtpLoginAuth) Start(*smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", nil, nil
}

func (a *smtpLoginAuth) Next(_ []byte, more bool) ([]byte, error) {
	if !more {
		return nil, nil
	}
	if a.step == 0 {
		a.step++
		return []byte(a.username), nil
	}
	if a.step == 1 {
		a.step++
		return []byte(a.password), nil
	}
	return nil, errors.New("unexpected SMTP LOGIN challenge")
}

func tlsVersionName(version uint16) string {
	switch version {
	case tls.VersionTLS13:
		return "TLS 1.3"
	case tls.VersionTLS12:
		return "TLS 1.2"
	default:
		return "TLS"
	}
}

func (s *SMTP) Send(ctx context.Context, message Message) error {
	if s == nil {
		return errors.New("SMTP sender is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validateMessage(message); err != nil {
		return err
	}
	if err := s.sendLimiter.Wait(ctx); err != nil {
		return err
	}
	timeout := s.config.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	operationCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	address := net.JoinHostPort(s.config.Host, strconv.Itoa(s.config.Port))
	conn, err := (&net.Dialer{}).DialContext(operationCtx, "tcp", address)
	if err != nil {
		return fmt.Errorf("SMTP dial: %w", err)
	}
	defer conn.Close()
	if deadline, ok := operationCtx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}
	client, err := smtp.NewClient(conn, s.config.Host)
	if err != nil {
		return fmt.Errorf("SMTP client: %w", err)
	}
	defer client.Close()
	if s.config.StartTLS {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return errors.New("SMTP server does not support STARTTLS")
		}
		if err := client.StartTLS(&tls.Config{ServerName: s.config.Host, MinVersion: tls.VersionTLS12}); err != nil {
			return fmt.Errorf("SMTP STARTTLS: %w", err)
		}
	}
	if s.config.Username != "" {
		if ok, _ := client.Extension("AUTH"); !ok {
			return errors.New("SMTP server does not support AUTH")
		}
		if err := client.Auth(smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Host)); err != nil {
			return fmt.Errorf("SMTP auth: %w", err)
		}
	}
	if err := client.Mail(s.config.From); err != nil {
		return fmt.Errorf("SMTP MAIL FROM: %w", err)
	}
	if err := client.Rcpt(message.To); err != nil {
		return fmt.Errorf("SMTP RCPT TO: %w", err)
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA: %w", err)
	}
	if _, err := io.WriteString(writer, formatMessage(s.config.From, message)); err != nil {
		_ = writer.Close()
		return fmt.Errorf("SMTP message: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("SMTP message close: %w", err)
	}
	if err := client.Quit(); err != nil {
		return fmt.Errorf("SMTP quit: %w", err)
	}
	return nil
}

type tokenBucket struct {
	mu       sync.Mutex
	capacity float64
	rate     float64
	tokens   float64
	updated  time.Time
}

func newTokenBucket(rate, capacity float64) tokenBucket {
	return tokenBucket{capacity: capacity, rate: rate, tokens: capacity}
}

func (b *tokenBucket) reserve(now time.Time) time.Duration {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.updated.IsZero() {
		b.updated = now
	}
	if now.After(b.updated) {
		b.tokens += now.Sub(b.updated).Seconds() * b.rate
		if b.tokens > b.capacity {
			b.tokens = b.capacity
		}
		b.updated = now
	}
	if b.tokens >= 1 {
		b.tokens--
		return 0
	}
	wait := time.Duration((1 - b.tokens) / b.rate * float64(time.Second))
	b.tokens = 0
	b.updated = b.updated.Add(wait)
	if delay := b.updated.Sub(now); delay > 0 {
		return delay
	}
	return wait
}

type sendLimiter struct {
	perSecond tokenBucket
	perHour   tokenBucket
}

func (l *sendLimiter) Wait(ctx context.Context) error {
	now := time.Now()
	delay := l.perSecond.reserve(now)
	if hourDelay := l.perHour.reserve(now); hourDelay > delay {
		delay = hourDelay
	}
	if delay <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return ctx.Err()
	}
}

func validateMessage(message Message) error {
	parsed, parseErr := mail.ParseAddress(strings.TrimSpace(message.To))
	if parseErr != nil || parsed.Address != strings.TrimSpace(message.To) || strings.TrimSpace(message.Subject) == "" {
		return fmt.Errorf("%w: recipient and subject are required", ErrInvalidMessage)
	}
	if strings.ContainsAny(message.To+message.Subject, "\r\n") {
		return fmt.Errorf("%w: header contains a newline", ErrInvalidMessage)
	}
	if len(message.Body) > 1<<20 || len(message.HTMLBody) > 1<<20 {
		return fmt.Errorf("%w: body is too large", ErrInvalidMessage)
	}
	if len(message.Attachments) > 4 {
		return fmt.Errorf("%w: too many attachments", ErrInvalidMessage)
	}
	var attachmentBytes int
	for _, attachment := range message.Attachments {
		mediaType, _, typeErr := mime.ParseMediaType(attachment.ContentType)
		if strings.TrimSpace(attachment.Filename) == "" || len(attachment.Filename) > 128 || strings.ContainsAny(attachment.Filename, "\\/:\r\n\x00") || typeErr != nil || mediaType == "" || len(attachment.Data) == 0 || len(attachment.Data) > 8<<20 || attachmentBytes > (8<<20)-len(attachment.Data) {
			return fmt.Errorf("%w: attachment is invalid or too large", ErrInvalidMessage)
		}
		attachmentBytes += len(attachment.Data)
	}
	return nil
}

func formatMessage(from string, message Message) string {
	headers := "From: " + from + "\r\n" +
		"To: " + message.To + "\r\n" +
		"Subject: " + mime.QEncoding.Encode("UTF-8", message.Subject) + "\r\n" +
		"Date: " + time.Now().UTC().Format(time.RFC1123Z) + "\r\n" +
		"MIME-Version: 1.0\r\n"
	if len(message.Attachments) > 0 {
		var body strings.Builder
		writer := multipart.NewWriter(&body)
		_, _ = io.WriteString(&body, headers+"Content-Type: multipart/mixed; boundary=\""+writer.Boundary()+"\"\r\n\r\n")
		if message.HTMLBody == "" {
			part, _ := writer.CreatePart(textproto.MIMEHeader{"Content-Type": {"text/plain; charset=utf-8"}, "Content-Transfer-Encoding": {"quoted-printable"}})
			encoded := quotedprintable.NewWriter(part)
			_, _ = io.WriteString(encoded, message.Body)
			_ = encoded.Close()
		} else {
			var alternative bytes.Buffer
			alternativeWriter := multipart.NewWriter(&alternative)
			writeQuotedPart(alternativeWriter, "text/plain; charset=utf-8", message.Body)
			writeQuotedPart(alternativeWriter, "text/html; charset=utf-8", message.HTMLBody)
			_ = alternativeWriter.Close()
			part, _ := writer.CreatePart(textproto.MIMEHeader{"Content-Type": {"multipart/alternative; boundary=\"" + alternativeWriter.Boundary() + "\""}})
			_, _ = part.Write(alternative.Bytes())
		}
		for _, attachment := range message.Attachments {
			params := mime.FormatMediaType("attachment", map[string]string{"filename": attachment.Filename})
			part, _ := writer.CreatePart(textproto.MIMEHeader{"Content-Type": {attachment.ContentType}, "Content-Transfer-Encoding": {"base64"}, "Content-Disposition": {params}})
			encoded := base64.NewEncoder(base64.StdEncoding, part)
			_, _ = encoded.Write(attachment.Data)
			_ = encoded.Close()
		}
		_ = writer.Close()
		return body.String()
	}
	if message.HTMLBody == "" {
		return headers + "Content-Type: text/plain; charset=utf-8\r\n\r\n" + message.Body + "\r\n"
	}
	var body strings.Builder
	writer := multipart.NewWriter(&body)
	_, _ = io.WriteString(&body, headers+"Content-Type: multipart/alternative; boundary=\""+writer.Boundary()+"\"\r\n\r\n")
	writeQuotedPart(writer, "text/plain; charset=utf-8", message.Body)
	writeQuotedPart(writer, "text/html; charset=utf-8", message.HTMLBody)
	_ = writer.Close()
	return body.String()
}

func writeQuotedPart(writer *multipart.Writer, contentType, value string) {
	part, err := writer.CreatePart(textproto.MIMEHeader{"Content-Type": {contentType}, "Content-Transfer-Encoding": {"quoted-printable"}})
	if err != nil {
		return
	}
	encoded := quotedprintable.NewWriter(part)
	_, _ = io.WriteString(encoded, value)
	_ = encoded.Close()
}

func (n NotificationSender) SendEmailVerification(ctx context.Context, email, link string) error {
	if guards.IsNil(n.Sender) {
		return ErrSenderRequired
	}
	var html bytes.Buffer
	if err := EmailVerificationEmail(link).Render(ctx, &html); err != nil {
		return err
	}
	return n.Sender.Send(ctx, Message{To: email, Subject: "Confirm your HydroGo email address", Body: "Confirm your email address using this one-time link:\n" + link, HTMLBody: html.String()})
}
