package notify

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTelegramSendPostsToTheConfiguredBotAndChat(t *testing.T) {
	var gotPath string
	var gotBody map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	}))
	defer server.Close()

	telegram := Telegram{BotToken: "test-token", ChatID: "12345", BaseURL: server.URL}
	if err := telegram.Send(context.Background(), Message{Title: "Yeni sipariş", Body: "order_abc"}); err != nil {
		t.Fatal(err)
	}
	if gotPath != "/bottest-token/sendMessage" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotBody["chat_id"] != "12345" {
		t.Fatalf("chat_id = %q", gotBody["chat_id"])
	}
	if gotBody["text"] != "Yeni sipariş\norder_abc" {
		t.Fatalf("text = %q", gotBody["text"])
	}
}

func TestTelegramSendSurfacesATelegramAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "description": "chat not found"})
	}))
	defer server.Close()

	telegram := Telegram{BotToken: "test-token", ChatID: "wrong", BaseURL: server.URL}
	if err := telegram.Send(context.Background(), Message{Title: "x"}); err == nil {
		t.Fatal("expected an error for a rejected message")
	}
}

func TestTelegramSendRejectsAnUnconfiguredSender(t *testing.T) {
	var telegram Telegram
	if err := telegram.Send(context.Background(), Message{Title: "x"}); err == nil {
		t.Fatal("expected an error with no bot token/chat id configured")
	}
}
