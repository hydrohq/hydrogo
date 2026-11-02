package ai

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestEncryptAndDecryptBYOKAPIKey(t *testing.T) {
	secret := "platform-master-secret-key-2026"
	rawKey := "sk-live-tenant-custom-key-987654321"

	encrypted, err := EncryptAPIKey(rawKey, secret)
	if err != nil {
		t.Fatalf("EncryptAPIKey: %v", err)
	}
	if encrypted == rawKey {
		t.Fatal("expected encrypted key to differ from plaintext")
	}

	decrypted, err := DecryptAPIKey(encrypted, secret)
	if err != nil {
		t.Fatalf("DecryptAPIKey: %v", err)
	}
	if decrypted != rawKey {
		t.Fatalf("expected %q, got %q", rawKey, decrypted)
	}
}

func TestBYOKEncryptionRefusesMissingMasterSecret(t *testing.T) {
	if _, err := EncryptAPIKey("sk-live-tenant-key", ""); !errors.Is(err, ErrEncryptionNotConfigured) {
		t.Fatalf("expected ErrEncryptionNotConfigured, got %v", err)
	}
	if _, err := DecryptAPIKey("enc:v1:abc", "   "); !errors.Is(err, ErrEncryptionNotConfigured) {
		t.Fatalf("expected ErrEncryptionNotConfigured on decrypt, got %v", err)
	}
}

func TestAdapterEndpointsRejectNonLoopbackHTTP(t *testing.T) {
	ctx := context.Background()
	blocked := []string{
		"http://169.254.169.254/latest/meta-data",
		"http://internal-admin:8080/",
		"https://user:pass@evil.example/",
	}
	for _, endpoint := range blocked {
		for _, provider := range []string{"gemini", "anthropic", "openai"} {
			adapter := NewAdapter(AdapterConfig{Provider: provider, APIKey: "k", Endpoint: endpoint})
			if _, err := adapter.Generate(ctx, ChatRequest{Messages: []ChatMessage{{Role: "user", Content: "hi"}}}); err == nil {
				t.Fatalf("provider %s accepted unsafe endpoint %q", provider, endpoint)
			}
		}
	}
	// Loopback http is allowed by policy (local Ollama): it must reach the
	// network layer rather than being rejected by the security gate.
	_, err := NewAdapter(AdapterConfig{Provider: "gemini", APIKey: "k", Endpoint: "http://127.0.0.1:11434"}).Generate(ctx, ChatRequest{})
	if err != nil && (strings.Contains(err.Error(), "provider endpoint") || strings.Contains(err.Error(), "loopback")) {
		t.Fatalf("loopback endpoint rejected by security policy: %v", err)
	}
}

func TestSSRFControlAndEndpointHardening(t *testing.T) {
	ctx := context.Background()

	// 1. Static endpoint hardening: HTTPS literal private IPs and localhost are rejected
	blockedHTTPS := []string{
		"https://169.254.169.254/latest/meta-data",
		"https://10.0.0.1/admin",
		"https://172.16.0.1:8443/",
		"https://192.168.1.1/",
		"https://127.0.0.1:8443/",
		"https://localhost:8443/",
	}
	for _, endpoint := range blockedHTTPS {
		if _, err := endpointSecurityCheck(endpoint); err == nil {
			t.Fatalf("expected endpointSecurityCheck to reject %q", endpoint)
		}
	}

	// 2. Socket-level net.Dialer.Control blocks connection to private/loopback/link-local destinations
	client := NewSSRFProtectedClient(2*time.Second, false)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://127.0.0.1:9999", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Do(req)
	if err == nil {
		t.Fatal("expected socket dial to be rejected for 127.0.0.1")
	}
	if !strings.Contains(err.Error(), "restricted address") && !strings.Contains(err.Error(), "ssrf") {
		t.Fatalf("expected SSRF error, got: %v", err)
	}
}

func TestMultiProviderAdaptersAndCopilotPlanParser(t *testing.T) {
	ctx := context.Background()

	// 1. Mock Gemini Server
	geminiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"candidates": [{
				"content": {
					"parts": [{"text": "{\"reply\":\"İpek eşarp fiyatı güncellendi.\",\"operations\":[{\"op\":\"update\",\"collection_id\":\"products\",\"record_id\":\"p1\",\"data\":{\"price\":900}}]}"}]
				}
			}],
			"usageMetadata": {"totalTokenCount": 145}
		}`))
	}))
	defer geminiSrv.Close()

	adapter := NewAdapter(AdapterConfig{
		Provider: "gemini",
		Model:    "gemini-2.5-flash",
		APIKey:   "test-gemini-key",
		Endpoint: geminiSrv.URL,
		Client:   geminiSrv.Client(),
	})

	resp, err := adapter.Generate(ctx, ChatRequest{
		SystemPrompt: "System",
		Messages:     []ChatMessage{{Role: "user", Content: "Fiyatı 900 yap"}},
		JSONMode:     true,
	})
	if err != nil {
		t.Fatalf("GeminiAdapter.Generate: %v", err)
	}
	if resp.TokensUsed != 145 || resp.Provider != "gemini" {
		t.Fatalf("unexpected Gemini response metadata: %+v", resp)
	}

	plan, err := ParseCopilotPlan(resp.Content)
	if err != nil {
		t.Fatalf("ParseCopilotPlan: %v", err)
	}
	if len(plan.Operations) != 1 || plan.Operations[0].CollectionID != "products" || plan.Operations[0].RecordID != "p1" {
		t.Fatalf("unexpected parsed plan: %+v", plan)
	}

	// 2. Mock Anthropic Server
	anthropicSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "test-claude-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"content": [{"text": "{\"reply\":\"Tamamlandı\",\"operations\":[]}"}],
			"usage": {"input_tokens": 50, "output_tokens": 30}
		}`))
	}))
	defer anthropicSrv.Close()

	claude := NewAdapter(AdapterConfig{
		Provider: "anthropic",
		Model:    "claude-3-5-sonnet-latest",
		APIKey:   "test-claude-key",
		Endpoint: anthropicSrv.URL,
		Client:   anthropicSrv.Client(),
	})
	claudeResp, err := claude.Generate(ctx, ChatRequest{
		Messages: []ChatMessage{{Role: "user", Content: "Merhaba"}},
	})
	if err != nil || claudeResp.TokensUsed != 80 {
		t.Fatalf("AnthropicAdapter failed: resp=%+v err=%v", claudeResp, err)
	}
}
