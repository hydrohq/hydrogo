package mailer

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestSMTPPingChecksGreetingWithoutSendingEmail(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	commands := make(chan string, 4)
	go func() {
		connection, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}
		defer connection.Close()
		reader := bufio.NewReader(connection)
		writer := bufio.NewWriter(connection)
		_, _ = fmt.Fprint(writer, "220 mail.example.test ESMTP ready\r\n")
		_ = writer.Flush()
		for {
			line, readErr := reader.ReadString('\n')
			if readErr != nil {
				return
			}
			command := strings.TrimSpace(line)
			commands <- command
			switch strings.ToUpper(strings.SplitN(command, " ", 2)[0]) {
			case "EHLO", "HELO":
				_, _ = fmt.Fprint(writer, "250-mail.example.test\r\n250 OK\r\n")
			case "QUIT":
				_, _ = fmt.Fprint(writer, "221 bye\r\n")
				_ = writer.Flush()
				return
			default:
				_, _ = fmt.Fprint(writer, "500 unsupported\r\n")
			}
			_ = writer.Flush()
		}
	}()

	port := listener.Addr().(*net.TCPAddr).Port
	sender, err := NewSMTP(SMTPConfig{Host: "127.0.0.1", Port: port, From: "from@example.test", Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	result, err := sender.Ping(context.Background())
	if err != nil || result.Stage != "complete" || result.Elapsed >= time.Second {
		t.Fatalf("ping result=%+v err=%v", result, err)
	}
	for len(commands) > 0 {
		command := <-commands
		if strings.HasPrefix(strings.ToUpper(command), "MAIL ") || strings.HasPrefix(strings.ToUpper(command), "RCPT ") || strings.EqualFold(command, "DATA") {
			t.Fatalf("SMTP ping tried to send a message: %q", command)
		}
	}
}

func TestSMTPLoginAuthReturnsConfiguredCredentialsInOrder(t *testing.T) {
	auth := &smtpLoginAuth{username: "mailer", password: "secret"}
	mechanism, initial, err := auth.Start(nil)
	if err != nil || mechanism != "LOGIN" || len(initial) != 0 {
		t.Fatalf("AUTH LOGIN start = %q %q %v", mechanism, initial, err)
	}
	username, err := auth.Next([]byte("Username:"), true)
	if err != nil || string(username) != "mailer" {
		t.Fatalf("AUTH LOGIN username = %q %v", username, err)
	}
	password, err := auth.Next([]byte("Password:"), true)
	if err != nil || string(password) != "secret" {
		t.Fatalf("AUTH LOGIN password = %q %v", password, err)
	}
}

type flakySender struct{ calls int }

func (s *flakySender) Send(context.Context, Message) error {
	s.calls++
	if s.calls < 3 {
		return errors.New("temporary provider failure")
	}
	return nil
}

func TestRetrySenderRetriesTransientProviderFailure(t *testing.T) {
	underlying := &flakySender{}
	sender := RetrySender{Sender: underlying, Attempts: 3, BaseDelay: time.Millisecond}
	if err := sender.Send(context.Background(), Message{To: "to@example.test", Subject: "subject"}); err != nil {
		t.Fatal(err)
	}
	if underlying.calls != 3 {
		t.Fatalf("attempts = %d", underlying.calls)
	}
}

func TestRetrySenderHonorsPermanentFailureAndCancellation(t *testing.T) {
	underlying := &flakySender{}
	sender := RetrySender{Sender: underlying, Attempts: 5, BaseDelay: time.Millisecond, ShouldRetry: func(error) bool { return false }}
	if err := sender.Send(context.Background(), Message{To: "to@example.test", Subject: "subject"}); err == nil || underlying.calls != 1 {
		t.Fatalf("permanent failure = %v, calls = %d", err, underlying.calls)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := (RetrySender{}).Send(ctx, Message{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation = %v", err)
	}
}

func TestRetrySenderNormalizesNilContext(t *testing.T) {
	underlying := &flakySender{}
	sender := RetrySender{Sender: underlying, Attempts: 1}
	if err := sender.Send(nil, Message{}); err == nil {
		t.Fatal("nil-context retry sender unexpectedly succeeded")
	}
}

func TestRetrySenderCapsAttempts(t *testing.T) {
	var calls atomic.Int32
	underlying := &alwaysFailSender{calls: &calls}
	sender := RetrySender{Sender: underlying, Attempts: 1 << 30, BaseDelay: time.Nanosecond, ShouldRetry: func(error) bool { return true }}
	if err := sender.Send(context.Background(), Message{}); err == nil {
		t.Fatal("expected retry failure")
	}
	if got := calls.Load(); got != maxRetryAttempts {
		t.Fatalf("attempts = %d, want %d", got, maxRetryAttempts)
	}
}

type alwaysFailSender struct{ calls *atomic.Int32 }

func (s *alwaysFailSender) Send(context.Context, Message) error {
	s.calls.Add(1)
	return errors.New("temporary")
}

func TestNotificationSenderFailsClosedWithoutTransport(t *testing.T) {
	sender := NotificationSender{}
	if err := sender.SendPasswordReset(context.Background(), "user@example.test", "token"); !errors.Is(err, ErrSenderRequired) {
		t.Fatalf("password reset sender error = %v, want %v", err, ErrSenderRequired)
	}
	if err := sender.SendInvite(context.Background(), "user@example.test", "token"); !errors.Is(err, ErrSenderRequired) {
		t.Fatalf("invite sender error = %v, want %v", err, ErrSenderRequired)
	}
}

func TestNewSMTPValidatesConfiguration(t *testing.T) {
	if _, err := NewSMTP(SMTPConfig{Port: 587, From: "from@example.test"}); err == nil {
		t.Fatal("missing SMTP host accepted")
	}
	if _, err := NewSMTP(SMTPConfig{Host: "smtp.example.test", Port: 587, From: "from\r\nBcc: bad@example.test"}); err == nil {
		t.Fatal("header injection accepted")
	}
	if _, err := NewSMTP(SMTPConfig{Host: "smtp.example.test", Port: 587, From: "from@example.test", Password: "secret"}); err == nil {
		t.Fatal("password without username accepted")
	}
	if _, err := NewSMTP(SMTPConfig{Host: "smtp.example.test", Port: 25, From: "from@example.test", Username: "user", Password: "secret"}); err == nil {
		t.Fatal("SMTP AUTH without STARTTLS accepted")
	}
	if _, err := NewSMTP(SMTPConfig{Host: "smtp.example.test", Port: 587, From: "from@example.test", Username: "user", StartTLS: true}); err != nil {
		t.Fatal(err)
	}
}

func TestSMTPSendHonorsCanceledContextAndMessageLimits(t *testing.T) {
	sender, err := NewSMTP(SMTPConfig{Host: "127.0.0.1", Port: 1, From: "from@example.test"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := sender.Send(ctx, Message{To: "to@example.test", Subject: "subject"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled send error = %v", err)
	}
	if err := sender.Send(context.Background(), Message{To: "to\r\nBcc: bad@example.test", Subject: "subject"}); err == nil {
		t.Fatal("recipient header injection accepted")
	}
}

func TestSMTPSendNormalizesNilContext(t *testing.T) {
	sender, err := NewSMTP(SMTPConfig{Host: "127.0.0.1", Port: 1, From: "from@example.test"})
	if err != nil {
		t.Fatal(err)
	}
	if err := sender.Send(nil, Message{To: "to@example.test", Subject: "subject"}); err == nil {
		t.Fatal("nil-context SMTP send unexpectedly succeeded")
	}
}

func TestSMTPSendAppliesTimeoutBeforeGreeting(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	go func() {
		conn, err := listener.Accept()
		if err == nil {
			defer conn.Close()
			_, _ = io.Copy(io.Discard, conn)
		}
	}()

	sender, err := NewSMTP(SMTPConfig{Host: "127.0.0.1", Port: listener.Addr().(*net.TCPAddr).Port, From: "from@example.test", Timeout: 20 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	if err := sender.Send(context.Background(), Message{To: "to@example.test", Subject: "subject"}); err == nil {
		t.Fatal("SMTP send succeeded without a server greeting")
	} else if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("SMTP send exceeded timeout bound: %s", elapsed)
	}
}
