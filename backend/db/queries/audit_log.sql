-- name: ListOwnerAuditLogs :many
SELECT al.id, al.created_at, al.actor_id, u.name AS actor_name, al.actor_role, al.action, al.entity_type, al.entity_id, al.changes
FROM audit_logs al
LEFT JOIN users u ON u.id = al.actor_id
WHERE al.owner_id = sqlc.arg(owner_id)
  AND (sqlc.narg(entity_type)::varchar IS NULL OR al.entity_type = sqlc.narg(entity_type))
  AND (sqlc.narg(from_time)::timestamptz IS NULL OR al.created_at >= sqlc.narg(from_time))
  AND (sqlc.narg(to_time)::timestamptz IS NULL OR al.created_at < sqlc.narg(to_time))
ORDER BY al.created_at DESC, al.id DESC
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);

-- name: ListAdminAuditLogs :many
SELECT al.id, al.created_at, al.actor_id, u.name AS actor_name, al.actor_role, al.action, al.entity_type, al.entity_id,
       al.changes, al.ip, al.user_agent
FROM audit_logs al
LEFT JOIN users u ON u.id = al.actor_id
WHERE (sqlc.narg(actor_id)::bigint IS NULL OR al.actor_id = sqlc.narg(actor_id))
  AND (sqlc.narg(action)::varchar IS NULL OR al.action = sqlc.narg(action))
  AND (sqlc.narg(entity_type)::varchar IS NULL OR al.entity_type = sqlc.narg(entity_type))
  AND (sqlc.narg(entity_id)::bigint IS NULL OR al.entity_id = sqlc.narg(entity_id))
  AND (sqlc.narg(from_time)::timestamptz IS NULL OR al.created_at >= sqlc.narg(from_time))
  AND (sqlc.narg(to_time)::timestamptz IS NULL OR al.created_at < sqlc.narg(to_time))
ORDER BY al.created_at DESC, al.id DESC
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);
