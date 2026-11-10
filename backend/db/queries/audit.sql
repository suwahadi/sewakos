-- name: InsertAuditLog :one
INSERT INTO audit_logs (
  actor_id, actor_role, action, entity_type, entity_id, owner_id, changes, ip, user_agent
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9
)
RETURNING id;
