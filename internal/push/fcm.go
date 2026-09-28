package push

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const fcmScope = "https://www.googleapis.com/auth/firebase.messaging"

type serviceAccount struct {
	ProjectID   string `json:"project_id"`
	ClientEmail string `json:"client_email"`
	PrivateKey  string `json:"private_key"`
	TokenURI    string `json:"token_uri"`
}
type FCMSender struct {
	account     serviceAccount
	key         *rsa.PrivateKey
	client      *http.Client
	Endpoint    string
	mu          sync.Mutex
	token       string
	tokenExpiry time.Time
}

func NewFCMSender(credentials []byte, client *http.Client) (*FCMSender, error) {
	var account serviceAccount
	if json.Unmarshal(credentials, &account) != nil || account.ProjectID == "" || account.ClientEmail == "" || account.TokenURI == "" {
		return nil, errors.New("invalid FCM service account")
	}
	tokenURL, err := url.Parse(account.TokenURI)
	if err != nil || tokenURL.Scheme != "https" || tokenURL.Host == "" {
		return nil, errors.New("invalid FCM token endpoint")
	}
	block, _ := pem.Decode([]byte(account.PrivateKey))
	if block == nil {
		return nil, errors.New("invalid FCM service account key")
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		var key *rsa.PrivateKey
		key, err = x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, errors.New("invalid FCM service account key")
		}
		parsed = key
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("invalid FCM service account key type")
	}
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &FCMSender{account: account, key: key, client: client}, nil
}
func (s *FCMSender) signedAssertion(now time.Time) (string, error) {
	encode := func(v any) string { raw, _ := json.Marshal(v); return base64.RawURLEncoding.EncodeToString(raw) }
	header := encode(map[string]string{"alg": "RS256", "typ": "JWT"})
	claims := encode(map[string]any{"iss": s.account.ClientEmail, "scope": fcmScope, "aud": s.account.TokenURI, "iat": now.Unix(), "exp": now.Add(time.Hour).Unix()})
	input := header + "." + claims
	digest := sha256.Sum256([]byte(input))
	signature, err := rsa.SignPKCS1v15(rand.Reader, s.key, crypto.SHA256, digest[:])
	if err != nil {
		return "", err
	}
	return input + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}
func (s *FCMSender) accessToken(ctx context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.token != "" && time.Until(s.tokenExpiry) > time.Minute {
		return s.token, nil
	}
	assertion, err := s.signedAssertion(time.Now())
	if err != nil {
		return "", err
	}
	form := url.Values{"grant_type": {"urn:ietf:params:oauth:grant-type:jwt-bearer"}, "assertion": {assertion}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.account.TokenURI, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 || json.NewDecoder(response.Body).Decode(&result) != nil || result.AccessToken == "" {
		return "", errors.New("FCM credentials could not be authorized")
	}
	s.token = result.AccessToken
	s.tokenExpiry = time.Now().Add(time.Duration(result.ExpiresIn) * time.Second)
	return s.token, nil
}
func (s *FCMSender) Send(ctx context.Context, sub Subscription, message Message) error {
	token, err := s.accessToken(ctx)
	if err != nil {
		return err
	}
	body, err := json.Marshal(map[string]any{"message": map[string]any{"token": sub.Endpoint, "notification": map[string]string{"title": message.Title, "body": message.Body}, "data": map[string]string{"url": message.URL}}})
	if err != nil {
		return err
	}
	endpoint := s.Endpoint
	if endpoint == "" {
		endpoint = "https://fcm.googleapis.com/v1"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(endpoint, "/")+"/projects/"+url.PathEscape(s.account.ProjectID)+"/messages:send", strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	response, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode == 404 || response.StatusCode == 410 {
		return ErrInvalidSubscription
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("FCM rejected notification (%d)", response.StatusCode)
	}
	return nil
}
