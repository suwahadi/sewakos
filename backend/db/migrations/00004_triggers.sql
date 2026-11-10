-- +goose Up
-- +goose StatementBegin
CREATE FUNCTION queue_photo_storage_deletion() RETURNS trigger AS $$
BEGIN
  INSERT INTO storage_deletions (object_key) VALUES (OLD.object_key);
  INSERT INTO storage_deletions (object_key) VALUES (OLD.thumb_key);
  RETURN OLD;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER property_photos_queue_storage_deletion
  AFTER DELETE ON property_photos
  FOR EACH ROW
  EXECUTE FUNCTION queue_photo_storage_deletion();

-- +goose StatementBegin
CREATE FUNCTION reject_append_only_mutation() RETURNS trigger AS $$
BEGIN
  RAISE EXCEPTION '% is append-only: % not allowed', TG_TABLE_NAME, TG_OP;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER ledger_entries_append_only
  BEFORE UPDATE OR DELETE ON ledger_entries
  FOR EACH ROW
  EXECUTE FUNCTION reject_append_only_mutation();

CREATE TRIGGER audit_logs_append_only
  BEFORE UPDATE OR DELETE ON audit_logs
  FOR EACH ROW
  EXECUTE FUNCTION reject_append_only_mutation();

-- +goose Down
DROP TRIGGER audit_logs_append_only ON audit_logs;
DROP TRIGGER ledger_entries_append_only ON ledger_entries;
DROP FUNCTION reject_append_only_mutation();
DROP TRIGGER property_photos_queue_storage_deletion ON property_photos;
DROP FUNCTION queue_photo_storage_deletion();
