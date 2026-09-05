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

		scores := make(map[match]int)
		for _, h := range hashes {
			entries, err := store.LookupAddress(ctx, h.Address)
			if err != nil {
				log.Printf("lookup error for address %d: %v", h.Address, err)
				continue
			}

			AddMatches(scores, entries, h)
		}

		ranked := RankMatches(scores)
		totalHashes := len(hashes)

		// if len(ranked) > 0 {
		// 	log.Printf("%s: best votes=%d/%d (need >=%d)", f, ranked[0].Count, totalHashes, minVotes)
		// }

		result := BestMatch(ranked, totalHashes)
		switch result.Confidence {
		case ConfidentMatch:
			song, _ := store.GetSong(ctx, result.Song.SongID)
			log.Printf("%s: Definitely %q (votes=%d)", f, song.Title, result.Song.Count)

		case PossibleMatch:
			song, _ := store.GetSong(ctx, result.Song.SongID)
			log.Printf("%s: Might be %q (votes=%d)", f, song.Title, result.Song.Count)

		case NoMatch:
			log.Print("No Match found")
		}
	}
}
