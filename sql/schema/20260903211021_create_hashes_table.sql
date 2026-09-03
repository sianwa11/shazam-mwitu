-- +goose Up
-- +goose StatementBegin
CREATE TABLE hashes (
  address     INTEGER NOT NULL,
  anchor_time INTEGER NOT NULL,
  song_id     INTEGER NOT NULL,
  FOREIGN KEY (song_id) REFERENCES songs(id)
);

CREATE INDEX idx_hashes_address ON hashes(address);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE hashes;
-- +goose StatementEnd
