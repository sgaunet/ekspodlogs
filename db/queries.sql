
-- name: PurgeAll :exec
DELETE FROM logs;

-- name: PurgeSpecificPeriod :exec
DELETE FROM logs WHERE profile = sqlc.arg(profile) AND loggroup = sqlc.arg(loggroup)
    AND pod_name like sqlc.arg(pod_name)
    AND event_time >= sqlc.arg(begindate) 
    AND event_time <= sqlc.arg(enddate);

-- name: PurgeSpecificLogPodLogs :exec
DELETE FROM logs
WHERE profile = sqlc.arg(profile) 
  AND loggroup = sqlc.arg(loggroup)
  AND pod_name LIKE sqlc.arg(pod_name);

-- name: InsertLog :exec
INSERT INTO logs (event_time, profile, loggroup, namespace_name, pod_name, container_name, log) VALUES (? , ?, ?, ?, ?, ?, ?);

-- name: GetLogs :many
SELECT * FROM logs 
WHERE event_time >= sqlc.arg(begindate) and event_time <= sqlc.arg(enddate)
    AND loggroup = sqlc.arg(loggroup)
    AND profile = sqlc.arg(profile)
    AND pod_name like sqlc.arg(pod_name)
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
