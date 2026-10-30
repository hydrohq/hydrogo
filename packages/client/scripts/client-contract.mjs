import assert from "node:assert/strict";
import { HydroGoError, createHydroGoClient, MemoryAuthStore, LocalStorageAuthStore } from "../dist/hydrogo.js";
import {
  createCollectionQuery,
  createCollectionRecordQuery,
  createCollectionMutation,
  autoSyncRealtimeCollection,
  pageQueryOptions,
  newsListQueryOptions,
  newsItemQueryOptions,
} from "../dist/query.js";

assert.throws(() => createHydroGoClient({ origin: "https://hydrogo.example", timeoutMs: 0 }), RangeError);
assert.throws(() => createHydroGoClient({ origin: "https://hydrogo.example", timeoutMs: 120001 }), RangeError);
assert.throws(() => createHydroGoClient({ origin: "javascript:alert(1)" }), RangeError);
assert.throws(() => createHydroGoClient({ origin: "https://user:pass@hydrogo.example" }), RangeError);

const timeoutClient = createHydroGoClient({
  origin: "https://hydrogo.example",
  timeoutMs: 10,
  fetch: async (_url, init) => await new Promise((_resolve, reject) => {
    init?.signal?.addEventListener("abort", () => reject(init.signal.reason), { once: true });
  }),
});
await assert.rejects(() => timeoutClient.listNews(), (error) => error?.name === "AbortError" || error?.name === "TimeoutError");

const externalAbort = new AbortController();
const externalClient = createHydroGoClient({
  origin: "https://hydrogo.example",
  timeoutMs: 120,
  fetch: async (_url, init) => await new Promise((_resolve, reject) => {
    init?.signal?.addEventListener("abort", () => reject(init.signal.reason), { once: true });
  }),
});
const externalRequest = externalClient.listNews("tr", 1, { signal: externalAbort.signal });
const cancelReason = new Error("caller-cancelled");
externalAbort.abort(cancelReason);
await assert.rejects(externalRequest, (error) => error === cancelReason);

let calls = 0;
const client = createHydroGoClient({
  origin: "https://hydrogo.example",
  fetch: async (url) => {
    calls++;
    assert.equal(url, "https://hydrogo.example/api/v1/news?locale=tr&limit=20");
    return new Response(JSON.stringify({ data: [{ id: "n1", locale: "tr", slug: "haber", title: "Haber", body: "", status: "published", revision: 1, published_revision: 1, updated_at: "2026-01-01T00:00:00Z" }], metadata: { limit: 20, has_more: false } }));
  },
});
const news = await client.listNews("tr");
assert.equal(news[0].id, "n1");
assert.equal(calls, 1);

const registrationClient = createHydroGoClient({
  origin: "https://hydrogo.example",
  fetch: async (url, init) => {
    assert.equal(url, "https://hydrogo.example/api/v1/mobile/member/auth/register");
    assert.equal(init?.credentials, "omit");
    assert.equal(new Headers(init?.headers).get("content-type"), "application/json");
    return new Response(JSON.stringify({ access_token: "access", refresh_token: "refresh", token_type: "Bearer", expires_in: 900, expires_at: "2026-01-01T00:00:00Z" }), { status: 200 });
  },
});
const registered = await registrationClient.memberRegister({ email: "reader@example.test", password: "secret1234", device_id: "ios-1", platform: "ios" });
assert.equal(registered.token_type, "Bearer");

const errorClient = createHydroGoClient({
  origin: "https://hydrogo.example",
  fetch: async () => new Response(JSON.stringify({ code: "denied", message: "Erişim reddedildi", request_id: "req-contract-1" }), { status: 403, headers: { "x-request-id": "req-contract-1" } }),
});
await assert.rejects(() => errorClient.listNews(), (error) => {
  assert.ok(error instanceof HydroGoError);
  assert.equal(error.code, "denied");
  assert.equal(error.requestId, "req-contract-1");
  assert.equal(error.message, "Erişim reddedildi");
  return true;
});

