package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"bakery/internal/pkg/authtoken"
)

const webSessionTTL = 7 * 24 * time.Hour

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
}

// handleLogin authenticates a plain-web visitor with username + password and
// issues a bearer session token. Mini App clients do not use this — they
// authenticate per request with Telegram initData.
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if s.authSvc == nil || strings.TrimSpace(s.config.BotToken) == "" {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "Вход временно недоступен."})
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Некорректные данные входа."})
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Укажите логин и пароль."})
		return
	}

	user, err := s.authSvc.VerifyPassword(r.Context(), req.Username, req.Password)
	if err != nil {
		slog.WarnContext(r.Context(), "web login rejected", "username", req.Username, "error", err)
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "Неверный логин или пароль."})
		return
	}

	token, err := authtoken.Issue(s.config.BotToken, user.ID, webSessionTTL, time.Now())
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
