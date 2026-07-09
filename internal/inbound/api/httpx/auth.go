package httpx

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

	"bakery/internal/pkg/authtoken"
	authdomain "bakery/internal/services/auth/domain"
	authuc "bakery/internal/services/auth/usecase/auth"
	departmentuc "bakery/internal/services/department/usecase/department"
)

const (
	miniAppAuthorizationScheme = "tma"
	miniAppAuthMaxAge          = 24 * time.Hour
)

type miniAppUserContextKey struct{}

// MiniAppUser is the authenticated viewer attached to a request context,
// optionally enriched with the user's department.
type MiniAppUser struct {
	Auth           authdomain.AuthUser
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

// Authenticator resolves the caller (Telegram Mini App initData or web bearer)
// and exposes the auth middlewares shared by every delivery adapter.
type Authenticator struct {
	authSvc       authuc.UseCase
	departmentSvc departmentuc.UseCase
	botToken      string
}

func NewAuthenticator(authSvc authuc.UseCase, departmentSvc departmentuc.UseCase, botToken string) *Authenticator {
	return &Authenticator{authSvc: authSvc, departmentSvc: departmentSvc, botToken: botToken}
}

func (a *Authenticator) ready() bool {
	return a.authSvc != nil && strings.TrimSpace(a.botToken) != ""
}

// RequireMiniAppAuth authenticates a request (Mini App initData or web bearer)
// and requires the user to be attached to a supported department.
func (a *Authenticator) RequireMiniAppAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !a.ready() || a.departmentSvc == nil {
			WriteError(w, http.StatusServiceUnavailable, "Приложение временно недоступно.")
			return
		}
		user, apiErr := a.resolveUser(r.Context(), r)
		if apiErr != nil {
			WriteError(w, apiErr.status, apiErr.message)
			return
		}
		viewer, apiErr := a.buildDepartmentViewer(r.Context(), user)
		if apiErr != nil {
			WriteError(w, apiErr.status, apiErr.message)
			return
		}
		next.ServeHTTP(w, r.WithContext(WithMiniAppUser(r.Context(), viewer)))
	})
}

// RequireAuth authenticates a request without requiring a department. Used by
// endpoints that work for any logged-in user, including admins with no location.
func (a *Authenticator) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !a.ready() {
			WriteError(w, http.StatusServiceUnavailable, "Приложение временно недоступно.")
			return
		}
		user, apiErr := a.resolveUser(r.Context(), r)
		if apiErr != nil {
			WriteError(w, apiErr.status, apiErr.message)
			return
		}
		next.ServeHTTP(w, r.WithContext(WithMiniAppUser(r.Context(), MiniAppUser{Auth: user})))
	})
}

// RequireAdmin authenticates a request and requires the admin role.
func (a *Authenticator) RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !a.ready() {
			WriteError(w, http.StatusServiceUnavailable, "Приложение временно недоступно.")
			return
		}
		user, apiErr := a.resolveUser(r.Context(), r)
		if apiErr != nil {
			WriteError(w, apiErr.status, apiErr.message)
			return
		}
		if authdomain.NormalizeRole(user.Role) != authdomain.RoleAdmin {
			WriteError(w, http.StatusForbidden, "Доступ только для администратора.")
			return
		}
		next.ServeHTTP(w, r.WithContext(WithMiniAppUser(r.Context(), MiniAppUser{Auth: user})))
	})
}

type viewerError struct {
	status  int
	message string
}

