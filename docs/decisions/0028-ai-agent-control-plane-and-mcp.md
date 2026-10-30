# 0028: AI-Agent Control Plane & Model Context Protocol (MCP) Architecture

- Status: Accepted
- Date: 2026-10-30

## Context

HydroGo is evolving from a traditional headless CMS into an AI-native backend engine and autonomous agent control plane. Modern AI coding assistants (Claude Code, Cursor, Cline, OpenCode) and autonomous agents frequently need to:
1. Programmatically inspect schemas, relation graphs, and collection constraints.
2. Formulate and apply schema evolutions (adding collections, modifying field validations, configuring access rules) without human intervention.
3. Query and mutate data safely without fragile shell commands or server reboots.

Previously, modifying schemas required editing YAML files on disk and rebooting the server process. Additionally, operations produced human-oriented text logs rather than machine-parseable JSON envelopes, making agent error recovery error-prone.

## Decision

Adopt an AI-first control plane architecture with standardized agent interfaces and safety boundaries:

1. **Native Model Context Protocol (MCP) Endpoint (`/api/v1/mcp`)**:
   - Provide an embedded JSON-RPC / Server-Sent Events (SSE) MCP server endpoint in HydroGo core.
   - Expose first-class MCP tools:
     - `schema_list`: Retrieves all active collection definitions, validation rules, and relational schemas.
     - `schema_apply`: Atomically provisions or updates collection schemas with validation and dry-run preview.
     - `content_query`: Executes structured queries, filters, and aggregations.
     - `content_mutate`: Performs atomic creations, updates, and deletes with rollback protection.
   - Expose MCP resources for OpenAPI specifications (`hydro://openapi.json`) and active project configuration.

2. **Zero-Downtime Dynamic Schema Runtime**:
   - Administrative schema endpoints (`POST /api/v1/admin/content-types`, `PUT /api/v1/admin/content-types/{id}`, `DELETE /api/v1/admin/content-types/{id}`) update disk YAML definitions, SQLite/PostgreSQL/MySQL metadata tables (`hydro_collections`), and in-memory route registries in a single atomic transaction.
   - Public collection endpoints (`/api/v1/content/<new_id>`) become immediately accessible without restarting the HTTP server.

3. **Deterministic Machine-Readable CLI Operations**:
   - Standardized `--json` and `--dry-run` across all schema commands (`hydro content-type list|validate|apply|diff --json`).
   - Emit deterministic error envelopes specifying exact file, line, field, and syntax failure details for automated self-correction by agents.

4. **Security, Sandboxing & Audit Governance**:
   - **Identity & Authorization**: AI agents authenticate via standard Bearer tokens or dedicated API keys with granular permissions (`content.write`, `schema.manage`).
   - **Audit Attribution**: All agent-originated schema evolutions and data mutations are cryptographically signed in the tamper-evident audit log with explicit actor attribution (`actor_type: "ai_agent"`).
   - **Transactional Rollbacks**: Schema migrations and batch mutations operate atomically; validation failures or rule violations trigger an immediate database rollback with zero orphaned state.
   - **Tenant Isolation**: Multi-tenant boundaries are strictly enforced across all MCP tool calls and schema modifications.

## Consequences

- AI agents can autonomously scaffold, extend, and maintain complete backend architectures on top of HydroGo with verified safety guarantees.
- Zero server restarts required during ongoing schema development and AI code generation.
- Full compatibility with the open Model Context Protocol ecosystem (Claude Desktop, Cursor, and custom agent harnesses).
- Eliminates manual drift between disk-based YAML schemas, database tables, and runtime OpenAPI specifications.