let requestedUrl = "";
const contentClient = createHydroGoClient({
  origin: "https://hydrogo.example/",
  fetch: async (url) => { requestedUrl = String(url); return new Response(JSON.stringify({ data: [{ id: "a" }], metadata: { limit: 5, has_more: false } })); },
});
const faq = await contentClient.listContent("faq", { limit: 5, search: "refund", filter: { category: "billing" } });
assert.equal(faq.data[0].id, "a");
assert.equal(requestedUrl, "https://hydrogo.example/api/v1/content/faq?limit=5&search=refund&filter%5Bcategory%5D=billing");
await assert.rejects(async () => contentClient.listContent("faq", { limit: 0 }), RangeError);

const pagedClient = createHydroGoClient({
  origin: "https://hydrogo.example",
  fetch: async (url) => { requestedUrl = String(url); return new Response(JSON.stringify({ data: [], metadata: { limit: 10, offset: 20, has_more: true } })); },
});
const page2 = await pagedClient.listNewsPage("en", 10, 20);
assert.equal(page2.metadata.has_more, true);
assert.equal(requestedUrl, "https://hydrogo.example/api/v1/news?locale=en&limit=10&offset=20");
assert.equal(pagedClient.mediaUrl("a b"), "https://hydrogo.example/media/a%20b");
await assert.rejects(async () => pagedClient.listNewsPage("en", 10, -1), RangeError);

const oversized = new Uint8Array(2 * 1024 * 1024 + 1);
const boundedClient = createHydroGoClient({ origin: "https://hydrogo.example", fetch: async () => new Response(oversized) });
await assert.rejects(() => boundedClient.listNews(), (error) => {
  assert.ok(error instanceof HydroGoError);
  assert.equal(error.code, "response_too_large");
  return true;
});

console.log("PASS: headless client contract smoke");

const listClient = createHydroGoClient({
  origin: "https://hydrogo.example",
  fetch: async (url) => {
    const query = new URL(url).searchParams;
    assert.equal(query.get("after"), "cursor-1");
    assert.equal(query.get("expand"), "company,partners");
    assert.equal(query.get("filter[status]"), "open");
    assert.equal(query.get("filter[amount][gte]"), "100");
    assert.equal(query.get("filter[amount][lt]"), "900");
    return new Response(JSON.stringify({ data: [], metadata: { limit: 10, has_more: true, next_cursor: "cursor-2" } }));
  },
});
const listed = await listClient.listContent("deals", { limit: 10, after: "cursor-1", expand: ["company", "partners"], filter: { status: "open", amount: { gte: "100", lt: "900" } } });
assert.equal(listed.metadata.next_cursor, "cursor-2");

// Test BaaS mutations and realtime
const crudClient = createHydroGoClient({
  origin: "https://hydrogo.example",
  fetch: async (url, init) => {
    const urlStr = String(url);
    if (init?.method === "POST" && urlStr.endsWith("/api/v1/content/messages")) {
      assert.equal(init?.headers?.["Authorization"], "Bearer test_token");
      const body = JSON.parse(String(init.body));
      assert.equal(body.text, "hello");
      return new Response(JSON.stringify({ data: { id: "msg_1", text: "hello" } }));
    }
    if (init?.method === "PUT" && urlStr.endsWith("/api/v1/content/messages/msg_1")) {
      assert.equal(init?.headers?.["Authorization"], "Bearer test_token");
      const body = JSON.parse(String(init.body));
      assert.equal(body.text, "updated");
      return new Response(JSON.stringify({ data: { id: "msg_1", text: "updated" } }));
    }
    if (init?.method === "DELETE" && urlStr.endsWith("/api/v1/content/messages/msg_1")) {
      return new Response(JSON.stringify({ data: { id: "msg_1", deleted: true } }));
    }
    throw new Error(`unexpected request: ${init?.method} ${urlStr}`);
  },
});

const created = await crudClient.createContent("messages", { text: "hello" }, "test_token");
assert.equal(created.id, "msg_1");
assert.equal(created.text, "hello");

const updated = await crudClient.updateContent("messages", "msg_1", { text: "updated" }, "test_token");
assert.equal(updated.text, "updated");