func (a *Authenticator) resolveUser(ctx context.Context, r *http.Request) (authdomain.AuthUser, *viewerError) {
	scheme, data, ok := authorizationCredentials(r.Header.Get("Authorization"))
	if !ok {
		return authdomain.AuthUser{}, &viewerError{http.StatusUnauthorized, "Войдите, чтобы продолжить."}
	}
	switch strings.ToLower(scheme) {
	case miniAppAuthorizationScheme:
		init, err := validateMiniAppInitData(data, a.botToken, time.Now(), miniAppAuthMaxAge)
		if err != nil {
			slog.WarnContext(ctx, "mini app auth rejected", "error", err)
			return authdomain.AuthUser{}, &viewerError{http.StatusUnauthorized, "Не удалось подтвердить вход. Откройте приложение заново."}
		}
		return a.lookupUser(ctx, func() (authdomain.AuthUser, error) {
			return a.authSvc.GetUserByTelegramUsername(ctx, strings.TrimSpace(init.Username))
		})
	case "bearer":
		claims, err := authtoken.Parse(a.botToken, data, time.Now())
		if err != nil {
			slog.WarnContext(ctx, "bearer auth rejected", "error", err)
			return authdomain.AuthUser{}, &viewerError{http.StatusUnauthorized, "Сессия истекла. Войдите заново."}
		}
		return a.lookupUser(ctx, func() (authdomain.AuthUser, error) {
			return a.authSvc.GetUserByID(ctx, claims.UserID)
		})
	default:
		return authdomain.AuthUser{}, &viewerError{http.StatusUnauthorized, "Неподдерживаемый способ авторизации."}
	}
}

func (a *Authenticator) lookupUser(ctx context.Context, load func() (authdomain.AuthUser, error)) (authdomain.AuthUser, *viewerError) {
	user, err := load()
	if err != nil {
		if errors.Is(err, authuc.ErrAuthUserNotFound) {
			return authdomain.AuthUser{}, &viewerError{http.StatusForbidden, "Пользователь не найден."}
		}
		slog.ErrorContext(ctx, "api user lookup failed", "error", err)
		return authdomain.AuthUser{}, &viewerError{http.StatusInternalServerError, "Не удалось определить пользователя."}
	}
	return user, nil
}

func (a *Authenticator) buildDepartmentViewer(ctx context.Context, user authdomain.AuthUser) (MiniAppUser, *viewerError) {
	// Access is role-driven: users are no longer bound to a single department.
	role := authdomain.NormalizeRole(user.Role)
	if role != authdomain.RoleAdmin && role != authdomain.RoleShop && role != authdomain.RoleBaker {
		return MiniAppUser{}, &viewerError{http.StatusForbidden, "Ваша роль не поддерживается в приложении."}
	}
	telegramID := int64(0)
	if user.TelegramID != nil {
		telegramID = *user.TelegramID
	}
	telegramUsername := ""
	if user.TelegramUsername != nil {
		telegramUsername = strings.TrimSpace(*user.TelegramUsername)
	}
	viewer := MiniAppUser{
		Auth:         user,
		TelegramID:   telegramID,
		TelegramUser: telegramUsername,
	}
	// A department binding is optional now; populate it for display when present.
	if user.DepartmentID != nil {
		if department, err := a.departmentSvc.GetByID(ctx, *user.DepartmentID); err != nil {
			slog.WarnContext(ctx, "api department lookup failed", "error", err)
		} else {
			viewer.DepartmentID = department.ID
			viewer.DepartmentCode = department.Code
			viewer.DepartmentName = department.Name
			viewer.DepartmentType = department.Type
		}
	}
	return viewer, nil
}

// WithMiniAppUser attaches an authenticated viewer to the context. Used by the
// auth middlewares and by tests.
func WithMiniAppUser(ctx context.Context, user MiniAppUser) context.Context {
	return context.WithValue(ctx, miniAppUserContextKey{}, user)
}

// MiniAppUserFromContext returns the authenticated viewer attached by an auth
// middleware.
func MiniAppUserFromContext(ctx context.Context) (MiniAppUser, bool) {
	user, ok := ctx.Value(miniAppUserContextKey{}).(MiniAppUser)
	return user, ok
}

// IsShopUser reports whether the viewer may create/edit orders (shop or admin).
func IsShopUser(user MiniAppUser) bool {
	role := authdomain.NormalizeRole(user.Auth.Role)
	return role == authdomain.RoleShop || role == authdomain.RoleAdmin
}

// TelegramUsernameOf returns the trimmed telegram username, or "".
func TelegramUsernameOf(u authdomain.AuthUser) string {
	if u.TelegramUsername != nil {
		return strings.TrimSpace(*u.TelegramUsername)
	}
	return ""
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
