package audio

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// Play plays the configured sound file using a local player.
func Play(path string) error {
	player, args, err := playerCommand(path)
	if err != nil {
		return err
	}

	cmd := exec.Command(player, args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("play sound with %s: %w: %s", player, err, strings.TrimSpace(string(output)))
	}

	return nil
}

func playerCommand(path string) (string, []string, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".mp3":
		if _, err := exec.LookPath("mpg123"); err == nil {
			return "mpg123", []string{"-q", path}, nil
		}
	}

	if _, err := exec.LookPath("aplay"); err == nil {
		return "aplay", []string{path}, nil
	}

	return "", nil, fmt.Errorf("no supported audio player found for %q", path)
}
