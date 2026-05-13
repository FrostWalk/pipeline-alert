package auth

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"pipeline-horn/internal/config"
)

const issuer = "pipeline-horn"

// JWT issues and validates access tokens for management API.
type JWT struct {
	cfg config.ServerConfig
}

func NewJWT(cfg config.ServerConfig) *JWT {
	return &JWT{cfg: cfg}
}

// Login checks credentials and returns a signed JWT (HS256).
func (j *JWT) Login(username, password string) (token string, expiresIn time.Duration, err error) {
	u := strings.TrimSpace(username)
	p := password // do not trim password bytes
	if u == "" || p == "" {
		return "", 0, ErrEmptyLoginFields
	}

	if u != j.cfg.AuthUsername || !constantTimeStringEqual(j.cfg.AuthPassword, p) {
		return "", 0, ErrInvalidCredentials
	}

	ttl := time.Duration(j.cfg.JWTTTLMinutes) * time.Minute
	tok, err := j.sign(u, ttl)
	if err != nil {
		return "", 0, err
	}
	return tok, ttl, nil
}

var ErrInvalidCredentials = errors.New("invalid credentials")
var ErrEmptyLoginFields = errors.New("username and password required")

func constantTimeStringEqual(expected, actual string) bool {
	eb := []byte(expected)
	ab := []byte(actual)
	if len(eb) != len(ab) {
		// compare against self to keep similar timing when lengths differ
		if len(eb) > 0 {
			_ = subtle.ConstantTimeCompare(eb, eb)
		}
		return false
	}
	return subtle.ConstantTimeCompare(eb, ab) == 1
}

func (j *JWT) sign(subject string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Issuer:    issuer,
		Subject:   subject,
		IssuedAt:  jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now.Add(-1 * time.Minute)),
		ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
	}

	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString([]byte(j.cfg.JWTSecret))
}

// ParseBearer validates Authorization: Bearer token and returns subject.
func (j *JWT) ParseBearer(header string) (subject string, err error) {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return "", fmt.Errorf("missing bearer token")
	}
	raw := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	if raw == "" {
		return "", fmt.Errorf("empty bearer token")
	}

	claims := &jwt.RegisteredClaims{}
	_, err = jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method %v", token.Header["alg"])
		}
		return []byte(j.cfg.JWTSecret), nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(issuer),
		jwt.WithExpirationRequired(),
		jwt.WithLeeway(5*time.Second),
	)
	if err != nil {
		return "", fmt.Errorf("invalid token")
	}

	if claims.Subject == "" {
		return "", fmt.Errorf("invalid token")
	}
	return claims.Subject, nil
}
