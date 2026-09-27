package forms

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

func TestTurnstileVerifierUsesFixedEndpointAndBoundedFormPost(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.String() != turnstileVerifyURL || request.Method != http.MethodPost {
			t.Fatalf("unexpected verification request %s %s", request.Method, request.URL)
		}
		body, _ := io.ReadAll(request.Body)
		if !strings.Contains(string(body), "secret=secret-key") || !strings.Contains(string(body), "response=proof") {
			t.Fatalf("verification payload = %s", body)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"success":true}`)), Header: make(http.Header)}, nil
	})}
	valid, err := (TurnstileVerifier{Secret: "secret-key", Client: client}).Verify(context.Background(), "proof", "192.0.2.1")
	if err != nil || !valid {
		t.Fatalf("verify = %v/%v", valid, err)
	}
}

func TestTurnstileVerifierRejectsOversizedAndMalformedResponses(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(strings.Repeat("x", 4097))), Header: make(http.Header)}, nil
	})}
	if _, err := (TurnstileVerifier{Secret: "secret", Client: client}).Verify(context.Background(), "proof", ""); err == nil {
		t.Fatal("oversized verification response was accepted")
	}
	if _, err := (TurnstileVerifier{Secret: "secret"}).Verify(context.Background(), strings.Repeat("x", 2049), ""); err == nil {
		t.Fatal("oversized token was accepted")
	}
}
