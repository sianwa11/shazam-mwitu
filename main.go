package main

import (
	"log"

	"github.com/sianwa11/shazam-mwitu/audio"
	"github.com/sianwa11/shazam-mwitu/convert"
)

func main() {

	err := convert.ToWAV("songs/Risk-It-All.mp3", "songs/Risk-It-All.wav")
	if err != nil {
		log.Fatal(err)
	}

	wav, err := audio.ReadWav("songs/Risk-It-All.wav")
	if err != nil {
		log.Fatal(err)
	}

	mono, err := audio.ToMono(wav)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Sample Rate: %d", wav.SampleRate)
	log.Printf("Num Channels: %d", wav.NumChannels)
	log.Printf("Bits Per Sample: %d", wav.BitsPerSample)
	log.Printf("Num Samples Len: %v", len(wav.Samples))
	log.Printf("Mono Len: %v", len(mono))

}
