-- name: CreateUserNotification :one
INSERT INTO user_notifications (user_id, category, title, body, link)
VALUES ($1, $2, $3, $4, $5)
RETURNING id;

-- name: EnqueueEmail :one
INSERT INTO email_outbox (to_email, subject, body_text)
VALUES ($1, $2, $3)
RETURNING id;
