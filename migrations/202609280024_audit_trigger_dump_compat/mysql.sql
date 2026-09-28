DROP TRIGGER audit_events_immutable_update;
DROP TRIGGER audit_events_immutable_delete;
CREATE TRIGGER audit_events_immutable_update BEFORE UPDATE ON audit_events FOR EACH ROW
BEGIN
    SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'audit events are immutable';
END;
CREATE TRIGGER audit_events_immutable_delete BEFORE DELETE ON audit_events FOR EACH ROW
BEGIN
    SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'audit events are immutable';
END;
