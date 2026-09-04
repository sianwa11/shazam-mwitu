package main

import (
	"sort"

	"github.com/sianwa11/shazam-mwitu/db"
	"github.com/sianwa11/shazam-mwitu/fingerprint"
)

const matchCoefficient = 0 // tune this

type match struct {
	songID int64
	offset int64
}

type SongMatch struct {
	SongID int64
	Offset int64
	Count  int
}

func AddMatches(scores map[match]int, entries []db.Hash, hash fingerprint.Hash) {
	for _, e := range entries {
		offset := e.AnchorTime - int64(hash.AnchorTime)
		scores[match{songID: e.SongID, offset: offset}]++
	}
}

func RankMatches(scores map[match]int) []SongMatch {
	best := make(map[int64]SongMatch)

	for m, count := range scores {
		if existing, ok := best[m.songID]; !ok || count > existing.Count {
			best[m.songID] = SongMatch{SongID: m.songID, Offset: m.offset, Count: count}
		}
	}

	ranked := make([]SongMatch, 0, len(best))
	for _, sm := range best {
		ranked = append(ranked, sm)
	}
	sort.Slice(ranked, func(i, j int) bool { return ranked[i].Count > ranked[j].Count })

	return ranked
}

func BestMatch(ranked []SongMatch, totalHashes int) (SongMatch, bool) {
	if len(ranked) == 0 {
		return SongMatch{}, false
	}

	best := ranked[0]
	threshold := float64(totalHashes) * matchCoefficient


	if float64(best.Count) < threshold {
		return SongMatch{}, false
	}

	return best, true
}
