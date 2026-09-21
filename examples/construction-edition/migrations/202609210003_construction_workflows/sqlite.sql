CREATE TABLE construction_organizations (id TEXT PRIMARY KEY, parent_id TEXT NOT NULL DEFAULT '', name TEXT NOT NULL);
CREATE TABLE construction_projects (id TEXT PRIMARY KEY, organization_id TEXT NOT NULL, name TEXT NOT NULL, created_at TEXT NOT NULL);
CREATE TABLE construction_memberships (project_id TEXT NOT NULL, user_id TEXT NOT NULL, role TEXT NOT NULL, PRIMARY KEY(project_id,user_id));
CREATE TABLE construction_documents (id TEXT PRIMARY KEY, project_id TEXT NOT NULL, object_key TEXT NOT NULL, checksum TEXT NOT NULL, status TEXT NOT NULL, revision INTEGER NOT NULL, created_by TEXT NOT NULL, updated_at TEXT NOT NULL);
CREATE TABLE construction_daily_reports (id TEXT PRIMARY KEY, project_id TEXT NOT NULL, report_day TEXT NOT NULL, author_id TEXT NOT NULL, summary TEXT NOT NULL, status TEXT NOT NULL, revision INTEGER NOT NULL, updated_at TEXT NOT NULL);
CREATE TABLE construction_audit_events (id TEXT PRIMARY KEY, project_id TEXT NOT NULL, actor_id TEXT NOT NULL, action TEXT NOT NULL, target_id TEXT NOT NULL, created_at TEXT NOT NULL);
