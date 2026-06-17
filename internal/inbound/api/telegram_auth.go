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

	accessdomain "bakery/internal/domain/access"
	"bakery/internal/pkg/authtoken"
	"bakery/internal/pkg/enum"
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
// requireMiniAppAuth authenticates a request (Mini App initData or web bearer)
// and requires the user to be attached to a supported department.
func (s *Server) requireMiniAppAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.authSvc == nil || s.departmentSvc == nil || strings.TrimSpace(s.config.BotToken) == "" {
			writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "Приложение временно недоступно."})
			return
		}
		user, apiErr := s.resolveUser(r.Context(), r)
		if apiErr != nil {
			writeJSON(w, apiErr.status, errorResponse{Error: apiErr.message})
			return
		}
		viewer, apiErr := s.buildDepartmentViewer(r.Context(), user)
		if apiErr != nil {
			writeJSON(w, apiErr.status, errorResponse{Error: apiErr.message})
			return
		}
		ctx := context.WithValue(r.Context(), miniAppUserContextKey{}, viewer)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// requireAuth authenticates a request without requiring a department. Used by
// endpoints that work for any logged-in user, including admins with no location.
func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.authSvc == nil || strings.TrimSpace(s.config.BotToken) == "" {
			writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "Приложение временно недоступно."})
			return
		}
		user, apiErr := s.resolveUser(r.Context(), r)
		if apiErr != nil {
			writeJSON(w, apiErr.status, errorResponse{Error: apiErr.message})
			return
		}
		ctx := context.WithValue(r.Context(), miniAppUserContextKey{}, miniAppUser{Auth: user})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// requireAdmin authenticates a request and requires the admin role. Used for
// user management; no department is required.
func (s *Server) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.authSvc == nil || strings.TrimSpace(s.config.BotToken) == "" {
			writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "Приложение временно недоступно."})
			return
		}
		user, apiErr := s.resolveUser(r.Context(), r)
		if apiErr != nil {
			writeJSON(w, apiErr.status, errorResponse{Error: apiErr.message})
			return
		}
		if accessdomain.NormalizeRole(user.Role) != accessdomain.RoleAdmin {
			writeJSON(w, http.StatusForbidden, errorResponse{Error: "Доступ только для администратора."})
			return
		}
		ctx := context.WithValue(r.Context(), miniAppUserContextKey{}, miniAppUser{Auth: user})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type viewerError struct {
	status  int
	message string
}

// resolveUser authenticates the caller from either the Mini App initData header
// ("tma ...") or a web bearer session ("Bearer ...") and loads the user.
func (s *Server) resolveUser(ctx context.Context, r *http.Request) (accessdomain.AuthUser, *viewerError) {
	scheme, data, ok := authorizationCredentials(r.Header.Get("Authorization"))
	if !ok {
		return accessdomain.AuthUser{}, &viewerError{http.StatusUnauthorized, "Войдите, чтобы продолжить."}
	}
	switch strings.ToLower(scheme) {
	case miniAppAuthorizationScheme:
		init, err := validateMiniAppInitData(data, s.config.BotToken, time.Now(), miniAppAuthMaxAge)
		if err != nil {
			slog.WarnContext(ctx, "mini app auth rejected", "error", err)
			return accessdomain.AuthUser{}, &viewerError{http.StatusUnauthorized, "Не удалось подтвердить вход. Откройте приложение заново."}
		}
		return s.lookupUser(ctx, func() (accessdomain.AuthUser, error) {
			return s.authSvc.GetUserByTelegramUsername(ctx, strings.TrimSpace(init.Username))
		})
	case "bearer":
		claims, err := authtoken.Parse(s.config.BotToken, data, time.Now())
		if err != nil {
			slog.WarnContext(ctx, "bearer auth rejected", "error", err)
			return accessdomain.AuthUser{}, &viewerError{http.StatusUnauthorized, "Сессия истекла. Войдите заново."}
		}
		return s.lookupUser(ctx, func() (accessdomain.AuthUser, error) {
			return s.authSvc.GetUserByID(ctx, claims.UserID)
		})
	default:
		return accessdomain.AuthUser{}, &viewerError{http.StatusUnauthorized, "Неподдерживаемый способ авторизации."}
	}
}

func (s *Server) lookupUser(ctx context.Context, load func() (accessdomain.AuthUser, error)) (accessdomain.AuthUser, *viewerError) {
	user, err := load()
	if err != nil {
		if errors.Is(err, authuc.ErrAuthUserNotFound) {
			return accessdomain.AuthUser{}, &viewerError{http.StatusForbidden, "Пользователь не найден."}
		}
		slog.ErrorContext(ctx, "api user lookup failed", "error", err)
		return accessdomain.AuthUser{}, &viewerError{http.StatusInternalServerError, "Не удалось определить пользователя."}
	}
	return user, nil
}

// buildDepartmentViewer enriches an authenticated user with their department,
// requiring a supported location.
func (s *Server) buildDepartmentViewer(ctx context.Context, user accessdomain.AuthUser) (miniAppUser, *viewerError) {
	if user.DepartmentID == nil {
		return miniAppUser{}, &viewerError{http.StatusForbidden, "Сначала выберите магазин или цех в боте через /start."}
	}
	department, err := s.departmentSvc.GetByID(ctx, *user.DepartmentID)
	if err != nil {
		slog.ErrorContext(ctx, "api department lookup failed", "error", err)
		return miniAppUser{}, &viewerError{http.StatusInternalServerError, "Не удалось определить локацию пользователя."}
	}
	if department.Type != string(enum.DepartmentTypeShop) && department.Type != string(enum.DepartmentTypeWorkshop) {
		return miniAppUser{}, &viewerError{http.StatusForbidden, "Выбранная локация не поддерживается в приложении."}
	}
	telegramID := int64(0)
	if user.TelegramID != nil {
		telegramID = *user.TelegramID
	}
	telegramUsername := ""
	if user.TelegramUsername != nil {
		telegramUsername = strings.TrimSpace(*user.TelegramUsername)
	}
	return miniAppUser{
		Auth:           user,
		TelegramID:     telegramID,
		TelegramUser:   telegramUsername,
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
