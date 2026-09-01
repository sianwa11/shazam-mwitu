package fingerprint

const (
	zoneSize = 5
)

type Hash struct {
	Address    [3]int
	AnchorTime int
}

func Hashing(peaks []Peak) []Hash {
	hashes := make([]Hash, 0)

	for i := 0; i+zoneSize <= len(peaks); i++ {
		anchorIndex := i - 3
		if anchorIndex < 0 {
			continue
		}

		anchor := peaks[anchorIndex]
		zone := peaks[i : i+zoneSize]

		for _, point := range zone {
			delta := point.Time - anchor.Time
			hashes = append(hashes, Hash{
				Address:    [3]int{anchor.Frequency, point.Frequency, delta},
				AnchorTime: anchor.Time,
			})
		}

	}

	return hashes
}
