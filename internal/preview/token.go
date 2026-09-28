package preview

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

var ErrInvalidToken = errors.New("invalid preview token")

type Claims struct {
	PageID   string `json:"page_id"`
	TenantID string `json:"tenant_id,omitempty"`
	Revision int    `json:"revision"`
	Expires  int64  `json:"expires"`
}
type Signer struct{ Key []byte }

func (s Signer) Issue(claims Claims, now time.Time, ttl time.Duration) (string, error) {
	if len(s.Key) < 32 || claims.PageID == "" || len(claims.PageID) > 128 || claims.Revision < 1 || ttl <= 0 || ttl > 7*24*time.Hour {
		return "", ErrInvalidToken
	}
	claims.Expires = now.Add(ttl).Unix()
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	body := base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, s.Key)
	_, _ = mac.Write([]byte(body))
	return body + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}
func (s Signer) Verify(token string, now time.Time) (Claims, error) {
	if len(s.Key) < 32 || len(token) > 2048 {
		return Claims{}, ErrInvalidToken
	}
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return Claims{}, ErrInvalidToken
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Claims{}, ErrInvalidToken
	}
	mac := hmac.New(sha256.New, s.Key)
	_, _ = mac.Write([]byte(parts[0]))
	if !hmac.Equal(sig, mac.Sum(nil)) {
		return Claims{}, ErrInvalidToken
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return Claims{}, ErrInvalidToken
	}
	var claims Claims
	if json.Unmarshal(payload, &claims) != nil || claims.PageID == "" || claims.Revision < 1 || claims.Expires <= now.Unix() {
		return Claims{}, ErrInvalidToken
	}
	return claims, nil
}
