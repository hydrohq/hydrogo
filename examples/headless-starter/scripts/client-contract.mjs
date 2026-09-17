import assert from "node:assert/strict";
import { HydroGoError, createHydroGoClient, validateIdempotencyKey } from "../dist/hydrogo.js";

assert.throws(() => validateIdempotencyKey("too-short"), RangeError);
assert.equal(validateIdempotencyKey("checkout-key-0001"), "checkout-key-0001");
assert.throws(() => createHydroGoClient({ origin: "https://hydrogo.example", timeoutMs: 0 }), RangeError);
assert.throws(() => createHydroGoClient({ origin: "https://hydrogo.example", timeoutMs: 120001 }), RangeError);

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
  signal: externalAbort.signal,
  fetch: async (_url, init) => await new Promise((_resolve, reject) => {
    init?.signal?.addEventListener("abort", () => reject(init.signal.reason), { once: true });
  }),
});
const externalRequest = externalClient.listNews("tr", 1);
const cancelReason = new Error("caller-cancelled");
externalAbort.abort(cancelReason);
await assert.rejects(externalRequest, (error) => error === cancelReason);

let calls = 0;
const client = createHydroGoClient({
  origin: "https://hydrogo.example",
  fetch: async (url) => {
    calls++;
    assert.equal(url, "https://hydrogo.example/api/v1/news?locale=tr&limit=20");
    return new Response(JSON.stringify([{ id: "n1", locale: "tr", slug: "haber", title: "Haber", body: "", status: "published", revision: 1, published_revision: 1, updated_at: "2026-01-01T00:00:00Z" }]));
  },
});
const news = await client.listNews("tr");
assert.equal(news[0].id, "n1");
assert.equal(calls, 1);

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

const oversized = new Uint8Array(2 * 1024 * 1024 + 1);
const boundedClient = createHydroGoClient({ origin: "https://hydrogo.example", fetch: async () => new Response(oversized) });
await assert.rejects(() => boundedClient.listNews(), (error) => {
  assert.ok(error instanceof HydroGoError);
  assert.equal(error.code, "response_too_large");
  return true;
});

console.log("PASS: headless client contract smoke");
