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
	db, err := sql.Open("sqlite3", "./songs.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	store := store.NewStore(db)
	ctx := context.Background()

	files, err := filepath.Glob("./recordings/*.mp4")
	if err != nil {
		log.Fatal(err)
	}

	for _, f := range files {
		_, peaks, err := pipeline.BuildFingerprint(f)
		if err != nil {
			log.Printf("skipping %s: %v", f, err)
			continue
		}

		hashes := fingerprint.Hashing(peaks)

		for _, h := range hashes {
			entries, err := store.LookupAddress(ctx, h.Address)
			if err != nil {
				log.Printf("lookup error for address %d: %v", h.Address, err)
				continue
			}
			if len(entries) > 0 {
				log.Printf("address %d -> %d matches: %v", h.Address, len(entries), entries)
			}

		}
	}
}
