package stripe

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/modules/commerce"
)

func TestVerifyHeader(t *testing.T) {
	secret, payload, timestamp := "whsec_test", []byte(`{"id":"evt_1"}`), time.Now().Unix()
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strconv.FormatInt(timestamp, 10) + "."))
	mac.Write(payload)
	header := "t=" + strconv.FormatInt(timestamp, 10) + ",v1=" + hex.EncodeToString(mac.Sum(nil))
	if !verifyHeader(header, payload, secret, time.Minute, time.Now()) {
		t.Fatal("valid signature rejected")
	}
	if verifyHeader(strings.Replace(header, "v1=", "v1=00", 1), payload, secret, time.Minute, time.Now()) {
		t.Fatal("invalid signature accepted")
	}
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(payload)))
	req.Header.Set("Stripe-Signature", header)
	if _, err := (Provider{WebhookSecret: secret}).VerifyWebhookSignature(req); err == nil {
		t.Fatal("incomplete event accepted")
	}
}

func TestCreatePaymentIntentWithKeySendsProviderIdempotencyHeader(t *testing.T) {
	var gotKey string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("Idempotency-Key")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "pi_1", "amount": 1250, "currency": "try", "status": "requires_payment_method", "client_secret": "secret"})
	}))
	defer server.Close()
	order := commerce.Order{ID: "order-1", Currency: "TRY", TotalCents: 1250}
	intent, err := (Provider{SecretKey: "sk_test", BaseURL: server.URL}).CreatePaymentIntentWithKey(context.Background(), order, "payment-intent-key-01")
	if err != nil || intent.ID != "pi_1" || gotKey != "payment-intent-key-01" {
		t.Fatalf("intent=%+v key=%q err=%v", intent, gotKey, err)
	}
}
