package contenttypes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/realtime"
	"github.com/ardaozyildirim/hydrogo/internal/router"
)

const chatMessagesYAML = `id: messages
label: Messages
title_field: text
public: true
realtime: true
access_rules:
  list: "public"
  read: "public"
  create: "@auth.id != '' && @auth.id == record.sender_id"
  update: "@auth.id == record.sender_id"
  delete: "@auth.id == record.sender_id"
fields:
  - {id: text, label: Text, kind: text, required: true}
  - {id: sender_id, label: Sender, kind: text, required: true}
`

type testPrincipalKey struct{}

func withTestPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, testPrincipalKey{}, p)
}

func testPrincipalResolver(r *http.Request) Principal {
	if p, ok := r.Context().Value(testPrincipalKey{}).(Principal); ok {
		return p
	}
	return Principal{Anonymous: true}
}

func baasPublicMux(api PublicAPI) http.Handler {
	routes := []router.Route{
		{Pattern: "GET /api/v1/content/{type}", Access: router.Public, Handler: http.HandlerFunc(api.List)},
		{Pattern: "GET /api/v1/content/{type}/{id}", Access: router.Public, Handler: http.HandlerFunc(api.Get)},
		{Pattern: "POST /api/v1/content/{type}", Access: router.Public, Handler: http.HandlerFunc(api.Create)},
		{Pattern: "PUT /api/v1/content/{type}/{id}", Access: router.Public, Handler: http.HandlerFunc(api.Update)},
		{Pattern: "DELETE /api/v1/content/{type}/{id}", Access: router.Public, Handler: http.HandlerFunc(api.Delete)},
		{Pattern: "GET /api/v1/content/{type}/subscribe", Access: router.Public, Handler: http.HandlerFunc(api.Subscribe)},
	}
	handler, err := router.New(routes, func(*http.Request) bool { return true })
	if err != nil {
		panic(err)
	}
	return handler
}

