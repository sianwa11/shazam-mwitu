package fingerprint

import "fmt"

const (
	zoneSize     = 5
	anchorOffset = 3
)

type Hash struct {
	Address    uint32
	AnchorTime int
}

func (h Hash) String() string {
	return fmt.Sprintf("Hash{Address: 0x%08X, AnchorTime: %d}", h.Address, h.AnchorTime)
}

func Hashing(peaks []Peak) []Hash {
	hashes := make([]Hash, 0)

	for i := 0; i+zoneSize <= len(peaks); i++ {
		anchorIndex := i - anchorOffset
		if anchorIndex < 0 {
			continue
		}

		anchor := peaks[anchorIndex]
		zone := peaks[i : i+zoneSize]

		for _, point := range zone {
			delta := point.Time - anchor.Time
			address := packAddress(anchor.Frequency, point.Frequency, delta)

			hashes = append(hashes, Hash{
				Address:    address,
				AnchorTime: anchor.Time,
			})
		}

	}

	return hashes
}

func packAddress(anchorFrequency, pointFrequency, deltaTime int) uint32 {
	af := uint32(anchorFrequency) & 0x1FF // 9 bits
	pf := uint32(pointFrequency) & 0x1FF  // 9 bits
	dt := uint32(deltaTime) & 0x3FFF      // 14 bits (0x3FFF = 16383)

	return af<<23 | pf<<14 | dt
}
