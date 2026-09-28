CREATE TABLE hydro_search_documents (
    id VARCHAR(128) PRIMARY KEY,
    locale VARCHAR(16) NOT NULL,
    title VARCHAR(512) NOT NULL,
    body LONGTEXT NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE INDEX idx_hydro_search_documents_locale_id ON hydro_search_documents(locale, id);
