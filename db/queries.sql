
-- name: PurgeAllLogs :exec
DELETE FROM logs;

-- name: PurgeLogs :exec
DELETE FROM logs WHERE profile = ? AND loggroup = ?;

-- name: InsertLog :exec
INSERT INTO logs (event_time,profile,loggroup, namespace_name, pod_name, container_name, log) VALUES (? ,?, ?, ?, ?, ?, ?);

-- name: GetLogs :many
SELECT * FROM logs 
WHERE event_time >= sqlc.arg(begindate) and event_time <= sqlc.arg(enddate)
    AND loggroup = sqlc.arg(loggroup)
    AND profile = sqlc.arg(profile)
ORDER BY event_time;

-- name: GetLogsOfPod :many
SELECT * FROM logs 
WHERE event_time >= sqlc.arg(begindate) and event_time <= sqlc.arg(enddate)
    AND loggroup = sqlc.arg(loggroup)
    AND profile = sqlc.arg(profile)
    AND pod_name like sqlc.arg(pod_name)
    ORDER BY event_time;

-- name: CountLogs :one
SELECT COUNT(*) FROM logs;
