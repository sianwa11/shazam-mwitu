package main

import (
	"log"

	"github.com/sianwa11/shazam-mwitu/audio"
	"github.com/sianwa11/shazam-mwitu/convert"
	"github.com/sianwa11/shazam-mwitu/fingerprint"
)

const (
	frameSize = 1024
	hop       = 512
)

func main() {

	err := convert.ToWAV("songs/Risk-It-All.mp3", "songs/Risk-It-All.wav")
	if err != nil {
		log.Print(err)
	}

	wav, err := audio.ReadWav("songs/Risk-It-All.wav")
	if err != nil {
		log.Print(err)
	}
	mono, err := wav.ToMono()
	if err != nil {
		log.Print(err)
	}

	log.Printf("Sample Rate: %d", wav.SampleRate)
	log.Printf("Num Channels: %d", wav.NumChannels)
	log.Printf("Bits Per Sample: %d", wav.BitsPerSample)
	log.Printf("Num Samples Len: %v", len(wav.Samples))
	log.Printf("Mono Samples Len: %v", len(mono.Samples))

	slices := fingerprint.Slice(fingerprint.ConvertToFloat64Array(mono.Samples), frameSize, hop)

	log.Printf("Number of frames: %d", len(slices))
	log.Printf("First frame len: %d", len(slices[0]))
	log.Printf("Last frame len: %d", len(slices[len(slices)-1]))

	hammingWeights := fingerprint.HammingGenerator(frameSize)
	log.Printf("Hamming Weights len: %d", len(hammingWeights))

	log.Printf("Before windowing, frame 1000: %v", slices[1000][:20]) // just first 20 values

	for _, slice := range slices {
		fingerprint.ApplyWindow(slice, hammingWeights)
	}

	log.Printf("After windowing, frame 1000: %v", slices[1000][:20])

}
