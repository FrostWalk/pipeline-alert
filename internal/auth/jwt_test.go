package auth

import (
	"errors"
	"testing"
	"time"

	"pipeline-horn/internal/config"
)

func testCfg() config.ServerConfig {
	return config.ServerConfig{
		AuthUsername:        "admin",
		AuthPassword:        "supersecretpass",
		JWTSecret:           "0123456789abcdef0123456789abcdef",
		JWTTTLMinutes:       60,
		Port:                8080,
		Host:                "127.0.0.1",
		WebsocketSecret:     "ws",
		WebhookSecret:       "wh",
		TokenHeader:         "X-Gitlab-Token",
		GroupPath:           "g",
		SoundsDir:           "./data/sounds",
		MaxSoundUploadBytes: 1024,
		LogBroadcastCap:     10,
	}
}

func TestLoginAndBearerRoundTrip(t *testing.T) {
	t.Parallel()
	j := NewJWT(testCfg())
	tok, ttl, err := j.Login("admin", "supersecretpass")
	if err != nil {
		t.Fatal(err)
	}
	if ttl != time.Hour || tok == "" {
		t.Fatalf("unexpected ttl=%v tok empty=%v", ttl, tok == "")
	}
	sub, err := j.ParseBearer("Bearer " + tok)
	if err != nil {
		t.Fatal(err)
	}
	if sub != "admin" {
		t.Fatalf("subject: %q", sub)
	}
}

func TestLoginWrongPassword(t *testing.T) {
	t.Parallel()
	j := NewJWT(testCfg())
	_, _, err := j.Login("admin", "nope")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestParseBearerInvalid(t *testing.T) {
	t.Parallel()
	j := NewJWT(testCfg())
	_, err := j.ParseBearer("Bearer garbage")
	if err == nil {
		t.Fatal("expected error")
	}
}
