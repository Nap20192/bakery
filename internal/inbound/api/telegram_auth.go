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
	"bakery/internal/pkg/authtoken"
	authuc "bakery/internal/services/auth/usecase/auth"
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

// requireMiniAppAuth authenticates every API request from either surface:
// a Telegram Mini App (initData, "tma" scheme) or the plain web client
// (bearer session token issued by /login). Both resolve to the same viewer.
func (s *Server) requireMiniAppAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.authSvc == nil || s.departmentSvc == nil || strings.TrimSpace(s.config.BotToken) == "" {
			writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "Приложение временно недоступно."})
			return
		}

		telegramID, telegramUsername, err := s.authenticateRequest(r)
		if err != nil {
			slog.WarnContext(r.Context(), "api authentication rejected", "error", err)
			writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "Не удалось подтвердить вход. Войдите заново."})
			return
		}

		viewer, apiErr := s.buildViewer(r.Context(), telegramID, telegramUsername)
		if apiErr != nil {
			writeJSON(w, apiErr.status, errorResponse{Error: apiErr.message})
			return
		}
		ctx := context.WithValue(r.Context(), miniAppUserContextKey{}, viewer)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// authenticateRequest resolves the caller's Telegram identity from either the
// Mini App initData header ("tma ...") or a web bearer session ("Bearer ...").
func (s *Server) authenticateRequest(r *http.Request) (int64, string, error) {
	scheme, data, ok := authorizationCredentials(r.Header.Get("Authorization"))
	if !ok {
		return 0, "", fmt.Errorf("authorization header is missing")
	}
	switch strings.ToLower(scheme) {
	case miniAppAuthorizationScheme:
		user, err := validateMiniAppInitData(data, s.config.BotToken, time.Now(), miniAppAuthMaxAge)
		if err != nil {
			return 0, "", err
		}
		return user.ID, strings.TrimSpace(user.Username), nil
	case "bearer":
		claims, err := authtoken.Parse(s.config.BotToken, data, time.Now())
		if err != nil {
			return 0, "", err
		}
		return claims.TelegramID, "", nil
	default:
		return 0, "", fmt.Errorf("unsupported authorization scheme %q", scheme)
	}
}

type viewerError struct {
	status  int
	message string
}

// buildViewer loads the authenticated user and their department, enforcing that
// the account exists and is attached to a supported location.
func (s *Server) buildViewer(ctx context.Context, telegramID int64, telegramUsername string) (miniAppUser, *viewerError) {
	user, err := s.authSvc.GetUserByTelegramID(ctx, telegramID)
	if err != nil {
		if errors.Is(err, authuc.ErrAuthUserNotFound) {
			return miniAppUser{}, &viewerError{http.StatusForbidden, "Сначала выберите магазин или цех в боте через /start."}
		}
		slog.ErrorContext(ctx, "api user lookup failed", "error", err)
		return miniAppUser{}, &viewerError{http.StatusInternalServerError, "Не удалось определить пользователя."}
	}
	if user.DepartmentID == nil {
		return miniAppUser{}, &viewerError{http.StatusForbidden, "Сначала выберите магазин или цех в боте через /start."}
	}
	department, err := s.departmentSvc.GetByID(ctx, *user.DepartmentID)
	if err != nil {
		slog.ErrorContext(ctx, "api department lookup failed", "error", err)
		return miniAppUser{}, &viewerError{http.StatusInternalServerError, "Не удалось определить локацию пользователя."}
	}
	if department.Type != string(app.DepartmentTypeShop) && department.Type != string(app.DepartmentTypeWorkshop) {
		return miniAppUser{}, &viewerError{http.StatusForbidden, "Выбранная локация не поддерживается в приложении."}
	}
	if telegramUsername == "" && user.TelegramUsername != nil {
		telegramUsername = strings.TrimSpace(*user.TelegramUsername)
	}
	return miniAppUser{
		Auth:           user,
		TelegramID:     telegramID,
		TelegramUser:   strings.TrimSpace(telegramUsername),
		DepartmentID:   department.ID,
		DepartmentCode: department.Code,
		DepartmentName: department.Name,
		DepartmentType: department.Type,
	}, nil
}

func miniAppUserFromContext(ctx context.Context) (miniAppUser, bool) {
	user, ok := ctx.Value(miniAppUserContextKey{}).(miniAppUser)
	return user, ok
}

func authorizationCredentials(header string) (string, string, bool) {
	parts := strings.SplitN(strings.TrimSpace(header), " ", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	data := strings.TrimSpace(parts[1])
	return parts[0], data, data != ""
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
