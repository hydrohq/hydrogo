ALTER TABLE news_items ADD COLUMN key_takeaways LONGTEXT NULL;
UPDATE news_items SET key_takeaways = '' WHERE key_takeaways IS NULL;

CREATE TRIGGER news_key_takeaways_default BEFORE INSERT ON news_items FOR EACH ROW
BEGIN
    IF NEW.key_takeaways IS NULL THEN SET NEW.key_takeaways = ''; END IF;
END;
