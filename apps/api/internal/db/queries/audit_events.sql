-- name: InsertAuditEvent :one
INSERT INTO audit_events (id, villa_slug, type, message, actor_email)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListAuditEvents :many
SELECT * FROM audit_events
WHERE villa_slug = sqlc.arg('villa_slug')
ORDER BY created_at DESC
LIMIT sqlc.arg('lim') OFFSET sqlc.arg('off');

-- name: CountAuditEvents :one
SELECT COUNT(*) FROM audit_events
WHERE villa_slug = $1;
