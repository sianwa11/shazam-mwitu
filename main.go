package main

import (
	"log"

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

func main() {

	err := convert.ToWAV("songs/Risk-It-All.mp3", "songs/Risk-It-All.wav")
	if err != nil {
		log.Fatalf("convert to wav failed: %v", err)
	}

	wav, err := audio.ReadWav("songs/Risk-It-All.wav")
	if err != nil {
		log.Fatalf("read wav failed: %v", err)
	}
	mono, err := wav.ToMono()
	if err != nil {
		log.Fatalf("to mono failed: %v", err)
	}

	slices := fingerprint.Slice(fingerprint.ConvertToFloat64Array(mono.Samples), frameSize, hop)
	hammingWeights := fingerprint.HammingGenerator(frameSize)
	fft := fourier.NewFFT(frameSize)

	spectrogram := make([][]float64, 0, len(slices))
	for _, slice := range slices {
		fingerprint.ApplyWindow(slice, hammingWeights) // tapper the edges
		magnitude := fingerprint.FFT(slice, fft)       // transform to frequency domain
		spectrogram = append(spectrogram, magnitude)
	}

	visuals.WriteSpectrogramCSV(spectrogram, "visuals/spectrogram.csv")
}
