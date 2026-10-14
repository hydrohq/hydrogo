# ADR-0141: AI is opt-in and provider-neutral

- **Status:** Accepted (contract); no default provider or data egress
- **Implementation status:** Opt-in page-draft refinement, text-focused SEO suggestions, summarization, tone adjustment, translation with configured locale choices, and one-request form submission summaries are available through an OpenAI-compatible HTTP adapter behind the `ai` module and explicit environment enablement. Dedicated SEO metadata fields, production provider/legal approval, and prompt-injection review remain release gates.

## Decision

- AI capabilities are disabled by default and must be enabled per module, environment, and data classification policy.
- Core depends on a provider-neutral interface with bounded input/output tokens, timeout, retry budget, and audit metadata; provider SDKs stay in adapters.
- Customer content is not sent to an external provider unless explicit consent, region, retention, and redaction policy allow it.
- Prompts, model responses, and secrets are not persisted or logged by default; safety and human-review states are part of the workflow.

## Release gate

Prompt-injection/data-exfiltration review, provider outage fallback, cost/latency budgets, redaction tests, and a documented retention/DPA decision are required before enabling AI in a release.

The page editor assistant requires `HYDRO_AI_ENABLED=true`, an active `ai`
module, and operator-supplied `HYDRO_AI_ENDPOINT`, `HYDRO_AI_API_KEY`, and
`HYDRO_AI_MODEL`. It supports OpenAI-compatible chat-completions endpoints;
plain HTTP is restricted to loopback for local self-hosted providers. Editors
must also consent for every request. Email addresses and common API-key shapes
are redacted before egress; prompts and output are not persisted. A minimal
request audit event records the page reference and actor, never content. The
editor must explicitly apply the escaped suggestion to its in-memory draft and
save it as a separate action. Operator DPA, provider retention, model quality,
and prompt-injection review remain release gates.
