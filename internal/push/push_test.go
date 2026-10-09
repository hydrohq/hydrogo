package push

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"io"
	"net/http"
	"net/netip"
	"strings"
	"testing"
)

type memorySubscriptions struct {
	items    []Subscription
	disabled []string
}

func (m *memorySubscriptions) Save(_ context.Context, s Subscription) error {
	m.items = append(m.items, s)
	return nil
}
func (m *memorySubscriptions) Delete(_ context.Context, id, user string, p Platform, endpoint string) error {
	return nil
}
func (m *memorySubscriptions) List(_ context.Context, user string, p Platform) ([]Subscription, error) {
	out := []Subscription{}
	for _, s := range m.items {
		if s.Platform == p && (user == "" || s.UserID == user) {
			out = append(out, s)
		}
	}
	return out, nil
}
func (m *memorySubscriptions) Disable(_ context.Context, id string) error {
	m.disabled = append(m.disabled, id)
	return nil
}

type fakeSender struct {
	err  error
	sent int
}

func (s *fakeSender) Send(context.Context, Subscription, Message) error { s.sent++; return s.err }

func TestSubscriptionValidationAndBroadcastPrunesExpired(t *testing.T) {
	repo := &memorySubscriptions{}
	web := &fakeSender{err: ErrInvalidSubscription}
	mobile := &fakeSender{}
	d := Dispatcher{Repository: repo, Web: web, Mobile: mobile}
	repo.items = append(repo.items, Subscription{ID: "dead-web", UserID: "u1", Platform: PlatformWeb, Endpoint: "https://push.example.test/1"})
	if err := d.Subscribe(context.Background(), Subscription{UserID: "u1", Platform: PlatformMobile, Endpoint: strings.Repeat("a", 24)}); err != nil {
		t.Fatal(err)
	}
	if err := d.Subscribe(context.Background(), Subscription{UserID: "u1", Platform: PlatformWeb, Endpoint: "http://push.example.test/1", Keys: json.RawMessage(`{"endpoint":"http://push.example.test/1","keys":{"auth":"abcdefgh","p256dh":"abcdefghijklmnopqrstuvwx"}}`)}); err == nil {
		t.Fatal("accepted insecure web push URL")
	}
	result, err := d.Broadcast(context.Background(), "all", Message{Title: "Update", Body: "A short notice", URL: "/news"})
	if err != nil || result.Sent != 2 || result.Delivered != 1 || result.Failed != 1 || len(repo.disabled) != 1 || repo.disabled[0] != "dead-web" {
		t.Fatalf("result=%+v disabled=%v err=%v", result, repo.disabled, err)
	}
}

func TestFCMSenderUsesServiceAccountAndV1Endpoint(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	pemKey := string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		var body string
		switch r.URL.Path {
		case "/token":
			if r.Method != http.MethodPost {
				t.Errorf("token method=%s", r.Method)
			}
			body = `{"access_token":"access","expires_in":3600}`
		case "/v1/projects/demo-project/messages:send":
			if r.Header.Get("Authorization") != "Bearer access" {
				t.Errorf("auth header missing")
			}
			requestBody, err := io.ReadAll(r.Body)
			if err != nil || !strings.Contains(string(requestBody), `"title":"Hello"`) {
				t.Errorf("message body=%s err=%v", requestBody, err)
			}
			body = `{"name":"projects/demo-project/messages/abc"}`
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			return &http.Response{StatusCode: http.StatusNotFound, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("not found")), Request: r}, nil
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})}
	credentials, _ := json.Marshal(map[string]string{"project_id": "demo-project", "client_email": "push@example.test", "private_key": pemKey, "token_uri": "https://push.example.test/token"})
	sender, err := NewFCMSender(credentials, client)
	if err == nil {
		sender.Endpoint = "https://push.example.test/v1"
	}
	if err != nil {
		t.Fatal(err)
	}
	if err := sender.Send(context.Background(), Subscription{Endpoint: strings.Repeat("f", 24)}, Message{Title: "Hello", Body: "World"}); err != nil {
		t.Fatal(err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestWebPushEndpointBlocksPrivateAndLocalDestinations(t *testing.T) {
	for _, endpoint := range []string{"https://127.0.0.1/sub", "https://10.0.0.7/sub", "https://localhost/sub", "https://printer.local/sub", "https://[::1]/sub"} {
		if (Subscription{UserID: "u", Platform: PlatformWeb, Endpoint: endpoint, Keys: json.RawMessage(`{"endpoint":"` + endpoint + `","keys":{"auth":"abcdefgh1234","p256dh":"abcdefghijklmnopqrstuvwx123456"}}`)}).Validate() == nil {
			t.Fatalf("accepted private endpoint %q", endpoint)
		}
	}
	if !publicPushIP(netip.MustParseAddr("8.8.8.8")) {
		t.Fatal("public DNS address classified as private")
	}
	if publicPushIP(netip.MustParseAddr("169.254.169.254")) {
		t.Fatal("link-local metadata address classified as public")
	}
}
