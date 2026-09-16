CREATE TABLE hydro_search_documents (
  id TEXT PRIMARY KEY,
  locale TEXT NOT NULL,
  title TEXT NOT NULL,
  body TEXT NOT NULL
);
CREATE INDEX idx_hydro_search_documents_locale_id ON hydro_search_documents(locale, id);
