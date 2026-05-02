package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	defaultServerConfigPath = "config.json"

	defaultServerPort        = 8080
	defaultServerHost        = "0.0.0.0"
	defaultServerTokenHeader = "X-Gitlab-Token"

	envServerPort            = "PORT"
	envServerHost            = "HOST"
	envServerWebsocketSecret = "WEBSOCKET_SECRET"
	envServerWebhookSecret   = "WEBHOOK_SECRET"
	envServerTokenHeader     = "TOKEN_HEADER"
)

// ServerConfig contains runtime settings for the app server.
type ServerConfig struct {
	Port            int    `json:"port"`
	Host            string `json:"host"`
	WebsocketSecret string `json:"websocket_secret"`
	WebhookSecret   string `json:"webhook_secret"`
	TokenHeader     string `json:"token_header"`
}

// LoadServerConfig loads server config from config.json, then overlays environment variables.
func LoadServerConfig() (ServerConfig, error) {
	return LoadServerConfigFromFile(defaultServerConfigPath)
}

// LoadServerConfigFromFile loads server config from the path, then overlays environment variables.
// A missing file is ignored, so production can be configured through environment variables only.
func LoadServerConfigFromFile(path string) (ServerConfig, error) {
	cfg := ServerConfig{
		Port:        defaultServerPort,
		Host:        defaultServerHost,
		TokenHeader: defaultServerTokenHeader,
	}

	if err := loadServerConfigFile(path, &cfg); err != nil {
		return ServerConfig{}, err
	}

	if err := applyServerEnv(&cfg); err != nil {
		return ServerConfig{}, err
	}

	if err := validateServerConfig(cfg); err != nil {
		return ServerConfig{}, err
	}

	return cfg, nil
}

func loadServerConfigFile(path string, cfg *ServerConfig) error {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("open server config %q: %w", path, err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(cfg); err != nil {
		return fmt.Errorf("decode server config %q: %w", path, err)
	}

	return nil
}

func applyServerEnv(cfg *ServerConfig) error {
	if value, ok := os.LookupEnv(envServerPort); ok {
		port, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("parse %s: %w", envServerPort, err)
		}
		cfg.Port = port
	}

	if value, ok := os.LookupEnv(envServerHost); ok {
		cfg.Host = value
	}

	if value, ok := os.LookupEnv(envServerWebsocketSecret); ok {
		cfg.WebsocketSecret = value
	}

	if value, ok := os.LookupEnv(envServerWebhookSecret); ok {
		cfg.WebhookSecret = value
	}

	if value, ok := os.LookupEnv(envServerTokenHeader); ok {
		cfg.TokenHeader = value
	}

	return nil
}

func validateServerConfig(cfg ServerConfig) error {
	var errs []error

	if cfg.Port < 1 || cfg.Port > 65535 {
		errs = append(errs, fmt.Errorf("port must be between 1 and 65535"))
	}
	if strings.TrimSpace(cfg.Host) == "" {
		errs = append(errs, fmt.Errorf("host is required"))
	}
	if strings.TrimSpace(cfg.WebsocketSecret) == "" {
		errs = append(errs, fmt.Errorf("websocket_secret is required"))
	}
	if strings.TrimSpace(cfg.WebhookSecret) == "" {
		errs = append(errs, fmt.Errorf("webhook_secret is required"))
	}
	if strings.TrimSpace(cfg.TokenHeader) == "" {
		errs = append(errs, fmt.Errorf("token_header is required"))
	}

	return errors.Join(errs...)
}
