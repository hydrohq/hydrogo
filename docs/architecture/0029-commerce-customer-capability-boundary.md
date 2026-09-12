# ADR-0029: Commerce customer capability boundary

- **Status:** Accepted
- **Date:** 2026-09-12
- **Related decisions:** [ADR-0013](0013-identity-and-security-boundaries.md), [ADR-0017](0017-commerce-consistency-and-payments.md), [ADR-0025](0025-separate-frontend-members.md)

## Context

The initial commerce HTTP adapter treated an Admin Studio session as the only
identity allowed to create a payment intent. Public checkout could therefore
create an order but could not complete payment. In addition, it copied the raw
frontend member session cookie into the cart and order `user_id` field. That
turns a browser credential into persisted business data and prevents the
member's stable identifier from being used for order history.

## Decision

Commerce identifies an order owner through one of three existing capabilities:

1. an active Admin Studio identity, for administrative flows;
2. an active frontend member session resolved to its stable member ID at the
   application-composition boundary; or
3. the opaque, HttpOnly guest-cart cookie that created the order.

Payment-intent creation remains subject to the normal CSRF middleware and is
authorized by an exact order-owner comparison. It never creates a new guest
cookie during authorization. Raw member-session values are never stored in a
cart, order, audit event, or provider payload.

Public catalog reads use a dedicated published projection. Draft products are
not returned from public HTML or API routes, even if their opaque ID is known.
Provider webhook adapters must populate the common provider payment identifier
required by the idempotent payment-event contract.

## Consequences

This preserves guest checkout without conflating it with member authentication,
and lets members see future order-history queries using their stable IDs. The
change has no schema migration: old orders retain their historical owner value;
new writes use the corrected identity. It adds no worker, route, cache key, or
inactive-module overhead. Both database adapters remain behind the existing
repository contract. Existing browser cookies remain valid during a blue/green
deployment because the checkout capability format is unchanged.

## Validation

- A member session resolves to its member ID without setting a guest cookie.
- A guest owning an order can create a payment intent; another capability
  cannot.
- A draft product returns not found from public catalog routes.
- PayTR callback conversion supplies the common payment identifier.
