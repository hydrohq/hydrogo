ALTER TABLE pages ADD COLUMN blocks LONGTEXT NULL;
UPDATE pages SET blocks = '[]' WHERE blocks IS NULL;
ALTER TABLE pages MODIFY COLUMN blocks LONGTEXT NOT NULL;
ALTER TABLE pages ADD COLUMN published_blocks LONGTEXT NULL;
UPDATE pages SET published_blocks = '[]' WHERE published_blocks IS NULL;
ALTER TABLE pages MODIFY COLUMN published_blocks LONGTEXT NOT NULL;
ALTER TABLE page_revisions ADD COLUMN blocks LONGTEXT NULL;
UPDATE page_revisions SET blocks = '[]' WHERE blocks IS NULL;
ALTER TABLE page_revisions MODIFY COLUMN blocks LONGTEXT NOT NULL;

CREATE TRIGGER pages_blocks_defaults BEFORE INSERT ON pages FOR EACH ROW
BEGIN
    IF NEW.blocks IS NULL THEN SET NEW.blocks = '[]'; END IF;
    IF NEW.published_blocks IS NULL THEN SET NEW.published_blocks = '[]'; END IF;
END;
CREATE TRIGGER page_revisions_blocks_defaults BEFORE INSERT ON page_revisions FOR EACH ROW
BEGIN
    IF NEW.blocks IS NULL THEN SET NEW.blocks = '[]'; END IF;
END;
