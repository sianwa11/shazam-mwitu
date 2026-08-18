package fingerprint

import "math"

func HammingGenerator(size int) []float64 {
	window := make([]float64, size)
	for i := 0; i < size; i++ {
		window[i] = 0.54 - 0.46*math.Cos(2*math.Pi*float64(i)/float64(size-1))
	}
	return window
}

func ApplyWindow(frame, weights []float64) {
	for i := range frame {
		frame[i] *= weights[i]
	}
}

func Slice(samples []float64, frameSize, hop int) [][]float64 {
	slices := [][]float64{}
	i := 0

	for {
		if i+frameSize > len(samples) {
			break
		}

		frame := make([]float64, frameSize)
		copy(frame, samples[i:i+frameSize])
		slices = append(slices, frame)

		i = i + hop
	}

	return slices
}

func ConvertToFloat64Array(samples []int16) []float64 {
	float64Slice := make([]float64, 0, len(samples))

	for _, v := range samples {
		float64Slice = append(float64Slice, float64(v))
	}

	return float64Slice
}
