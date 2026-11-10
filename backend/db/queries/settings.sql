-- name: GetSetting :one
SELECT value FROM settings WHERE key = sqlc.arg(key);
