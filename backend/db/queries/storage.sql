-- name: QueueStorageDeletion :exec
INSERT INTO storage_deletions (object_key) VALUES (sqlc.arg(object_key));

-- name: SelectDueStorageDeletions :many
SELECT id, object_key, attempts FROM storage_deletions
WHERE next_attempt_at <= now()
ORDER BY id
LIMIT 100
FOR UPDATE SKIP LOCKED;

-- name: DeleteStorageDeletion :exec
DELETE FROM storage_deletions WHERE id = sqlc.arg(id);

-- name: MarkStorageDeletionFailed :exec
UPDATE storage_deletions
SET attempts = attempts + 1, next_attempt_at = sqlc.arg(next_attempt_at), last_error = sqlc.arg(last_error), updated_at = now()
WHERE id = sqlc.arg(id);

-- name: ListPhotoObjectKeys :many
SELECT id, object_key, thumb_key FROM property_photos;

-- name: TryAdvisoryXactLock :one
SELECT pg_try_advisory_xact_lock(sqlc.arg(lock_key)::bigint);
