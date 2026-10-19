import assert from "node:assert/strict";
import { HydroGoError, createHydroGoClient } from "../dist/hydrogo.js";

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
