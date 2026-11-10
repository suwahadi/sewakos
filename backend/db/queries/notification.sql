-- name: CreateUserNotification :one
INSERT INTO user_notifications (user_id, category, title, body, link)
VALUES ($1, $2, $3, $4, $5)
RETURNING id;

-- name: EnqueueEmail :one
INSERT INTO email_outbox (to_email, subject, body_text)
VALUES ($1, $2, $3)
RETURNING id;

-- name: ListUserNotifications :many
SELECT id, category, title, body, link, read_at, created_at
FROM user_notifications
WHERE user_id = $1 AND (sqlc.narg(category)::varchar IS NULL OR category = sqlc.narg(category))
ORDER BY created_at DESC, id DESC
LIMIT $2 OFFSET $3;

-- name: CountUserNotifications :one
SELECT COUNT(*) FROM user_notifications
WHERE user_id = $1 AND (sqlc.narg(category)::varchar IS NULL OR category = sqlc.narg(category));

-- name: CountUnreadNotifications :one
SELECT COUNT(*) FROM user_notifications WHERE user_id = $1 AND read_at IS NULL;

-- name: GetUserNotification :one
SELECT id, user_id, read_at FROM user_notifications WHERE id = $1;

-- name: MarkUserNotificationRead :exec
UPDATE user_notifications SET read_at = now(), updated_at = now() WHERE id = $1 AND read_at IS NULL;
