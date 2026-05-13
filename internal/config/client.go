package config

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultClientServerPort = 443
	defaultClientSoundPath  = "/usr/share/pipeline-horn-client/horn.mp3"
	defaultClientSoundDir   = "/var/lib/pipeline-horn-client/sounds"
)

type ClientConfig struct {
	ServerURL        string
	ServerPort       int
	AcceptInvalidTLS bool
	WebsocketSecret  string
	SoundPath        string
	SoundDir         string
}

func LoadClientConfigFromArgs(args []string) (ClientConfig, error) {
	cfg := ClientConfig{}

	flags := flag.NewFlagSet("pipeline-alert-client", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(&cfg.ServerURL, "server_url", "", "server URL")
	flags.IntVar(&cfg.ServerPort, "server_port", defaultClientServerPort, "server port")
	flags.BoolVar(&cfg.AcceptInvalidTLS, "accept_invalid_tls", false, "accept invalid TLS certificates")
	flags.StringVar(&cfg.WebsocketSecret, "websocket_secret", "", "websocket secret")
	flags.StringVar(&cfg.SoundPath, "sound_path", defaultClientSoundPath, "default sound file when nothing selected")
	flags.StringVar(&cfg.SoundDir, "sound_dir", defaultClientSoundDir, "directory for synced sounds and selection state")

	if err := flags.Parse(args); err != nil {
		return ClientConfig{}, fmt.Errorf("parse client flags: %w", err)
	}

	if err := validateClientConfig(cfg); err != nil {
		return ClientConfig{}, err
	}

	return cfg, nil
}

func validateClientConfig(cfg ClientConfig) error {
	var errs []error

	if strings.TrimSpace(cfg.ServerURL) == "" {
		errs = append(errs, fmt.Errorf("server_url is required"))
	}
	if cfg.ServerPort < 1 || cfg.ServerPort > 65535 {
		errs = append(errs, fmt.Errorf("server_port must be between 1 and 65535"))
	}
	if strings.TrimSpace(cfg.WebsocketSecret) == "" {
		errs = append(errs, fmt.Errorf("websocket_secret is required"))
	}
	if strings.TrimSpace(cfg.SoundPath) == "" {
		errs = append(errs, fmt.Errorf("sound_path is required"))
	} else if err := validateSoundPath(cfg.SoundPath); err != nil {
		errs = append(errs, err)
	}
	if strings.TrimSpace(cfg.SoundDir) == "" {
		errs = append(errs, fmt.Errorf("sound_dir is required"))
	}

	return errors.Join(errs...)
}

func validateSoundPath(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("sound_path %q is not usable: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("sound_path %q is not a regular file", path)
	}
	return nil
}

// EnsureSoundDir creates the sound directory tree.
func EnsureSoundDir(dir string) error {
	d := filepath.Clean(strings.TrimSpace(dir))
	if d == "" || d == "." {
		return fmt.Errorf("invalid sound_dir")
	}
	return os.MkdirAll(d, 0o750)
}
