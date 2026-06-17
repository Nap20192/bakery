package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"bakery/internal/pkg/authtoken"
)

const (
	telegramLoginMaxAge = 24 * time.Hour
	webSessionTTL       = 7 * 24 * time.Hour
)

type loginResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
}

// handleTelegramLogin authenticates a plain-web visitor via the Telegram Login
// Widget. The widget returns Telegram-signed fields; we verify the signature
// with the bot token, resolve the existing user, and issue a bearer session
// token the web client uses for subsequent requests.
func (s *Server) handleTelegramLogin(w http.ResponseWriter, r *http.Request) {
	if s.authSvc == nil || s.departmentSvc == nil || strings.TrimSpace(s.config.BotToken) == "" {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "Вход через Telegram временно недоступен."})
		return
	}

	var fields map[string]any
	decoder := json.NewDecoder(r.Body)
	decoder.UseNumber()
	if err := decoder.Decode(&fields); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Некорректные данные входа."})
		return
	}

	telegramID, telegramUsername, err := validateTelegramLogin(fields, s.config.BotToken, time.Now(), telegramLoginMaxAge)
	if err != nil {
		slog.WarnContext(r.Context(), "telegram login rejected", "error", err)
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "Не удалось подтвердить вход через Telegram."})
		return
	}

	if _, apiErr := s.buildViewer(r.Context(), telegramID, telegramUsername); apiErr != nil {
		writeJSON(w, apiErr.status, errorResponse{Error: apiErr.message})
		return
	}

	token, err := authtoken.Issue(s.config.BotToken, telegramID, webSessionTTL, time.Now())
	if err != nil {
		slog.ErrorContext(r.Context(), "issue session token failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "Не удалось создать сессию."})
		return
	}
	writeJSON(w, http.StatusOK, loginResponse{
		Token:     token,
		ExpiresAt: time.Now().Add(webSessionTTL).Unix(),
	})
}

// validateTelegramLogin verifies the Telegram Login Widget signature.
// The check string is built from all received fields except "hash", and the
// secret key is SHA256(bot_token) (this differs from Mini App initData, which
// keys with HMAC(bot_token, "WebAppData")).
func validateTelegramLogin(fields map[string]any, botToken string, now time.Time, maxAge time.Duration) (int64, string, error) {
	hashValue, _ := fields["hash"].(string)
	hashValue = strings.TrimSpace(hashValue)
	if hashValue == "" {
		return 0, "", fmt.Errorf("login hash is missing")
	}
	receivedHash, err := hex.DecodeString(hashValue)
	if err != nil {
		return 0, "", fmt.Errorf("decode login hash: %w", err)
	}

	keys := make([]string, 0, len(fields))
	for key := range fields {
		if key != "hash" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		value, ok := stringifyLoginField(fields[key])
		if !ok {
			continue
		}
		lines = append(lines, key+"="+value)
	}
	dataCheckString := strings.Join(lines, "\n")

	secret := sha256.Sum256([]byte(botToken))
	mac := hmac.New(sha256.New, secret[:])
	_, _ = mac.Write([]byte(dataCheckString))
	if !hmac.Equal(receivedHash, mac.Sum(nil)) {
		return 0, "", fmt.Errorf("login signature is invalid")
	}

	authDate, err := loginFieldInt(fields, "auth_date")
	if err != nil {
		return 0, "", fmt.Errorf("parse auth_date: %w", err)
	}
	authTime := time.Unix(authDate, 0)
	if authTime.After(now.Add(time.Minute)) || now.Sub(authTime) > maxAge {
		return 0, "", fmt.Errorf("login data has expired")
	}

	telegramID, err := loginFieldInt(fields, "id")
	if err != nil || telegramID <= 0 {
		return 0, "", fmt.Errorf("telegram user id is missing")
	}
	username, _ := stringifyLoginField(fields["username"])
	return telegramID, strings.TrimSpace(username), nil
}

func stringifyLoginField(value any) (string, bool) {
	switch typed := value.(type) {
	case string:
		return typed, true
	case json.Number:
		return typed.String(), true
	case bool:
		return strconv.FormatBool(typed), true
	case nil:
		return "", false
	default:
		return fmt.Sprintf("%v", typed), true
	}
}

func loginFieldInt(fields map[string]any, key string) (int64, error) {
	value, ok := stringifyLoginField(fields[key])
	if !ok || value == "" {
		return 0, fmt.Errorf("%s is missing", key)
	}
	return strconv.ParseInt(value, 10, 64)
}
