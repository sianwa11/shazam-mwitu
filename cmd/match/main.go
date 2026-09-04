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
		// 	ratio := float64(ranked[0].Count) / float64(totalHashes)
		// 	log.Printf("%s: best=%q votes=%d/%d ratio=%.4f (threshold ratio=%.4f)",
		// 		f, ranked[0].SongID, ranked[0].Count, totalHashes, ratio, matchCoefficient)
		// }

		match, ok := BestMatch(ranked, totalHashes)
		if !ok {
			log.Printf("%s no confident match found", f)
			continue
		}

		song, err := store.GetSong(ctx, match.SongID)
		if err != nil {
			log.Printf("%s matched song_id=%d but failed to load metadata: %v", f, match.SongID, err)
			continue
		}

		log.Printf("%s: Match -> %q (votes=%d/%d, offset=%d)", f, song.Title, match.Count, totalHashes, match.Offset)

	}

}
