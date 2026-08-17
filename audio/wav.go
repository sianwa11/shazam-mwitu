package audio

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

type WAVFile struct {
	SampleRate    uint32
	NumChannels   uint16
	BitsPerSample uint16
	Samples       []int16
}

func ReadWav(path string) (*WAVFile, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	wavFile := &WAVFile{}

	//---- RIFF Header
	b1 := make([]byte, 12)
	n, err := io.ReadFull(file, b1)
	if err != nil {
		return nil, err
	}

	if string(b1[0:4]) != "RIFF" {
		return nil, fmt.Errorf("not a valid WAV file")
	}

	if string(b1[8:n]) != "WAVE" {
		return nil, fmt.Errorf("not a valid WAV file")
	}

	//--- fmt chunk
	b2 := make([]byte, 24)
	n1, err := io.ReadFull(file, b2)
	if err != nil {
		return nil, err
	}

	if string(b2[0:4]) != "fmt " {
		return nil, fmt.Errorf("invalid fmt")
	}

	audioFormat := binary.LittleEndian.Uint16(b2[8:10])
	if audioFormat != 1 {
		return nil, fmt.Errorf("not PCM format: %d", audioFormat)
	}

	wavFile.NumChannels = binary.LittleEndian.Uint16(b2[10:12])
	wavFile.SampleRate = binary.LittleEndian.Uint32(b2[12:16])
	wavFile.BitsPerSample = binary.LittleEndian.Uint16(b2[22:n1])

	// --data chunk
	for {
		b3 := make([]byte, 8)
		_, err := io.ReadFull(file, b3)
		if err != nil {
			return nil, fmt.Errorf("data chunk not found: %w", err)
		}

		chunkID := string(b3[0:4])
		chunkSize := binary.LittleEndian.Uint32(b3[4:8])

		if chunkID == "data" {
			dataSize := chunkSize

			sampleData := make([]byte, dataSize)
			_, err := io.ReadFull(file, sampleData)
			if err != nil {
				return nil, fmt.Errorf("failed to read sample data: %w", err)
			}

			// Convert bytes to int16 samples
			numSamples := int(dataSize) / 2 // 2 bytes per sample
			samples := make([]int16, numSamples)

			for i := 0; i < numSamples; i++ {
				// Read 2 bytes and convert to int16
				val := binary.LittleEndian.Uint16(sampleData[i*2 : i*2+2])
				samples[i] = int16(val) // reinterpret as signed int16
			}

			wavFile.Samples = samples

			break
		}

		_, err = file.Seek(int64(chunkSize), io.SeekCurrent)
		if err != nil {
			return nil, fmt.Errorf("failed to skip chunk: %w", err)
		}
	}

	return wavFile, nil
}
