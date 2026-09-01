package pipeline

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/sianwa11/shazam-mwitu/audio"
	"github.com/sianwa11/shazam-mwitu/convert"
	"github.com/sianwa11/shazam-mwitu/fingerprint"
	"github.com/sianwa11/shazam-mwitu/visuals"
	"gonum.org/v1/gonum/dsp/fourier"
)

const (
	frameSize = 1024
	hop       = 512
)

func BuildFingerprint(mp3Path string) (string, []fingerprint.Peak, error) {
	wavPath := strings.TrimSuffix(mp3Path, filepath.Ext(mp3Path)) + ".wav"
	if err := convert.ToWAV(mp3Path, wavPath); err != nil {
		return "", nil, fmt.Errorf("convert: %w", err)
	}

	wav, err := audio.ReadWav(wavPath)
	if err != nil {
		return "", nil, fmt.Errorf("convert: %w", err)
	}

	mono, err := wav.ToMono()
	if err != nil {
		return "", nil, fmt.Errorf("convert: %w", err)
	}

	samples := fingerprint.ConvertToFloat64Array(mono.Samples)

	slices := fingerprint.Slice(samples, frameSize, hop)

	hammingWeights := fingerprint.HammingGenerator(frameSize)
	fft := fourier.NewFFT(frameSize)

	spectrogram := make([][]float64, 0, len(slices))
	for _, slice := range slices {
		fingerprint.ApplyWindow(slice, hammingWeights) // tapper the edges
		magnitude := fingerprint.FFT(slice, fft)       // transform slices to frequency domain
		spectrogram = append(spectrogram, magnitude)
	}

	visuals.WriteSpectrogramCSV(spectrogram, "visuals/spectrogram.csv")

	peaks := fingerprint.PeakPicking(spectrogram)

	return filepath.Base(mp3Path), peaks, nil
}
