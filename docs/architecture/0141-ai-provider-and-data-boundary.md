# ADR-0141: AI is opt-in and provider-neutral

- **Status:** Accepted (contract); no default provider or data egress

## Decision

- AI capabilities are disabled by default and must be enabled per module, environment, and data classification policy.
- Core depends on a provider-neutral interface with bounded input/output tokens, timeout, retry budget, and audit metadata; provider SDKs stay in adapters.
- Customer content is not sent to an external provider unless explicit consent, region, retention, and redaction policy allow it.
- Prompts, model responses, and secrets are not persisted or logged by default; safety and human-review states are part of the workflow.

## Release gate

Prompt-injection/data-exfiltration review, provider outage fallback, cost/latency budgets, redaction tests, and a documented retention/DPA decision are required before enabling AI in a release.
