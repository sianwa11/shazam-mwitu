-- name: CreateSong :one
INSERT INTO songs (title, path) VALUES (?, ?) RETURNING *;

-- name: GetSong :one
SELECT * FROM songs WHERE id = ?;

-- name: ListSongs :many
SELECT * FROM songs ORDER BY id;