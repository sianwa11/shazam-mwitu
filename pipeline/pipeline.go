package pipeline

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/sianwa11/shazam-mwitu/audio"
	"github.com/sianwa11/shazam-mwitu/convert"
	"github.com/sianwa11/shazam-mwitu/fingerprint"
	"github.com/sianwa11/shazam-mwitu/registry"
	"github.com/sianwa11/shazam-mwitu/visuals"
	"gonum.org/v1/gonum/dsp/fourier"
)

const (
	frameSize = 1024
	hop       = 512
)

type Entry struct {
	AnchorTime int
	SongID     int
}

type FingerprintDB struct {
	entries map[uint32][]Entry
}

func NewFingerprintDB() *FingerprintDB {
	return &FingerprintDB{
		entries: make(map[uint32][]Entry),
	}
}

func (db *FingerprintDB) Store(songID int, hashes []fingerprint.Hash) {
	for _, hash := range hashes {
		entry := Entry{
			AnchorTime: hash.AnchorTime,
			SongID:     songID,
		}
		db.entries[hash.Address] = append(db.entries[hash.Address], entry)
	}
}

func (db *FingerprintDB) Lookup(address uint32) []Entry {
	return db.entries[address]
}

func BuildFingerprint(mp3Path string) (registry.SongInfo, []fingerprint.Peak, error) {
	wavPath := strings.TrimSuffix(mp3Path, filepath.Ext(mp3Path)) + ".wav"
	if err := convert.ToWAV(mp3Path, wavPath); err != nil {
		return registry.SongInfo{}, nil, fmt.Errorf("convert: %w", err)
	}

	filename := filepath.Base(wavPath)
	ext := filepath.Ext(filename)
	songName := strings.TrimSuffix(filename, ext)

	registry := registry.NewSongRegistry()
	info := registry.Register(songName, wavPath)

	wav, err := audio.ReadWav(wavPath)
	if err != nil {
		return info, nil, fmt.Errorf("convert: %w", err)
	}

	mono, err := wav.ToMono()
	if err != nil {
		return info, nil, fmt.Errorf("convert: %w", err)
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

	return info, peaks, nil
}