func TestPublicCRUDWithAccessRules(t *testing.T) {
	service, types := newServiceFor(t, chatMessagesYAML)
	hub := &realtime.Hub{}
	api := PublicAPI{
		Service:          service,
		Types:            types,
		Hub:              hub,
		ResolvePrincipal: testPrincipalResolver,
	}
	handler := baasPublicMux(api)

	user1 := Principal{ID: "usr_alice", Email: "alice@example.com", Role: "member", IsMember: true}
	user2 := Principal{ID: "usr_bob", Email: "bob@example.com", Role: "member", IsMember: true}

	// 1. Anonymous create should fail with 401
	recAnon := httptest.NewRecorder()
	reqAnon := httptest.NewRequest(http.MethodPost, "/api/v1/content/messages", strings.NewReader(`{"text":"hello","sender_id":"usr_alice"}`))
	handler.ServeHTTP(recAnon, reqAnon)
	if recAnon.Code != http.StatusUnauthorized {
		t.Fatalf("anon create code = %d, want 401", recAnon.Code)
	}

	// 2. User2 creating as Alice should fail with 403 (@auth.id != sender_id)
	recSpoof := httptest.NewRecorder()
	reqSpoof := httptest.NewRequest(http.MethodPost, "/api/v1/content/messages", strings.NewReader(`{"text":"hello","sender_id":"usr_alice"}`))
	reqSpoof = reqSpoof.WithContext(withTestPrincipal(reqSpoof.Context(), user2))
	handler.ServeHTTP(recSpoof, reqSpoof)
	if recSpoof.Code != http.StatusForbidden {
		t.Fatalf("spoof create code = %d, want 403", recSpoof.Code)
	}

	// 3. User1 creating as Alice should succeed with 201
	recCreate := httptest.NewRecorder()
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/v1/content/messages", strings.NewReader(`{"text":"hello from alice","sender_id":"usr_alice"}`))
	reqCreate = reqCreate.WithContext(withTestPrincipal(reqCreate.Context(), user1))
	handler.ServeHTTP(recCreate, reqCreate)
	if recCreate.Code != http.StatusCreated {
		t.Fatalf("valid create code = %d (%s), want 201", recCreate.Code, recCreate.Body.String())
	}

	var createResp map[string]any
	if err := json.Unmarshal(recCreate.Body.Bytes(), &createResp); err != nil {
		t.Fatal(err)
	}
	recData := data(t, createResp)
	msgID, _ := recData["id"].(string)
	if msgID == "" {
		t.Fatal("missing record id in response")
	}

	// 4. User2 attempting to update Alice's message should fail with 403
	recBadUpdate := httptest.NewRecorder()
	reqBadUpdate := httptest.NewRequest(http.MethodPut, "/api/v1/content/messages/"+msgID, strings.NewReader(`{"text":"tampered","sender_id":"usr_alice"}`))
	reqBadUpdate = reqBadUpdate.WithContext(withTestPrincipal(reqBadUpdate.Context(), user2))
	handler.ServeHTTP(recBadUpdate, reqBadUpdate)
	if recBadUpdate.Code != http.StatusForbidden {
		t.Fatalf("unauthorized update code = %d, want 403", recBadUpdate.Code)
	}

	// 5. User1 updating their own message should succeed with 200
	recUpdate := httptest.NewRecorder()
	reqUpdate := httptest.NewRequest(http.MethodPut, "/api/v1/content/messages/"+msgID, strings.NewReader(`{"text":"edited by alice","sender_id":"usr_alice"}`))
	reqUpdate = reqUpdate.WithContext(withTestPrincipal(reqUpdate.Context(), user1))
	handler.ServeHTTP(recUpdate, reqUpdate)
	if recUpdate.Code != http.StatusOK {
		t.Fatalf("valid update code = %d, want 200", recUpdate.Code)
	}

	// 6. User2 attempting to delete Alice's message should fail with 403
	recBadDel := httptest.NewRecorder()
	reqBadDel := httptest.NewRequest(http.MethodDelete, "/api/v1/content/messages/"+msgID, nil)
	reqBadDel = reqBadDel.WithContext(withTestPrincipal(reqBadDel.Context(), user2))
	handler.ServeHTTP(recBadDel, reqBadDel)
	if recBadDel.Code != http.StatusForbidden {
		t.Fatalf("unauthorized delete code = %d, want 403", recBadDel.Code)
	}

	// 7. User1 deleting their own message should succeed with 200
	recDel := httptest.NewRecorder()
	reqDel := httptest.NewRequest(http.MethodDelete, "/api/v1/content/messages/"+msgID, nil)
	reqDel = reqDel.WithContext(withTestPrincipal(reqDel.Context(), user1))
	handler.ServeHTTP(recDel, reqDel)
	if recDel.Code != http.StatusOK {
		t.Fatalf("valid delete code = %d, want 200", recDel.Code)
	}
}

func TestPublicRealtimeSubscription(t *testing.T) {
	service, types := newServiceFor(t, chatMessagesYAML)
	hub := &realtime.Hub{}
	api := PublicAPI{
		Service:          service,
		Types:            types,
		Hub:              hub,
		ResolvePrincipal: testPrincipalResolver,
	}
	handler := baasPublicMux(api)

	alice := Principal{ID: "usr_alice", Email: "alice@example.com", Role: "member", IsMember: true}

	// Start SSE subscription in a goroutine
	streamCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reqSub := httptest.NewRequest(http.MethodGet, "/api/v1/content/messages/subscribe", nil).WithContext(streamCtx)
	recSub := httptest.NewRecorder()

	subDone := make(chan struct{})
	go func() {
		handler.ServeHTTP(recSub, reqSub)
		close(subDone)
	}()

	// Wait until subscriber is registered in hub
	deadline := time.Now().Add(2 * time.Second)
	for {
		if hub.SubscriberCount() >= 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("subscriber did not register with hub in time")
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Now Alice creates a message
	recCreate := httptest.NewRecorder()
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/v1/content/messages", strings.NewReader(`{"text":"hello realtime","sender_id":"usr_alice"}`))
	reqCreate = reqCreate.WithContext(withTestPrincipal(reqCreate.Context(), alice))
	handler.ServeHTTP(recCreate, reqCreate)
	if recCreate.Code != http.StatusCreated {
		t.Fatalf("create failed with %d: %s", recCreate.Code, recCreate.Body.String())
	}

	// Allow event to flush to SSE stream
	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case <-subDone:
	case <-time.After(time.Second):
		t.Fatal("SSE handler did not exit after cancel")
	}

	sseOutput := recSub.Body.String()
	if !strings.Contains(sseOutput, ": connected") {
		t.Fatalf("expected SSE to contain ': connected', got %q", sseOutput)
	}
	if !strings.Contains(sseOutput, "hello realtime") {
		t.Fatalf("expected SSE to contain 'hello realtime', got %q", sseOutput)
	}
	if !strings.Contains(sseOutput, `"action":"create"`) {
		t.Fatalf("expected SSE to contain action 'create', got %q", sseOutput)
	}
}

