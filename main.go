package main

import (
	"context"
	"database/sql"
	"log"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"

	"github.com/sianwa11/shazam-mwitu/fingerprint"
	"github.com/sianwa11/shazam-mwitu/pipeline"
	"github.com/sianwa11/shazam-mwitu/store"
)

func main() {

	db, err := sql.Open("sqlite3", "songs.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	store := store.NewStore(db)
	ctx := context.Background()

	files, _ := filepath.Glob("songs/*.mp3")

	for _, f := range files {
		songInfo, peaks, err := pipeline.BuildFingerprint(f)
		if err != nil {
			log.Printf("skipping %s: %v", f, err)
			continue
		}

		hashes := fingerprint.Hashing(peaks)

		if err := store.InsertSongWithHashes(ctx, songInfo.Name, songInfo.Path, hashes); err != nil {
			log.Printf("failed to store %s: %v", songInfo.Name, err)
			continue
		}

	}

}
