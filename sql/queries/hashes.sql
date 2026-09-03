-- name: CreateHash :one
INSERT INTO hashes (address, anchor_time, song_id) VALUES (?, ?, ?) RETURNING *;