package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Telegram sends a message through a Telegram bot's sendMessage API --
// one HTTP POST, no SDK needed. Create the bot with @BotFather, add it to
// the chat/channel/group you want alerts in, and ChatID is that chat's id
// (a group chat id is negative; @BotFather's own chat with you works too
// for a single-operator setup).
type Telegram struct {
	BotToken string
	ChatID   string
	// BaseURL overrides https://api.telegram.org for tests.
	BaseURL    string
	HTTPClient *http.Client
}

func (t Telegram) baseURL() string {
	if t.BaseURL != "" {
		return t.BaseURL
	}
	return "https://api.telegram.org"
}

func (t Telegram) httpClient() *http.Client {
	if t.HTTPClient != nil {
		return t.HTTPClient
	}
	return defaultHTTPClient
}

// defaultHTTPClient bounds every call; http.DefaultClient has no timeout.
var defaultHTTPClient = &http.Client{Timeout: 10 * time.Second}

// Enabled reports whether enough configuration is present to send -- cheap
// to check from wiring code before bothering to construct a Telegram at
// all.
func (t Telegram) Enabled() bool { return t.BotToken != "" && t.ChatID != "" }

func (t Telegram) Send(ctx context.Context, msg Message) error {
	if !t.Enabled() {
		return errors.New("notify: telegram bot_token and chat_id are required")
	}
	body, err := json.Marshal(map[string]string{"chat_id": t.ChatID, "text": msg.Text()})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.baseURL()+"/bot"+t.BotToken+"/sendMessage", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.httpClient().Do(req)
	if err != nil {
		// *url.Error embeds the request URL, which carries the bot token.
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			err = urlErr.Err
		}
		return fmt.Errorf("notify: telegram request failed: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("notify: read telegram response: %w", err)
	}
	var parsed struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return fmt.Errorf("notify: decode telegram response: %w", err)
	}
	if !parsed.OK {
		return fmt.Errorf("notify: telegram rejected the message: %s", parsed.Description)
	}
	return nil
}