await crudClient.deleteContent("messages", "msg_1");

// Test realtime subscription parsing
let receivedEvent = null;
const streamClient = createHydroGoClient({
  origin: "https://hydrogo.example",
  fetch: async (url, init) => {
    assert.equal(String(url), "https://hydrogo.example/api/v1/content/messages/subscribe");
    assert.equal(init?.headers?.["Accept"], "text/event-stream");
    const stream = new ReadableStream({
      start(controller) {
        controller.enqueue(new TextEncoder().encode(": connected\n\ndata: {\"action\":\"create\",\"type\":\"messages\",\"record\":{\"id\":\"m1\",\"text\":\"realtime message\"}}\n\n"));
        controller.close();
      },
    });
    return new Response(stream, { headers: { "Content-Type": "text/event-stream" } });
  },
});

const unsubscribe = streamClient.subscribeContent("messages", (event) => {
  receivedEvent = event;
});

await new Promise((resolve) => setTimeout(resolve, 50));
assert.ok(receivedEvent);
assert.equal(receivedEvent.action, "create");
assert.equal(receivedEvent.record.text, "realtime message");
unsubscribe();

// Test realtime subscription with reconnect, backoff, and Last-Event-ID
let connectionStates = [];
let connectionAttempts = 0;
let receivedLastEventIdHeader = null;
const reconnectEvents = [];

const reconnectClient = createHydroGoClient({
  origin: "https://hydrogo.example",
  fetch: async (url, init) => {
    connectionAttempts++;
    if (connectionAttempts === 1) {
      // First connection emits event 1 and then terminates
      const stream = new ReadableStream({
        start(controller) {
          controller.enqueue(new TextEncoder().encode(": connected\n\nid: evt-101\ndata: {\"action\":\"create\",\"type\":\"messages\",\"record\":{\"id\":\"m1\",\"text\":\"first msg\"}}\n\n"));
          controller.close();
        },
      });
      return new Response(stream, { headers: { "Content-Type": "text/event-stream" } });
    } else {
      // Second connection receives Last-Event-ID and sends event 2
      receivedLastEventIdHeader = init?.headers?.["Last-Event-ID"];
      let streamCtrl;
      init?.signal?.addEventListener("abort", () => {
        try { streamCtrl?.close(); } catch {}
      });
      const stream = new ReadableStream({
        start(controller) {
          streamCtrl = controller;
          controller.enqueue(new TextEncoder().encode(": connected\n\nid: evt-102\ndata: {\"action\":\"create\",\"type\":\"messages\",\"record\":{\"id\":\"m2\",\"text\":\"replayed msg\"}}\n\n"));
        },
      });
      return new Response(stream, { headers: { "Content-Type": "text/event-stream" } });
    }
  },
});

const unsubReconnect = reconnectClient.subscribeContent("messages", (event) => {
  reconnectEvents.push(event);
}, {
  minBackoffMs: 20,
  maxBackoffMs: 50,
  onStateChange: (state) => {
    connectionStates.push(state);
  },
});

await new Promise((resolve) => setTimeout(resolve, 300));
unsubReconnect();

assert.equal(connectionAttempts, 2, "Expected 2 connection attempts (initial + reconnect)");
assert.equal(receivedLastEventIdHeader, "evt-101", "Expected Last-Event-ID header to carry last seen event id");
assert.equal(reconnectEvents.length, 2, "Expected 2 events across connections");
assert.equal(reconnectEvents[0].record.id, "m1");
assert.equal(reconnectEvents[1].record.id, "m2");
assert.ok(connectionStates.includes("connecting"));
assert.ok(connectionStates.includes("connected"));
assert.ok(connectionStates.includes("reconnecting"));
assert.ok(connectionStates.includes("disconnected"));

// -------------------------------------------------------------
// Issue #397: Stateful AuthStore & Fluent Collection API Tests
// -------------------------------------------------------------

// 1. MemoryAuthStore unit test
const store = new MemoryAuthStore();
assert.equal(store.isValid, false);
assert.equal(store.token, "");
assert.equal(store.model, null);

