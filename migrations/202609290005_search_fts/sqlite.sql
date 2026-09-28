CREATE VIRTUAL TABLE hydro_search_fts USING fts5(title, body, content='hydro_search_documents', content_rowid='rowid', tokenize='unicode61 remove_diacritics 2');
CREATE TRIGGER hydro_search_fts_insert AFTER INSERT ON hydro_search_documents BEGIN
  INSERT INTO hydro_search_fts(rowid,title,body) VALUES (new.rowid,new.title,new.body);
END;
CREATE TRIGGER hydro_search_fts_delete AFTER DELETE ON hydro_search_documents BEGIN
  INSERT INTO hydro_search_fts(hydro_search_fts,rowid,title,body) VALUES ('delete',old.rowid,old.title,old.body);
END;
CREATE TRIGGER hydro_search_fts_update AFTER UPDATE ON hydro_search_documents BEGIN
  INSERT INTO hydro_search_fts(hydro_search_fts,rowid,title,body) VALUES ('delete',old.rowid,old.title,old.body);
  INSERT INTO hydro_search_fts(rowid,title,body) VALUES (new.rowid,new.title,new.body);
END;
INSERT INTO hydro_search_fts(hydro_search_fts) VALUES ('rebuild');
