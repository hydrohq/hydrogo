package ai

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/testhttp"
)

type providerFunc func(context.Context, Request) (Suggestion, error)

func (f providerFunc) Suggest(ctx context.Context, request Request) (Suggestion, error) {
	return f(ctx, request)
}

func TestAssistantRequiresEnablementConsentAndAllowedOperation(t *testing.T) {
	calls := 0
	provider := providerFunc(func(context.Context, Request) (Suggestion, error) {
		calls++
		return Suggestion{Body: "draft"}, nil
	})
	request := Request{Operation: Refine, Body: "Draft", Consent: true}
	if _, err := (Assistant{Provider: provider}).Suggest(context.Background(), request); !errors.Is(err, ErrDisabled) {
		t.Fatalf("disabled error = %v", err)
	}
	assistant := Assistant{Enabled: true, Provider: provider}
	request.Consent = false
	if _, err := assistant.Suggest(context.Background(), request); !errors.Is(err, ErrConsent) {
		t.Fatalf("consent error = %v", err)
	}
	request.Consent, request.Operation = true, Operation("arbitrary prompt")
	if _, err := assistant.Suggest(context.Background(), request); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("operation error = %v", err)
	}
	if calls != 0 {
		t.Fatalf("provider called %d times before validation", calls)
	}
}

func TestAssistantRedactsBoundsAndReturnsGenericProviderErrors(t *testing.T) {
	request := Request{Operation: Refine, Locale: "tr", Title: "mail owner@example.test", Body: "key sk-123456789012345678901234", Consent: true}
	provider := providerFunc(func(_ context.Context, got Request) (Suggestion, error) {
		if strings.Contains(got.Title+got.Body, "owner@example.test") || strings.Contains(got.Title+got.Body, "sk-123456789") {
			t.Fatal("sensitive data reached provider")
		}
		if !strings.Contains(got.Title, "[redacted-email]") || !strings.Contains(got.Body, "[redacted-secret]") {
			t.Fatalf("redaction missing: %+v", got)
		}
		return Suggestion{Title: "New title", Body: "New body"}, nil
	})
	result, err := (Assistant{Enabled: true, Provider: provider}).Suggest(context.Background(), request)
	if err != nil || result.Title != "New title" {
		t.Fatalf("suggestion=%+v err=%v", result, err)
	}
	failed := Assistant{Enabled: true, Provider: providerFunc(func(context.Context, Request) (Suggestion, error) {
		return Suggestion{}, errors.New("secret provider detail")
	})}
	if _, err := failed.Suggest(context.Background(), request); !errors.Is(err, ErrProvider) || strings.Contains(err.Error(), "secret provider") {
		t.Fatalf("provider error leaked detail: %v", err)
	}
}

func TestAssistantTranslationRequiresDistinctWellFormedTargetLocale(t *testing.T) {
	var received Request
	assistant := Assistant{Enabled: true, Provider: providerFunc(func(_ context.Context, request Request) (Suggestion, error) {
		received = request
		return Suggestion{Title: "Title", Body: "Body"}, nil
	})}
	request := Request{Operation: Translate, Locale: "tr-TR", TargetLocale: "en-US", Tone: "friendly", Body: "Metin", Consent: true}
	if _, err := assistant.Suggest(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if received.TargetLocale != "en-us" || received.Tone != "friendly" {
		t.Fatalf("provider request=%+v", received)
	}
	request.TargetLocale = "tr-tr"
	if _, err := assistant.Suggest(context.Background(), request); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("same-locale translation error=%v", err)
	}
	request.TargetLocale = "en/../../secret"
	if _, err := assistant.Suggest(context.Background(), request); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("malformed-locale translation error=%v", err)
	}
}

func TestOpenAICompatibleUsesRedactedRequestAndCapsResponse(t *testing.T) {
	const endpoint = "http://127.0.0.1:19081/v1"
	client := testhttp.Client(testhttp.Handlers{"127.0.0.1:19081": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" || r.Header.Get("Authorization") != "Bearer private-token" {
			t.Errorf("request path=%s auth=%q", r.URL.Path, r.Header.Get("Authorization"))
		}
		var body struct {
			Messages []chatMessage `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
			return
		}
		if strings.Contains(body.Messages[1].Content, "owner@example.test") {
			t.Error("unredacted email sent to provider")
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"title\":\"Öneri\",\"body\":\"Yeni taslak\"}"}}]}`))
	})})
	assistant := Assistant{Enabled: true, Provider: OpenAICompatible{Endpoint: endpoint, APIKey: "private-token", Model: "test", Client: client}, Timeout: time.Second}
	result, err := assistant.Suggest(context.Background(), Request{Operation: Refine, Locale: "tr", Title: "owner@example.test", Body: "Body", Consent: true})
	if err != nil || result.Title != "Öneri" || result.Body != "Yeni taslak" {
		t.Fatalf("suggestion=%+v err=%v", result, err)
	}
}

func TestHTTPProviderRejectsRemotePlaintextAndMalformedResponses(t *testing.T) {
	if _, err := (OpenAICompatible{Endpoint: "http://provider.example/v1", APIKey: "k", Model: "m"}).endpoint(); err == nil {
		t.Fatal("remote plaintext provider endpoint was accepted")
	}
	const endpoint = "http://127.0.0.1:19082"
	client := testhttp.Client(testhttp.Handlers{"127.0.0.1:19082": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"not-json"}}]}`))
	})})
	_, err := (Assistant{Enabled: true, Provider: OpenAICompatible{Endpoint: endpoint, APIKey: "k", Model: "m", Client: client}}).Suggest(context.Background(), Request{Operation: Summary, Body: "Body", Consent: true})
	if !errors.Is(err, ErrProvider) {
		t.Fatalf("malformed output error = %v", err)
	}
}
