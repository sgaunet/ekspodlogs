
-- name: PurgeLogs :exec
DELETE FROM logs;

-- name: InsertLog :exec
INSERT INTO logs (event_time, namespace_name, pod_name, container_name, log) VALUES (?, ?, ?, ?, ?);

-- name: GetLogs :many
SELECT * FROM logs ORDER BY event_time DESC LIMIT ? OFFSET ?;

-- name: CountLogs :one
SELECT COUNT(*) FROM logs;

-- name: GetLogsByPod :many
SELECT * FROM logs WHERE pod_name = ? ORDER BY event_time DESC LIMIT ? OFFSET ?;

-- name: CountLogsByPod :one
SELECT COUNT(*) FROM logs WHERE pod_name = ?;
