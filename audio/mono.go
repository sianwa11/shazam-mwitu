package audio

import "fmt"

func (wav *WAVFile) ToMono() (*WAVFile, error) {
	if wav.NumChannels == 1 {
		return wav, nil
	}

	if len(wav.Samples)%2 != 0 {
		return nil, fmt.Errorf("corrupted wav file")
	}

	monoSamples := len(wav.Samples) / 2
	mono := make([]int16, monoSamples)

	i := 0
	j := 0
	for {
		if j == monoSamples {
			break
		}
		left := int32(wav.Samples[i])
		right := int32(wav.Samples[i+1])
		res := (left + right) / 2
		mono[j] = int16(res)
		i = i + 2
		j++
	}

	return &WAVFile{
		SampleRate:    wav.SampleRate,
		NumChannels:   1,
		BitsPerSample: wav.BitsPerSample,
		Samples:       mono,
	}, nil
}
