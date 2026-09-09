// Package mailer provides bounded outbound notification adapters.
package mailer

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/guards"
)

type Message struct {
	To      string
	Subject string
	Body    string
}

type Sender interface {
	Send(context.Context, Message) error
}

var ErrSenderRequired = errors.New("notification sender is required")

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
	Host     string
	Port     int
	From     string
	Username string
	Password string
	StartTLS bool
	Timeout  time.Duration
}

type SMTP struct {
	config SMTPConfig
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
	return &SMTP{config: config}, nil
}

func (s *SMTP) Send(ctx context.Context, message Message) error {
	if s == nil {
		return errors.New("SMTP sender is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validateMessage(message); err != nil {
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

func validateMessage(message Message) error {
	if strings.TrimSpace(message.To) == "" || strings.TrimSpace(message.Subject) == "" {
		return errors.New("notification recipient and subject are required")
	}
	if strings.ContainsAny(message.To+message.Subject, "\r\n") {
		return errors.New("notification header contains a newline")
	}
	if len(message.Body) > 1<<20 {
		return errors.New("notification body is too large")
	}
	return nil
}

func formatMessage(from string, message Message) string {
	return "From: " + from + "\r\n" +
		"To: " + message.To + "\r\n" +
		"Subject: " + message.Subject + "\r\n" +
		"Date: " + time.Now().UTC().Format(time.RFC1123Z) + "\r\n" +
		"MIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n" +
		message.Body + "\r\n"
}
