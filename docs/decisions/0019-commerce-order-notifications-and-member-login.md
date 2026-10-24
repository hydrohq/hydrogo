# 0019: Order notifications and storefront member login

- Status: Accepted
- Date: 2026-10-23

## Context

[0012](0012-commerce-catalog-as-content-types.md) through
[0018](0018-commerce-reconciliation.md) built a complete commerce path, but
three real gaps remained: a shopper who just paid gets no confirmation that
anything happened; an operator finds out about a new order (or a
reconciliation anomaly) only by reading logs or querying the API by hand;
and `examples/commerce-starter` had no concept of a logged-in customer at
all — `modules/members` already provides session-cookie authentication,
nothing in the storefront used it.

## Decision

**Order notifications.** `internal/notify` is a small, generic sender
abstraction (`Sender.Send(ctx, Message)`), kept deliberately outside
`modules/commerce` so a Telegram bot today and Slack (or others) later
share one interface without commerce depending on a specific channel.
`internal/notify.Telegram` is the first implementation. Two options
considered and rejected before this:

- **Outbox-based async delivery**: `internal/outbox/sqlstore.Store.Claim`
  has no topic filter — any consumer claims the next available event
  regardless of topic, and `internal/webhooks.Worker` is already the only
  consumer. A second consumer (an "email/notify worker") would race it for
  the same rows. Not built.
- **Reusing `internal/mailer/sqlstore.Spool`**: its `Enqueue` takes a
  `modules/site.EmailNotification` and imports `modules/site` directly.
  Any caller constructing that type — including `modules/commerce` —
  would need to import `modules/site`, which `moduleguard` forbids
  (commerce imports zero `modules/*` packages today). Not reused.

Instead, `transport.Handler` gained two optional, best-effort fields —
`Mailer OrderMailer` (a narrow interface satisfied as-is by
`mailer.NotificationSender`, defined in commerce's own transport package
rather than `internal/mailer`, so `internal/` does not need to know about
commerce types) and `Notifier notify.Sender` — and calls both
synchronously right after `Checkout` succeeds, the same precedent
`PaymentProvider.StartPayment` already set for "a post-checkout step that
must never fail the checkout itself." `runCommerceReconciliationAlert`
also pushes a notification on a non-clean report, alongside its existing
`slog` warning. New config: `notifications.telegram.{bot_token,chat_id}`,
both-or-neither validated the same way `PayTR.Enabled()` is cheap to
precheck from `cmd/hydro`.

**Member login for the storefront.** `examples/commerce-starter` gained
`/uye/giris` (login), `/uye/kayit` (register) and `/uye/hesap` (account +
order history), calling `modules/members`' existing `/member/*` session
API. `src/lib/server/apiProxy.ts` factors out the cookie-proxying
mechanics `commerce.ts` already had (HydroGo's own `hydro_commerce_cart`/
`hydro_csrf`/`hydro_member_session` cookies forwarded verbatim, no
privileged secret to protect) so `member.ts` can reuse it; a separate
`memberAuthRequest` helper exists because `/member/*` responses are not
HydroGo's `{data, metadata}` envelope. Checkout now prefills the e-mail
field for a logged-in member.

## Consequences

- `go run ./tools/check` passes; `examples/commerce-starter`'s
  `npm run check`/`build`/`smoke` pass.
- Verified against a real `hydro serve` instance, not mocked: register →
  member session cookie set → cart added as that member → checkout →
  order correctly carries `member_id` (not a guest order) → `/member/orders`
  lists it in the shape `member.ts` expects → logout clears the session
  and `/member/me` then returns 401.
- Real gaps found and left named, not hidden: this repo's GitHub Actions
  is disabled, so "passes CI" only ever means the local `go run
  ./tools/check` gate, not independent CI validation. PayTR has still
  never been exercised against real or sandbox credentials. There is still
  no admin UI screen for commerce orders at all.
