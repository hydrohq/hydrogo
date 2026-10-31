# HydroGo AI Agent Guide (Model Context Protocol)

HydroGo includes a native **Model Context Protocol (MCP)** server that allows AI agents—such as Cursor, Claude Code, and Claude Desktop—to introspect, provision, and mutate content and collections in real time without restarting the server.

---

## 1. Quick Setup (Under 30 Seconds)

### Step 1: Export MCP Config
Run the following command in your HydroGo project directory:

```bash
hydro mcp config
```

This prints a ready-to-use JSON configuration snippet:

```json
{
  "mcpServers": {
    "hydrogo": {
      "command": "/path/to/hydro",
      "args": ["mcp"]
    }
  }
}
```

### Step 2: Configure Your Agent

#### A. Cursor
Add the snippet to `.cursor/mcp.json` in your project root or in **Cursor Settings > Features > MCP**:
* **Name:** `hydrogo`
* **Type:** `command`
* **Command:** `hydro mcp`

#### B. Claude Desktop
Add the snippet to your `claude_desktop_config.json`:
* **macOS:** `~/Library/Application Support/Claude/claude_desktop_config.json`
* **Windows:** `%APPDATA%\Claude\claude_desktop_config.json`
* **Linux:** `~/.config/Claude/claude_desktop_config.json`

#### C. Claude Code (CLI)
When starting Claude Code, connect the MCP server:
```bash
claude mcp add hydrogo -- hydro mcp
```

---

## 2. Available MCP Tools

| Tool Name | Aliases | Description |
|:---|:---|:---|
| `list_content_types` | `schema_list` | Introspects all active collection schemas, field validations, and declarative access rules. |
| `define_content_type` | `schema_apply` | Dynamically creates or updates a collection schema at runtime without server restart. |
| `query_collection` | `content_query` | Queries records in any collection with filtering, sorting, pagination, and full-text search. |
| `mutate_record` | `content_mutate` | Creates, updates, or deletes records in dynamic collections with cryptographic audit logging. |

---

## 3. Example Agent Scenarios & Prompts

### Scenario 1: Provisioning a New Vertical Collection
Give your AI coding assistant (Cursor / Claude Code) this prompt:

> *"Connect to HydroGo MCP. Create a new collection schema for 'vehicles' with fields: brand (text), model (text), daily_rate (number), and available (boolean). Then insert 2 sample rental cars."*

**What happens behind the scenes:**
1. The agent calls `define_content_type`:
   ```json
   {
     "id": "vehicles",
     "name": "Vehicle",
     "plural": "Vehicles",
     "public": true,
     "fields": [
       {"id": "brand", "name": "Brand", "kind": "text", "required": true},
       {"id": "model", "name": "Model", "kind": "text", "required": true},
       {"id": "daily_rate", "name": "Daily Rate", "kind": "number", "required": true},
       {"id": "available", "name": "Available", "kind": "boolean", "required": false}
     ]
   }
   ```
2. HydroGo writes `content-types/vehicles.yaml`, runs atomic schema migration across SQLite/Postgres/MySQL, and registers `/api/v1/content/vehicles`.
3. The agent calls `mutate_record` with `operation: "create"` to seed the vehicles.
4. The collections and records immediately appear in the HydroGo Admin Panel at `/admin/content/vehicles`!

---

## 4. Security & Audit Trail

All tool calls executed by AI agents are authenticated and cryptographically recorded in the HydroGo audit log ledger (`hydro_audit_logs`) with SHA-256 tamper-evident chaining.
