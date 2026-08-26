package visuals

import (
	"encoding/csv"
	"os"
	"strconv"
)

func WriteSpectrogramCSV(spectrogram [][]float64, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	w := csv.NewWriter(file)
	defer w.Flush()

	for _, frame := range spectrogram {
		row := make([]string, len(frame))
		for i, v := range frame {
			row[i] = strconv.FormatFloat(v, 'f', 4, 64)
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}
	return nil
}
