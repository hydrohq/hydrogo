ALTER TABLE pages ADD COLUMN published_locale VARCHAR(16);
ALTER TABLE pages ADD COLUMN published_slug VARCHAR(256);
ALTER TABLE pages ADD COLUMN published_title VARCHAR(512);
ALTER TABLE pages ADD COLUMN published_body LONGTEXT;
