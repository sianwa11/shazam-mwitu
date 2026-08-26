package fingerprint

import (
	"math"
	"math/cmplx"

	"gonum.org/v1/gonum/dsp/fourier"
)

// DFT computes the discrete Fourier transform of a real-valued frame in O(N^2) time.
func DFT(frame []float64) []complex128 {
	n := len(frame)
	out := make([]complex128, n)

	for k := 0; k < n; k++ {
		var real, imag float64
		for t := 0; t < n; t++ {
			angle := 2 * math.Pi * float64(k) * float64(t) / float64(n)
			real += frame[t] * math.Cos(angle)
			imag -= frame[t] * math.Sin(angle)
		}
		out[k] = complex(real, imag)
	}

	return out
}

// FFT computes the fast Fourier transform of a real-valued frame in O(N log N) time using the gonum library. It returns the magnitude of the complex coefficients.
func FFT(frame []float64, fft *fourier.FFT) []float64 {
	coeff := fft.Coefficients(nil, frame)

	out := make([]float64, len(coeff))
	for i, c := range coeff {
		out[i] = cmplx.Abs(c)
	}
	return out
}
