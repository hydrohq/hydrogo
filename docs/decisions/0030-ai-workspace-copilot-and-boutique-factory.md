# 0030. Multi-Adapter AI Workspace Copilot and Digital Boutique Factory

Date: 2026-10-31
Status: Accepted

## Context

Store owners and workspace administrators need to manage their catalog, pricing, and storefront visual themes using natural language instructions ("AI Copilot"), while the platform operator needs the flexibility to connect each tenant workspace to different AI providers (Gemini, OpenAI, Anthropic Claude, or OpenAI-compatible local/cloud endpoints like Ollama, Groq, and DeepSeek) with optional tenant-owned API keys (BYOK) and strict monthly token quotas.

At the same time, HydroGo's core must remain a clean, domain-neutral headless CMS without customer-specific or vertical-specific Go bloat.

## Decision

1. **Multi-Adapter AI Engine (`internal/ai/adapters.go`)**:
   - Introduced a provider-neutral `Adapter` interface (`Generate(ctx, ChatRequest) (ChatResponse, error)`) with native implementations for Google Gemini (`GeminiAdapter`), Anthropic Claude (`AnthropicAdapter`), and OpenAI / OpenAI-compatible endpoints (`OpenAIAdapter`).
   - Added AES-256-GCM encryption (`EncryptAPIKey` / `DecryptAPIKey`) so tenant BYOK keys stored in `tenant_governance.ai_encrypted_api_key` are encrypted at rest and never returned in API responses.
2. **Scoped Workspace Copilot with Snapshot Undo (`internal/ai/copilot.go`, `cmd/hydro/admin_api_copilot.go`)**:
   - Added `POST /api/v1/admin/ai/copilot`, `GET /api/v1/admin/ai/actions`, and `POST /api/v1/admin/ai/actions/{id}/undo`.
   - Every Copilot execution runs strictly inside the authenticated tenant's `tenant.Scope`, checks `tenant_governance.ai_enabled` and `ai_monthly_token_limit`, captures pre-mutation `BeforeData` snapshots on every record update/delete, and supports one-click reversal (`Undo`) from the admin `CopilotDrawer.svelte` UI.
3. **Zero-Core-Bloat Boutique Blueprint (`content-types/boutique/`, `examples/boutique-starter/`, `tools/factory/`)**:
   - Boutique store schemas (`store_settings`, `boutique_categories`, `boutique_products`, `boutique_orders`) are defined purely as declarative YAML in `content-types/boutique/`.
   - The mobile-first luxury storefront lives in `examples/boutique-starter/` with live CSS theme token resolution (`src/lib/theme.ts`) and WhatsApp quick-order deep linking (`src/lib/whatsapp.ts`).
   - `tools/factory` provisions a tenant, applies governance presets, syncs the boutique YAML schemas, and seeds the theme and catalog in a single command.
