// Package authhttp is the HTTP delivery adapter of the auth service: web
// password login and the current-user profile endpoint.
package authhttp

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"bakery/internal/inbound/api/contract"
	"bakery/internal/inbound/api/httpx"
	"bakery/internal/pkg/authtoken"
	authuc "bakery/internal/services/auth/usecase/auth"
	departmentuc "bakery/internal/services/department/usecase/department"
)

const webSessionTTL = 7 * 24 * time.Hour

// Handler is the auth service HTTP delivery adapter.
type Handler struct {
	authSvc       authuc.UseCase
	departmentSvc departmentuc.UseCase
	botToken      string
}

func New(authSvc authuc.UseCase, departmentSvc departmentuc.UseCase, botToken string) *Handler {
	return &Handler{authSvc: authSvc, departmentSvc: departmentSvc, botToken: botToken}
}

// RegisterRoutes wires login (public) and the profile endpoint (authenticated).
func (h *Handler) RegisterRoutes(mux *http.ServeMux, auth *httpx.Authenticator) {
	mux.HandleFunc("POST /login", h.handleLogin)
	mux.Handle("GET /me", auth.RequireAuth(http.HandlerFunc(h.handleMe)))
}

// handleLogin authenticates a plain-web visitor with username + password and
// issues a bearer session token. Mini App clients do not use this — they
// authenticate per request with Telegram initData.
func (h *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	if h.authSvc == nil || strings.TrimSpace(h.botToken) == "" {
		httpx.WriteError(w, http.StatusServiceUnavailable, "Вход временно недоступен.")
		return
	}

	var req contract.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Некорректные данные входа.")
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || req.Password == "" {
		httpx.WriteError(w, http.StatusBadRequest, "Укажите логин и пароль.")
		return
	}

	user, err := h.authSvc.VerifyPassword(r.Context(), req.Username, req.Password)
	if err != nil {
		slog.WarnContext(r.Context(), "web login rejected", "username", req.Username, "error", err)
		httpx.WriteError(w, http.StatusUnauthorized, "Неверный логин или пароль.")
		return
	}

	token, err := authtoken.Issue(h.botToken, user.ID, webSessionTTL, time.Now())
	if err != nil {
		slog.ErrorContext(r.Context(), "issue session token failed", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "Не удалось создать сессию.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, contract.LoginResponse{
		Token:     token,
		ExpiresAt: time.Now().Add(webSessionTTL).Unix(),
	})
}

func (h *Handler) handleMe(w http.ResponseWriter, r *http.Request) {
	viewer, _ := httpx.MiniAppUserFromContext(r.Context())
	resp := contract.Me{
		Role:             viewer.Auth.Role,
		TelegramUsername: httpx.TelegramUsernameOf(viewer.Auth),
	}
	if viewer.Auth.TelegramID != nil {
		resp.TelegramID = *viewer.Auth.TelegramID
	}
	// Department is optional (admins may have none). Resolve it when present.
	if viewer.Auth.DepartmentID != nil && h.departmentSvc != nil {
		if department, err := h.departmentSvc.GetByID(r.Context(), *viewer.Auth.DepartmentID); err == nil {
			resp.DepartmentID = department.ID
			resp.DepartmentCode = department.Code
			resp.DepartmentName = department.Name
			resp.DepartmentType = department.Type
		}
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
}
