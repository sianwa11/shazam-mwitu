package fingerprint

type Peak struct {
	Time      int
	Frequency int
}

const (
	timeRadius     = 3
	freqRadius     = 10
	thresholdRatio = 0.1
)

func PeakPicking(spectogram [][]float64) []Peak {
	maxMag := maxMagnitude(spectogram)
	minThreshold := maxMag * thresholdRatio

	peaks := []Peak{}
	for t := range spectogram {
		for f := range spectogram[t] {
			if isLocalMax(spectogram, t, f, timeRadius, freqRadius) && spectogram[t][f] >= minThreshold {
				peaks = append(peaks, Peak{Time: t, Frequency: f})
			}
		}
	}

	return peaks
}

func isLocalMax(spectogram [][]float64, t, f int, timeRadius, freqRadius int) bool {

	numFrames := len(spectogram)
	numBins := len(spectogram[0])

	tStart := max(0, t-timeRadius)
	tEnd := min(numFrames-1, t+timeRadius)
	fStart := max(0, f-freqRadius)
	fEnd := min(numBins-1, f+freqRadius)

	isPeak := true
	for tt := tStart; tt <= tEnd; tt++ {
		for ff := fStart; ff <= fEnd; ff++ {
			if spectogram[tt][ff] > spectogram[t][f] {
				isPeak = false
			}
		}
	}

	return isPeak
}

func maxMagnitude(spectogram [][]float64) float64 {
	max := 0.0

	for _, frame := range spectogram {
		for _, mag := range frame {
			if mag > max {
				max = mag
			}
		}
	}
	return max
}
