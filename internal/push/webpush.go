package push

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"

	webpush "github.com/SherClockHolmes/webpush-go"
)

type WebPushSender struct {
	Public, Private, Subject string
	Client                   webpush.HTTPClient
}

func (s WebPushSender) PublicKey() string { return s.Public }
func (s WebPushSender) Send(ctx context.Context, sub Subscription, message Message) error {
	if s.Public == "" || s.Private == "" || s.Subject == "" {
		return errors.New("web push is not configured")
	}
	subject, err := url.Parse(s.Subject)
	if err != nil || (subject.Scheme != "mailto" && subject.Scheme != "https") || subject.Scheme == "https" && subject.Host == "" {
		return errors.New("invalid VAPID subject")
	}
	var subscription webpush.Subscription
	if err := json.Unmarshal(sub.Keys, &subscription); err != nil {
		return ErrInvalidSubscription
	}
	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}
	client := s.Client
	if client == nil {
		client = webPushHTTPClient()
	}
	response, err := webpush.SendNotificationWithContext(ctx, payload, &subscription, &webpush.Options{Subscriber: s.Subject, VAPIDPublicKey: s.Public, VAPIDPrivateKey: s.Private, TTL: 3600, HTTPClient: client})
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode == 404 || response.StatusCode == 410 {
		return ErrInvalidSubscription
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return errors.New("push service rejected notification")
	}
	return nil
}
