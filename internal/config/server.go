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
	defaultAuthUsername      = "admin"
	defaultJWTTTLMinutes     = 60
	defaultSoundsDir         = "./data/sounds"
	defaultMaxSoundUpload    = 10 << 20 // 10 MiB
	defaultLogBroadcastCap   = 2000

	envServerPort            = "PORT"
	envServerHost            = "HOST"
	envServerWebsocketSecret = "WEBSOCKET_SECRET"
	envServerWebhookSecret   = "WEBHOOK_SECRET"
	envServerTokenHeader     = "TOKEN_HEADER"
	envServerGroupPath       = "GITLAB_GROUP_PATH"

	envAuthUsername        = "AUTH_USERNAME"
	envAuthPassword        = "AUTH_PASSWORD"
	envJWTSecret           = "JWT_SECRET"
	envJWTTTLMinutes       = "JWT_TTL_MINUTES"
	envSoundsDir           = "SOUNDS_DIR"
	envMaxSoundUploadBytes = "MAX_SOUND_UPLOAD_BYTES"
	envLogBroadcastCap     = "LOG_BROADCAST_CAP"
)

// ServerConfig contains runtime settings for the app server.
type ServerConfig struct {
	Port            int    `json:"port"`
	Host            string `json:"host"`
	WebsocketSecret string `json:"websocket_secret"`
	WebhookSecret   string `json:"webhook_secret"`
	TokenHeader     string `json:"token_header"`
	GroupPath       string `json:"group_path"`

	AuthUsername        string `json:"auth_username"`
	AuthPassword        string `json:"auth_password"`
	JWTSecret           string `json:"jwt_secret"`
	JWTTTLMinutes       int    `json:"jwt_ttl_minutes"`
	SoundsDir           string `json:"sounds_dir"`
	MaxSoundUploadBytes int64  `json:"max_sound_upload_bytes"`
	LogBroadcastCap     int    `json:"log_broadcast_cap"`
}

// LoadServerConfig loads server config from config.json, then overlays environment variables.
func LoadServerConfig() (ServerConfig, error) {
	return LoadServerConfigFromFile(defaultServerConfigPath)
}

// LoadServerConfigFromFile loads server config from the path, then overlays environment variables.
// A missing file is ignored, so production can be configured through environment variables only.
func LoadServerConfigFromFile(path string) (ServerConfig, error) {
	cfg := ServerConfig{
		Port:                defaultServerPort,
		Host:                defaultServerHost,
		TokenHeader:         defaultServerTokenHeader,
		AuthUsername:        defaultAuthUsername,
		JWTTTLMinutes:       defaultJWTTTLMinutes,
		SoundsDir:           defaultSoundsDir,
		MaxSoundUploadBytes: defaultMaxSoundUpload,
		LogBroadcastCap:     defaultLogBroadcastCap,
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

	if value, ok := os.LookupEnv(envServerGroupPath); ok {
		cfg.GroupPath = value
	}

	if value, ok := os.LookupEnv(envAuthUsername); ok {
		cfg.AuthUsername = value
	}

	if value, ok := os.LookupEnv(envAuthPassword); ok {
		cfg.AuthPassword = value
	}

	if value, ok := os.LookupEnv(envJWTSecret); ok {
		cfg.JWTSecret = value
	}

	if value, ok := os.LookupEnv(envJWTTTLMinutes); ok {
		minutes, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("parse %s: %w", envJWTTTLMinutes, err)
		}
		cfg.JWTTTLMinutes = minutes
	}

	if value, ok := os.LookupEnv(envSoundsDir); ok {
		cfg.SoundsDir = value
	}

	if value, ok := os.LookupEnv(envMaxSoundUploadBytes); ok {
		n, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("parse %s: %w", envMaxSoundUploadBytes, err)
		}
		cfg.MaxSoundUploadBytes = n
	}

	if value, ok := os.LookupEnv(envLogBroadcastCap); ok {
		capacity, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("parse %s: %w", envLogBroadcastCap, err)
		}
		cfg.LogBroadcastCap = capacity
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
	if strings.TrimSpace(cfg.GroupPath) == "" {
		errs = append(errs, fmt.Errorf("group_path is required"))
	}

	if strings.TrimSpace(cfg.AuthUsername) == "" {
		errs = append(errs, fmt.Errorf("auth_username is required"))
	}
	if strings.TrimSpace(cfg.AuthPassword) == "" {
		errs = append(errs, fmt.Errorf("auth_password is required (set %s)", envAuthPassword))
	}
	if strings.TrimSpace(cfg.JWTSecret) == "" {
		errs = append(errs, fmt.Errorf("jwt_secret is required (set %s)", envJWTSecret))
	}
	if len(strings.TrimSpace(cfg.JWTSecret)) < 16 {
		errs = append(errs, fmt.Errorf("jwt_secret must be at least 16 bytes"))
	}
	if cfg.JWTTTLMinutes < 1 {
		errs = append(errs, fmt.Errorf("jwt_ttl_minutes must be at least 1"))
	}
	if strings.TrimSpace(cfg.SoundsDir) == "" {
		errs = append(errs, fmt.Errorf("sounds_dir is required"))
	}
	if cfg.MaxSoundUploadBytes < 1 {
		errs = append(errs, fmt.Errorf("max_sound_upload_bytes must be at least 1"))
	}
	if cfg.LogBroadcastCap < 1 {
		errs = append(errs, fmt.Errorf("log_broadcast_cap must be at least 1"))
	}

	return errors.Join(errs...)
}
