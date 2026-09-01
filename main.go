package main

import (
	"log"
	"path/filepath"

	"github.com/sianwa11/shazam-mwitu/fingerprint"
	"github.com/sianwa11/shazam-mwitu/pipeline"
)

func main() {
	db := make(map[string][]fingerprint.Peak)

	files, _ := filepath.Glob("songs/*.mp3")
	for _, f := range files {
	id, peaks, err := pipeline.BuildFingerprint(f)
		if err != nil {
			log.Printf("skipping %s: %v", f, err)
			continue
		}

		db[id] = peaks
		log.Printf("%s: %d peaks", id, len(peaks))
	}

}
