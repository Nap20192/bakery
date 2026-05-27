package api

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"bakery/internal/app"
	accessdomain "bakery/internal/domain/access"
)

const (
	miniAppAuthorizationScheme = "tma"
	miniAppAuthMaxAge          = 24 * time.Hour
)

type miniAppUserContextKey struct{}

type miniAppUser struct {
	Auth           accessdomain.AuthUser
	TelegramID     int64
	TelegramUser   string
	DepartmentID   int64
	DepartmentCode string
	DepartmentName string
	DepartmentType string
}

type telegramInitUser struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

func (s *Server) requireMiniAppAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.authSvc == nil || s.departmentSvc == nil || strings.TrimSpace(s.config.BotToken) == "" {
			writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "Mini App временно недоступен."})
			return
		}

		initData, ok := miniAppAuthorizationData(r.Header.Get("Authorization"))
		if !ok {
			writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "Откройте приложение через кнопку в Telegram."})
			return
		}
		telegramUser, err := validateMiniAppInitData(initData, s.config.BotToken, time.Now(), miniAppAuthMaxAge)
		if err != nil {
			slog.WarnContext(r.Context(), "telegram mini app authentication rejected", "error", err)
			writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "Не удалось подтвердить вход через Telegram. Откройте приложение заново."})
			return
		}

		user, err := s.authSvc.GetUserByTelegramID(r.Context(), telegramUser.ID)
		if err != nil {
			if errors.Is(err, app.ErrAuthUserNotFound) {
				writeJSON(w, http.StatusForbidden, errorResponse{Error: "Сначала выберите магазин или цех в боте через /start."})
				return
			}
			slog.ErrorContext(r.Context(), "mini app user lookup failed", "error", err)
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "Не удалось определить пользователя."})
			return
		}
		if user.DepartmentID == nil {
			writeJSON(w, http.StatusForbidden, errorResponse{Error: "Сначала выберите магазин или цех в боте через /start."})
			return
		}
		department, err := s.departmentSvc.GetByID(r.Context(), *user.DepartmentID)
		if err != nil {
			slog.ErrorContext(r.Context(), "mini app department lookup failed", "error", err)
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "Не удалось определить локацию пользователя."})
			return
		}
		if department.Type != string(app.DepartmentTypeShop) && department.Type != string(app.DepartmentTypeWorkshop) {
			writeJSON(w, http.StatusForbidden, errorResponse{Error: "Выбранная локация не поддерживается в приложении."})
			return
		}

		viewer := miniAppUser{
			Auth:           user,
			TelegramID:     telegramUser.ID,
			TelegramUser:   strings.TrimSpace(telegramUser.Username),
			DepartmentID:   department.ID,
			DepartmentCode: department.Code,
			DepartmentName: department.Name,
			DepartmentType: department.Type,
		}
		ctx := context.WithValue(r.Context(), miniAppUserContextKey{}, viewer)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func miniAppUserFromContext(ctx context.Context) (miniAppUser, bool) {
	user, ok := ctx.Value(miniAppUserContextKey{}).(miniAppUser)
	return user, ok
}

func miniAppAuthorizationData(header string) (string, bool) {
	parts := strings.SplitN(strings.TrimSpace(header), " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], miniAppAuthorizationScheme) {
		return "", false
	}
	data := strings.TrimSpace(parts[1])
	return data, data != ""
}

func validateMiniAppInitData(raw string, botToken string, now time.Time, maxAge time.Duration) (telegramInitUser, error) {
	params, err := url.ParseQuery(raw)
	if err != nil {
		return telegramInitUser{}, fmt.Errorf("parse init data: %w", err)
	}
	hashValue := strings.TrimSpace(params.Get("hash"))
	if hashValue == "" {
		return telegramInitUser{}, fmt.Errorf("init data hash is missing")
	}
	receivedHash, err := hex.DecodeString(hashValue)
	if err != nil {
		return telegramInitUser{}, fmt.Errorf("decode init data hash: %w", err)
	}

	keys := make([]string, 0, len(params))
	for key := range params {
		if key != "hash" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	checkLines := make([]string, 0, len(keys))
	for _, key := range keys {
		checkLines = append(checkLines, key+"="+params.Get(key))
	}
	dataCheckString := strings.Join(checkLines, "\n")

	secretMAC := hmac.New(sha256.New, []byte("WebAppData"))
	_, _ = secretMAC.Write([]byte(botToken))
	signatureMAC := hmac.New(sha256.New, secretMAC.Sum(nil))
	_, _ = signatureMAC.Write([]byte(dataCheckString))
	if !hmac.Equal(receivedHash, signatureMAC.Sum(nil)) {
		return telegramInitUser{}, fmt.Errorf("init data signature is invalid")
	}

	authDate, err := strconv.ParseInt(params.Get("auth_date"), 10, 64)
	if err != nil {
		return telegramInitUser{}, fmt.Errorf("parse auth date: %w", err)
	}
	authTime := time.Unix(authDate, 0)
	if authTime.After(now.Add(time.Minute)) || now.Sub(authTime) > maxAge {
		return telegramInitUser{}, fmt.Errorf("init data has expired")
	}

	var user telegramInitUser
	if err := json.Unmarshal([]byte(params.Get("user")), &user); err != nil {
		return telegramInitUser{}, fmt.Errorf("parse telegram user: %w", err)
	}
	if user.ID <= 0 {
		return telegramInitUser{}, fmt.Errorf("telegram user ID is missing")
	}
	return user, nil
}
