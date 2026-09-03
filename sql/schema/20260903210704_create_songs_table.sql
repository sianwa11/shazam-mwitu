-- +goose Up
-- +goose StatementBegin
CREATE TABLE songs (
  id    INTEGER PRIMARY KEY AUTOINCREMENT,
  title TEXT NOT NULL,
  path  TEXT NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE songs;
-- +goose StatementEnd
