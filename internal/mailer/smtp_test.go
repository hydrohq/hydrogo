package mailer

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"time"
)

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
	if _, err := NewSMTP(SMTPConfig{Host: "smtp.example.test", Port: 587, From: "from@example.test", Username: "user"}); err != nil {
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
