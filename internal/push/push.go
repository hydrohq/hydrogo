// Package push delivers bounded, opt-in notifications to registered Web Push and FCM devices.
package push

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"time"
)

var (
	ErrInvalidSubscription = errors.New("invalid push subscription")
	ErrInvalidMessage      = errors.New("invalid push message")
)

type Platform string

const (
	PlatformWeb    Platform = "web"
	PlatformMobile Platform = "mobile"
)

type Subscription struct {
	ID        string          `json:"id"`
	UserID    string          `json:"user_id"`
	Platform  Platform        `json:"platform"`
	Endpoint  string          `json:"endpoint"`
	Keys      json.RawMessage `json:"keys,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}

type Message struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	URL   string `json:"url,omitempty"`
}

func (m Message) Validate() error {
	if strings.TrimSpace(m.Title) == "" || len(m.Title) > 120 || strings.TrimSpace(m.Body) == "" || len(m.Body) > 1024 {
		return ErrInvalidMessage
	}
	if m.URL != "" {
		if len(m.URL) > 2048 {
			return ErrInvalidMessage
		}
		u, err := url.Parse(m.URL)
		if err != nil || u.IsAbs() && (u.Scheme != "https" && u.Scheme != "http") || strings.HasPrefix(m.URL, "//") {
			return ErrInvalidMessage
		}
	}
	return nil
}
func (s Subscription) Validate() error {
	if s.UserID == "" || len(s.UserID) > 128 {
		return ErrInvalidSubscription
	}
	switch s.Platform {
	case PlatformWeb:
		var web struct {
			Endpoint string `json:"endpoint"`
			Keys     struct {
				Auth   string `json:"auth"`
				P256dh string `json:"p256dh"`
			} `json:"keys"`
		}
		if json.Unmarshal(s.Keys, &web) != nil || web.Endpoint != s.Endpoint || len(s.Endpoint) > 4096 || len(web.Keys.Auth) < 8 || len(web.Keys.P256dh) < 20 {
			return ErrInvalidSubscription
		}
		u, err := url.Parse(s.Endpoint)
		if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || !safePushEndpointHost(u.Hostname()) {
			return ErrInvalidSubscription
		}
	case PlatformMobile:
		if s.Endpoint != strings.TrimSpace(s.Endpoint) || len(s.Endpoint) < 20 || len(s.Endpoint) > 4096 {
			return ErrInvalidSubscription
		}
	default:
		return ErrInvalidSubscription
	}
	return nil
}
func endpointID(platform Platform, endpoint string) string {
	sum := sha256.Sum256([]byte(string(platform) + "\x00" + endpoint))
	return hex.EncodeToString(sum[:])
}

type Repository interface {
	Save(context.Context, Subscription) error
	Delete(context.Context, string, string, Platform, string) error
	List(context.Context, string, Platform) ([]Subscription, error)
	Disable(context.Context, string) error
}
type Sender interface {
	Send(context.Context, Subscription, Message) error
}
type Dispatcher struct {
	Repository  Repository
	Web, Mobile Sender
}

func (d Dispatcher) Subscribe(ctx context.Context, s Subscription) error {
	if d.Repository == nil || s.Validate() != nil {
		return ErrInvalidSubscription
	}
	s.ID = endpointID(s.Platform, s.Endpoint)
	return d.Repository.Save(ctx, s)
}
func (d Dispatcher) Unsubscribe(ctx context.Context, userID string, p Platform, endpoint string) error {
	if d.Repository == nil || userID == "" || (p != PlatformWeb && p != PlatformMobile) {
		return ErrInvalidSubscription
	}
	return d.Repository.Delete(ctx, endpointID(p, endpoint), userID, p, endpoint)
}

type Result struct {
	Sent      int `json:"sent"`
	Delivered int `json:"delivered"`
	Failed    int `json:"failed"`
}

func (d Dispatcher) Broadcast(ctx context.Context, audience string, m Message) (Result, error) {
	if d.Repository == nil || m.Validate() != nil || (audience != "all" && audience != "web" && audience != "mobile") {
		return Result{}, ErrInvalidMessage
	}
	platforms := []Platform{PlatformWeb, PlatformMobile}
	if audience == "web" {
		platforms = []Platform{PlatformWeb}
	}
	if audience == "mobile" {
		platforms = []Platform{PlatformMobile}
	}
	var result Result
	for _, platform := range platforms {
		items, err := d.Repository.List(ctx, "", platform)
		if err != nil {
			return result, err
		}
		for _, sub := range items {
			result.Sent++
			sender := d.Web
			if platform == PlatformMobile {
				sender = d.Mobile
			}
			if sender == nil {
				result.Failed++
				continue
			}
			if err := sender.Send(ctx, sub, m); err != nil {
				result.Failed++
				if errors.Is(err, ErrInvalidSubscription) {
					_ = d.Repository.Disable(ctx, sub.ID)
				}
				continue
			}
			result.Delivered++
		}
	}
	return result, nil
}
func (d Dispatcher) PublicKey() string {
	if sender, ok := d.Web.(interface{ PublicKey() string }); ok {
		return sender.PublicKey()
	}
	return ""
}
