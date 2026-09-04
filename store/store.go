package store

import (
	"context"
	"database/sql"

	"github.com/sianwa11/shazam-mwitu/db"
	"github.com/sianwa11/shazam-mwitu/fingerprint"
)

type Store struct {
	db      *sql.DB
	queries *db.Queries // from the db package
}

func NewStore(d *sql.DB) *Store {
	return &Store{
		db:      d,
		queries: db.New(d),
	}
}

func (s *Store) InsertSongWithHashes(ctx context.Context, title, path string, hashes []fingerprint.Hash) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	qtx := s.queries.WithTx(tx)

	songs, err := qtx.CreateSong(ctx, db.CreateSongParams{
		Title: title,
		Path:  path,
	})
	if err != nil {
		return err
	}

	for _, hash := range hashes {
		_, err := qtx.CreateHash(ctx, db.CreateHashParams{
			Address:    int64(hash.Address),
			AnchorTime: int64(hash.AnchorTime),
			SongID:     songs.ID,
		})

		if err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (s *Store) LookupAddress(ctx context.Context, address uint32) ([]db.Hash, error) {
	return s.queries.GetHashByAddress(ctx, int64(address))
}

func (s *Store) GetSong(ctx context.Context, songID int64) (db.Song, error) {
	return s.queries.GetSong(ctx, int64(songID))
}
