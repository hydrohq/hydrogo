CREATE OR REPLACE FUNCTION hydro_block_restore_writes() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF EXISTS (SELECT 1 FROM restore_controls WHERE id = 1 AND old_writes_blocked = 1) THEN
        RAISE EXCEPTION 'restore old writes blocked';
    END IF;
    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER restore_block_pages BEFORE INSERT OR UPDATE OR DELETE ON pages FOR EACH ROW EXECUTE FUNCTION hydro_block_restore_writes();
CREATE TRIGGER restore_block_media BEFORE INSERT OR UPDATE OR DELETE ON media_assets FOR EACH ROW EXECUTE FUNCTION hydro_block_restore_writes();
CREATE TRIGGER restore_block_news BEFORE INSERT OR UPDATE OR DELETE ON news_items FOR EACH ROW EXECUTE FUNCTION hydro_block_restore_writes();
CREATE TRIGGER restore_block_jobs BEFORE INSERT OR UPDATE OR DELETE ON jobs FOR EACH ROW EXECUTE FUNCTION hydro_block_restore_writes();