let changeCount = 0;
let lastChangeToken = "";
const unsubChange = store.onChange((token, model) => {
  changeCount++;
  lastChangeToken = token;
});

const sampleMember = {
  id: "m-123",
  email: "test@example.com",
  active: true,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};

store.save("sample-token", sampleMember);
assert.equal(store.isValid, true);
assert.equal(store.token, "sample-token");
assert.equal(store.model?.email, "test@example.com");
assert.equal(changeCount, 1);
assert.equal(lastChangeToken, "sample-token");

store.clear();
assert.equal(store.isValid, false);
assert.equal(store.token, "");
assert.equal(store.model, null);
assert.equal(changeCount, 2);

unsubChange();
store.save("another-token", sampleMember);
assert.equal(changeCount, 2, "Unsubscribed callback should not receive notifications");

// 2. Automatic Token Injection with custom AuthStore
const authStore = new MemoryAuthStore("auto-injected-token", sampleMember);
let inspectedHeaders = null;

const fluentClient = createHydroGoClient({
  origin: "https://hydrogo.example",
  authStore,
  fetch: async (url, init) => {
    inspectedHeaders = init?.headers;
    if (url.endsWith("/api/v1/content/messages") && init?.method === "POST") {
      return new Response(JSON.stringify({ data: { id: "msg-1", text: "Hello" } }));
    }
    if (url.endsWith("/api/v1/content/messages/msg-1") && init?.method === "PUT") {
      return new Response(JSON.stringify({ data: { id: "msg-1", text: "Updated" } }));
    }
    if (url.endsWith("/api/v1/content/messages/msg-1") && init?.method === "DELETE") {
      return new Response(JSON.stringify({ data: { id: "msg-1" } }));
    }
    if (url.endsWith("/api/v1/content/messages/msg-1")) {
      return new Response(JSON.stringify({ data: { id: "msg-1", text: "Hello" } }));
    }
    if (url.includes("/api/v1/content/messages") && (!init?.method || init.method === "GET")) {
      return new Response(JSON.stringify({ data: [{ id: "msg-1", text: "Hello" }], metadata: { limit: 20, has_more: false } }));
    }
    return new Response(JSON.stringify({}), { status: 404 });
  },
});

// Fluent collection tests
const messages = fluentClient.collection("messages");
assert.equal(messages.collectionName, "messages");

// Create with automatic token injection
const fluentCreated = await messages.create({ text: "Hello" });
assert.equal(fluentCreated.id, "msg-1");
assert.equal(inspectedHeaders["Authorization"], "Bearer auto-injected-token");

// Update with automatic token injection
const fluentUpdated = await messages.update("msg-1", { text: "Updated" });
assert.equal(fluentUpdated.text, "Updated");
assert.equal(inspectedHeaders["Authorization"], "Bearer auto-injected-token");

// Delete with automatic token injection
await messages.delete("msg-1");
assert.equal(inspectedHeaders["Authorization"], "Bearer auto-injected-token");

// Explicit token overrides authStore
await messages.create({ text: "Override" }, "explicit-override-token");
assert.equal(inspectedHeaders["Authorization"], "Bearer explicit-override-token");

// List and Get
const list = await messages.list();
assert.equal(list.data.length, 1);
const item = await messages.get("msg-1");
assert.equal(item.id, "msg-1");

// 3. auth.loginMember and auth.logout integration
const authClient = createHydroGoClient({
  origin: "https://hydrogo.example",
  fetch: async (url, init) => {
    if (url.endsWith("/api/v1/mobile/member/auth/login")) {
      return new Response(JSON.stringify({
        access_token: "login-access-token",
        refresh_token: "login-refresh-token",
        token_type: "Bearer",
        expires_in: 900,
        expires_at: "2026-01-01T00:00:00Z",
      }));
    }
    if (url.endsWith("/api/v1/mobile/member/me")) {
      return new Response(JSON.stringify({
        id: "m-logged-in",
        email: "login@example.test",
        active: true,
        created_at: "2026-01-01T00:00:00Z",
        updated_at: "2026-01-01T00:00:00Z",
      }));
    }
    if (url.endsWith("/api/v1/mobile/member/auth/revoke")) {
      return new Response(null, { status: 204 });
    }
    return new Response(null, { status: 404 });
  },
});

