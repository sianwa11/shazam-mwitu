package convert

import (
	"fmt"
	"os/exec"
)

func ToWAV(inputPath, outputPath string) error {

	cmd := exec.Command("ffmpeg", "-i", inputPath, "-ar", "44100", "-ac", "2", "-y", outputPath)

	output, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf("ffmpeg failed: %w\n%s", err, output)
	}

	return nil
}
