-- name: CreateHash :one
INSERT INTO hashes (address, anchor_time, song_id) VALUES (?, ?, ?) RETURNING *;

-- name: GetHashByAddress :many
SELECT * FROM hashes WHERE address = ?;