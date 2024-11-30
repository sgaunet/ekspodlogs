
-- name: PurgeAllLogs :exec
DELETE FROM logs;

-- name: PurgeLogs :exec
DELETE FROM logs WHERE profile = ? AND loggroup = ?;

-- name: InsertLog :exec
INSERT INTO logs (event_time,profile,loggroup, namespace_name, pod_name, container_name, log) VALUES (? ,?, ?, ?, ?, ?, ?);

-- name: GetLogs :many
SELECT * FROM logs 
WHERE event_time >= sqlc.arg(begindate) and event_time <= sqlc.arg(enddate)
    AND loggroup = ?
    AND profile = ?
ORDER BY event_time;

-- name: CountLogs :one
SELECT COUNT(*) FROM logs;

-- name: GetLogsByPod :many
SELECT * FROM logs 
    WHERE pod_name = ?
        AND event_time >= ?
        AND event_time <= ? 
    ORDER BY event_time DESC;
