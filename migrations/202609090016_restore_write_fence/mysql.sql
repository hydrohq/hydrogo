CREATE TRIGGER restore_block_pages_insert BEFORE INSERT ON pages FOR EACH ROW
BEGIN
    IF EXISTS (SELECT 1 FROM restore_controls WHERE id = 1 AND old_writes_blocked = 1) THEN
        SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'restore old writes blocked';
    END IF;
END;
CREATE TRIGGER restore_block_pages_update BEFORE UPDATE ON pages FOR EACH ROW
BEGIN
    IF EXISTS (SELECT 1 FROM restore_controls WHERE id = 1 AND old_writes_blocked = 1) THEN
        SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'restore old writes blocked';
    END IF;
END;
CREATE TRIGGER restore_block_pages_delete BEFORE DELETE ON pages FOR EACH ROW
BEGIN
    IF EXISTS (SELECT 1 FROM restore_controls WHERE id = 1 AND old_writes_blocked = 1) THEN
        SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'restore old writes blocked';
    END IF;
END;
CREATE TRIGGER restore_block_media_insert BEFORE INSERT ON media_assets FOR EACH ROW
BEGIN
    IF EXISTS (SELECT 1 FROM restore_controls WHERE id = 1 AND old_writes_blocked = 1) THEN
        SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'restore old writes blocked';
    END IF;
END;
CREATE TRIGGER restore_block_media_update BEFORE UPDATE ON media_assets FOR EACH ROW
BEGIN
    IF EXISTS (SELECT 1 FROM restore_controls WHERE id = 1 AND old_writes_blocked = 1) THEN
        SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'restore old writes blocked';
    END IF;
END;
CREATE TRIGGER restore_block_media_delete BEFORE DELETE ON media_assets FOR EACH ROW
BEGIN
    IF EXISTS (SELECT 1 FROM restore_controls WHERE id = 1 AND old_writes_blocked = 1) THEN
        SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'restore old writes blocked';
    END IF;
END;
CREATE TRIGGER restore_block_news_insert BEFORE INSERT ON news_items FOR EACH ROW
BEGIN
    IF EXISTS (SELECT 1 FROM restore_controls WHERE id = 1 AND old_writes_blocked = 1) THEN
        SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'restore old writes blocked';
    END IF;
END;
CREATE TRIGGER restore_block_news_update BEFORE UPDATE ON news_items FOR EACH ROW
BEGIN
    IF EXISTS (SELECT 1 FROM restore_controls WHERE id = 1 AND old_writes_blocked = 1) THEN
        SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'restore old writes blocked';
    END IF;
END;
CREATE TRIGGER restore_block_news_delete BEFORE DELETE ON news_items FOR EACH ROW
BEGIN
    IF EXISTS (SELECT 1 FROM restore_controls WHERE id = 1 AND old_writes_blocked = 1) THEN
        SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'restore old writes blocked';
    END IF;
END;
CREATE TRIGGER restore_block_jobs_insert BEFORE INSERT ON jobs FOR EACH ROW
BEGIN
    IF EXISTS (SELECT 1 FROM restore_controls WHERE id = 1 AND old_writes_blocked = 1) THEN
        SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'restore old writes blocked';
    END IF;
END;
CREATE TRIGGER restore_block_jobs_update BEFORE UPDATE ON jobs FOR EACH ROW
BEGIN
    IF EXISTS (SELECT 1 FROM restore_controls WHERE id = 1 AND old_writes_blocked = 1) THEN
        SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'restore old writes blocked';
    END IF;
END;
CREATE TRIGGER restore_block_jobs_delete BEFORE DELETE ON jobs FOR EACH ROW
BEGIN
    IF EXISTS (SELECT 1 FROM restore_controls WHERE id = 1 AND old_writes_blocked = 1) THEN
        SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'restore old writes blocked';
    END IF;
END;
