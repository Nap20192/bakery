// Package authtoken issues and verifies the compact HMAC-signed bearer tokens
// used to authenticate API callers outside Telegram.
//
// The token format is "v1.<base64url(payload)>.<base64url(hmac)>". It is shared
// by the API server (which verifies tokens) and the Telegram bot, which — once
// it runs as a separate process talking to the API over HTTP — mints a token
// per Telegram user using the same bot-token secret. Keeping it in one package
// lets both sides agree on the algorithm without importing each other.
package authtoken

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const version = "v1"

// Claims is the verified payload of a session token.
type Claims struct {
	TelegramID int64 `json:"tid"`
	ExpiresAt  int64 `json:"exp"`
}

// Issue creates a signed token for the given Telegram user.
func Issue(secret string, telegramID int64, ttl time.Duration, now time.Time) (string, error) {
	if strings.TrimSpace(secret) == "" {
		return "", fmt.Errorf("session secret is empty")
	}
	claims := Claims{TelegramID: telegramID, ExpiresAt: now.Add(ttl).Unix()}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("marshal session claims: %w", err)
	}
	body := base64.RawURLEncoding.EncodeToString(payload)
	return strings.Join([]string{version, body, sign(secret, body)}, "."), nil
}

// Parse verifies a token's signature and expiry and returns its claims.
func Parse(secret, token string, now time.Time) (Claims, error) {
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) != 3 || parts[0] != version {
		return Claims{}, fmt.Errorf("malformed session token")
	}
	if !hmac.Equal([]byte(sign(secret, parts[1])), []byte(parts[2])) {
		return Claims{}, fmt.Errorf("invalid session token signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Claims{}, fmt.Errorf("decode session token: %w", err)
	}
	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return Claims{}, fmt.Errorf("decode session claims: %w", err)
	}
	if claims.TelegramID == 0 {
		return Claims{}, fmt.Errorf("session token has no subject")
	}
	if claims.ExpiresAt <= now.Unix() {
		return Claims{}, fmt.Errorf("session token expired")
	}
	return claims, nil
}

func sign(secret, body string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(body))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
