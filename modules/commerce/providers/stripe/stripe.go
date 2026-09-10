package stripe

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/modules/commerce"
)

var ErrSignature = errors.New("invalid Stripe webhook signature")

type Provider struct {
	SecretKey     string
	WebhookSecret string
	BaseURL       string
	Client        *http.Client
	Tolerance     time.Duration
}

func (p Provider) Name() string { return "stripe" }

func (p Provider) CreatePaymentIntent(ctx context.Context, order commerce.Order) (commerce.PaymentIntent, error) {
	if p.SecretKey == "" {
		return commerce.PaymentIntent{}, errors.New("Stripe secret key is required")
	}
	form := url.Values{}
	form.Set("amount", strconv.FormatInt(order.TotalCents, 10))
	form.Set("currency", strings.ToLower(order.Currency))
	form.Set("metadata[order_id]", order.ID)
	var response struct {
		ID           string `json:"id"`
		Amount       int64  `json:"amount"`
		Currency     string `json:"currency"`
		Status       string `json:"status"`
		ClientSecret string `json:"client_secret"`
	}
	if err := p.request(ctx, http.MethodPost, "/v1/payment_intents", form, &response); err != nil {
		return commerce.PaymentIntent{}, err
	}
	return commerce.PaymentIntent{ID: response.ID, Provider: p.Name(), OrderID: order.ID, AmountCents: response.Amount, Currency: response.Currency, Status: response.Status, ClientToken: response.ClientSecret}, nil
}

func (p Provider) Refund(ctx context.Context, paymentID string, amountCents int64) error {
	form := url.Values{"payment_intent": []string{paymentID}, "amount": []string{strconv.FormatInt(amountCents, 10)}}
	var response struct {
		ID string `json:"id"`
	}
	return p.request(ctx, http.MethodPost, "/v1/refunds", form, &response)
}

func (p Provider) VerifyWebhookSignature(request *http.Request) (commerce.WebhookEvent, error) {
	if request == nil || p.WebhookSecret == "" {
		return commerce.WebhookEvent{}, ErrSignature
	}
	payload, err := io.ReadAll(io.LimitReader(request.Body, 1<<20))
	if err != nil {
		return commerce.WebhookEvent{}, ErrSignature
	}
	request.Body = io.NopCloser(bytes.NewReader(payload))
	if !verifyHeader(request.Header.Get("Stripe-Signature"), payload, p.WebhookSecret, p.Tolerance, time.Now()) {
		return commerce.WebhookEvent{}, ErrSignature
	}
	var envelope struct {
		ID      string `json:"id"`
		Type    string `json:"type"`
		Created int64  `json:"created"`
		Data    struct {
			Object struct {
				ID       string `json:"id"`
				Amount   int64  `json:"amount"`
				Currency string `json:"currency"`
				Status   string `json:"status"`
				Metadata struct {
					OrderID string `json:"order_id"`
				} `json:"metadata"`
			} `json:"object"`
		} `json:"data"`
	}
	if json.Unmarshal(payload, &envelope) != nil {
		return commerce.WebhookEvent{}, ErrSignature
	}
	status := "failed"
	if envelope.Type == "payment_intent.succeeded" {
		status = "paid"
	}
	if envelope.Type != "payment_intent.succeeded" && envelope.Type != "payment_intent.payment_failed" {
		return commerce.WebhookEvent{}, ErrSignature
	}
	return commerce.WebhookEvent{ID: envelope.ID, Provider: p.Name(), ProviderEventID: envelope.ID, OrderID: envelope.Data.Object.Metadata.OrderID, PaymentID: envelope.Data.Object.ID, Status: status, AmountCents: envelope.Data.Object.Amount, Currency: strings.ToUpper(envelope.Data.Object.Currency), OccurredAt: time.Unix(envelope.Created, 0).UTC()}, nil
}

func verifyHeader(header string, payload []byte, secret string, tolerance time.Duration, now time.Time) bool {
	if tolerance <= 0 {
		tolerance = 5 * time.Minute
	}
	var timestamp int64
	var valid bool
	for _, part := range strings.Split(header, ",") {
		key, value, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		switch key {
		case "t":
			timestamp, _ = strconv.ParseInt(value, 10, 64)
		case "v1":
			mac := hmac.New(sha256.New, []byte(secret))
			_, _ = mac.Write([]byte(strconv.FormatInt(timestamp, 10) + "."))
			_, _ = mac.Write(payload)
			expected := hex.EncodeToString(mac.Sum(nil))
			valid = valid || hmac.Equal([]byte(strings.ToLower(value)), []byte(expected))
		}
	}
	when := time.Unix(timestamp, 0)
	return timestamp > 0 && valid && now.Sub(when) <= tolerance && now.Sub(when) >= -tolerance
}

func (p Provider) request(ctx context.Context, method, path string, form url.Values, target any) error {
	base := p.BaseURL
	if base == "" {
		base = "https://api.stripe.com"
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(base, "/")+path, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(p.SecretKey, "")
	client := p.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Stripe API returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	if err := json.Unmarshal(body, target); err != nil {
		return err
	}
	return nil
}
