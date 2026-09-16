CREATE TABLE hydro_forms (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  schema_json TEXT NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);
