# 0031. AI Generation Engine, Prompt Registry, and Section-Based Storefront

Date: 2026-11-02
Status: Accepted

## Context

HydroGo Cloud (the HydroGo Engine hosting offer) sells "AI builds your site, you
keep full control, backend included" — the Bolt.new comparison only holds if the
generation is actually AI-driven. Today it is not:

- Storefront HTML is compiled into the binary (`internal/storefront/storefront.html`
  via `go:embed`); customers can only change a handful of `store_settings` scalars.
- Catalog and editorial content come from a keyword-matched hardcoded generator
  (`internal/blueprints.GenerateLuxuryCatalog`, 3 niches x 6 products) with
  best-effort AI polish bolted on afterwards.
- Every system prompt is a Go string literal spread across `internal/ai/copilot.go`,
  `internal/ai/openai_compat.go`, and `cmd/hydro/superadmin_consultant.go`.
- The AI surface is abusable: no rate limits on AI endpoints, token quota only on
  the copilot path, hardcoded BYOK master-secret fallbacks, copilot operations
  accepted without field-level schema validation, and a raw-JSON prompt path that
  parses the prompt itself as an executable plan when no LLM key is set.

Selling hosting + site generation requires both halves fixed: generation must be
driven by our instructions (prompts) only, and the AI surface must be safe against
tenant and platform abuse.

## Decision

1. **Prompt Registry as the single source of instructions**
   (`internal/ai/prompts/`): every system prompt and every generation instruction
   set lives in embedded `.prompt` / `.json` files with `{id, version, description}`
   headers, overridable at runtime via `HYDRO_PROMPTS_DIR`. No system prompt
   remains a Go string. Every AI response records `prompt_id`, `prompt_version`
   and `prompt_hash` for audit.

2. **SitePlan pipeline replaces the hardcoded catalog**
   (`internal/sitegen/`): site generation produces a validated JSON `SitePlan`
   (theme tokens, ordered layout sections, collections, records) that is executed
   as copilot-style operations inside the tenant scope — snapshot, undo, quota and
   audit all apply. The deterministic fallback reads instruction files
   (`internal/blueprints/instructions/*.json`) instead of Go literals; AI is
   optional, never a hard dependency.

3. **Section-based storefront** (`internal/storefront/sections/`): storefront
   sections become template partials with a registry (id + JSON prop schema).
   `store_settings.layout_json` stores the ordered `[{type, props}]` layout; the
   renderer falls back to the editorial default when absent and skips unknown
   section types, so a bad plan can never break a live store. Layout is editable
   zero-rebuild through the existing content API.

4. **Abuse controls on every AI path**: rate limiters on
   `/api/v1/admin/ai/copilot`, `/api/v1/admin/pages/{id}/ai-suggest` and
   `/api/v1/superadmin/chat`; pre/post `tenant.ConsumeAITokens` enforcement with
   surfaced 429s; platform `default_monthly_token_limit` applied at provisioning
   and clamped onto tenant limits; monthly `ai_tokens_used` reset job; BYOK
   encryption refuses to operate without an explicit `HYDRO_AI_KEY_SECRET`
   (hardcoded fallback secrets removed); adapter endpoints restricted to
   https/loopback (SSRF); copilot operation `Data` keys validated against the
   collection schema; the raw-JSON plan path restricted to dev mode.

## Consequences

- Public API surface grows: `POST /api/v1/admin/ai/generate`, extended
  `/api/v1/superadmin/provision` response, `store_settings.layout_json` field.
- Prompt edits become an operational (deploy/config) change instead of a code
  change; version bumps show up in audit records.
- New tenant stores stay backward compatible: absent `layout_json` and absent AI
  keys render the same storefront as before, seeded from instruction files.
- Storefront template work moves from one 1500-line file to per-section partials;
  section changes now need schema + fallback tests instead of full-page tests.