func TestPublicRealtimeSubscriptionRowLevelFiltering(t *testing.T) {
	privateMessagesYAML := `id: dms
label: Direct Messages
title_field: text
public: true
realtime: true
access_rules:
  list: "@auth.id == record.sender_id"
  read: "@auth.id == record.sender_id"
  create: "@auth.id != '' && @auth.id == record.sender_id"
fields:
  - {id: text, label: Text, kind: text, required: true}
  - {id: sender_id, label: Sender, kind: text, required: true}
`
	service, types := newServiceFor(t, privateMessagesYAML)
	hub := &realtime.Hub{}
	api := PublicAPI{
		Service:          service,
		Types:            types,
		Hub:              hub,
		ResolvePrincipal: testPrincipalResolver,
	}
	handler := baasPublicMux(api)

	alice := Principal{ID: "usr_alice", Email: "alice@example.com", Role: "member", IsMember: true}
	bob := Principal{ID: "usr_bob", Email: "bob@example.com", Role: "member", IsMember: true}

	// Bob subscribes
	streamCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reqSub := httptest.NewRequest(http.MethodGet, "/api/v1/content/dms/subscribe", nil).WithContext(streamCtx)
	reqSub = reqSub.WithContext(withTestPrincipal(reqSub.Context(), bob))
	recSub := httptest.NewRecorder()

	subDone := make(chan struct{})
	go func() {
		handler.ServeHTTP(recSub, reqSub)
		close(subDone)
	}()

	for {
		if hub.SubscriberCount() >= 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Alice creates a message (Bob should NOT receive it due to read rule @auth.id == record.sender_id)
	recAlice := httptest.NewRecorder()
	reqAlice := httptest.NewRequest(http.MethodPost, "/api/v1/content/dms", strings.NewReader(`{"text":"alice secret","sender_id":"usr_alice"}`))
	reqAlice = reqAlice.WithContext(withTestPrincipal(reqAlice.Context(), alice))
	handler.ServeHTTP(recAlice, reqAlice)
	if recAlice.Code != http.StatusCreated {
		t.Fatalf("alice create failed: %d", recAlice.Code)
	}

	// Bob creates a message (Bob SHOULD receive it)
	recBob := httptest.NewRecorder()
	reqBob := httptest.NewRequest(http.MethodPost, "/api/v1/content/dms", strings.NewReader(`{"text":"bob public message","sender_id":"usr_bob"}`))
	reqBob = reqBob.WithContext(withTestPrincipal(reqBob.Context(), bob))
	handler.ServeHTTP(recBob, reqBob)
	if recBob.Code != http.StatusCreated {
		t.Fatalf("bob create failed: %d", recBob.Code)
	}

	time.Sleep(50 * time.Millisecond)
	cancel()
	<-subDone

	sseOutput := recSub.Body.String()
	if strings.Contains(sseOutput, "alice secret") {
		t.Fatalf("security violation: subscriber bob received alice's private message: %s", sseOutput)
	}
	if !strings.Contains(sseOutput, "bob public message") {
		t.Fatalf("subscriber bob should have received his own message: %s", sseOutput)
	}
}