assert.equal(authClient.auth.isValid, false);
const loginResult = await authClient.auth.loginMember({
  email: "login@example.test",
  password: "password123",
  device_id: "test-device",
  platform: "node",
});

assert.equal(loginResult.tokens.access_token, "login-access-token");
assert.equal(loginResult.profile.id, "m-logged-in");
assert.equal(authClient.auth.isValid, true);
assert.equal(authClient.auth.token, "login-access-token");
assert.equal(authClient.auth.model?.email, "login@example.test");

await authClient.auth.logout();
assert.equal(authClient.auth.isValid, false);
assert.equal(authClient.auth.token, "");
assert.equal(authClient.auth.model, null);

// -------------------------------------------------------------
// Issue #409: TanStack Query Hooks & Auto-Invalidation Tests
// -------------------------------------------------------------

// 1. Query options generation
const queryClientMock = {
  invalidated: [],
  invalidateQueries(filters) {
    this.invalidated.push(filters.queryKey);
  },
};

const qOpts = createCollectionQuery(fluentClient, "messages", { limit: 10 });
assert.deepEqual(qOpts.queryKey, ["hydrogo", "content", "messages", "list", { limit: 10 }]);
const qRes = await qOpts.queryFn();
assert.equal(qRes.data.length, 1);

const rOpts = createCollectionRecordQuery(fluentClient, "messages", "msg-1");
assert.deepEqual(rOpts.queryKey, ["hydrogo", "content", "messages", "record", "msg-1", ""]);
const rRes = await rOpts.queryFn();
assert.equal(rRes.id, "msg-1");

// 2. Mutation options generation & cache invalidation
const mutationOpts = createCollectionMutation(fluentClient, "messages", queryClientMock);

const mutatedCreate = await mutationOpts.mutationFn({
  action: "create",
  data: { text: "New from mutation" },
});
assert.equal(mutatedCreate.id, "msg-1");
await mutationOpts.onSuccess();
assert.equal(queryClientMock.invalidated.length, 1);
assert.deepEqual(queryClientMock.invalidated[0], ["hydrogo", "content", "messages"]);

// 3. Realtime auto-invalidation
let sseStreamCtrl;
const sseClient = createHydroGoClient({
  origin: "https://hydrogo.example",
  fetch: async (url) => {
    if (url.includes("/api/v1/content/messages/subscribe")) {
      const stream = new ReadableStream({
        start(controller) {
          sseStreamCtrl = controller;
          controller.enqueue(
            new TextEncoder().encode(
              ": connected\n\nid: evt-sync-1\ndata: {\"action\":\"update\",\"type\":\"messages\",\"record\":{\"id\":\"msg-42\",\"text\":\"Live update\"}}\n\n"
            )
          );
        },
      });
      return new Response(stream, { headers: { "Content-Type": "text/event-stream" } });
    }
    return new Response(null, { status: 404 });
  },
});

const realtimeQueryMock = {
  invalidated: [],
  invalidateQueries(filters) {
    this.invalidated.push(filters.queryKey);
  },
};

const unsubAutoSync = autoSyncRealtimeCollection(sseClient, "messages", realtimeQueryMock);

// Wait for event to arrive
await new Promise((resolve) => setTimeout(resolve, 50));
unsubAutoSync();
try { sseStreamCtrl?.close(); } catch {}

assert.ok(
  realtimeQueryMock.invalidated.some((key) => key[0] === "hydrogo" && key[2] === "messages"),
  "Expected collection query key to be invalidated on SSE event"
);
assert.ok(
  realtimeQueryMock.invalidated.some((key) => key[0] === "hydrogo" && key[2] === "messages" && key[3] === "record" && key[4] === "msg-42"),
  "Expected specific record query key to be invalidated on SSE event"
);




