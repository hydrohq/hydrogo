package stripe

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
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
