package web

import (
	"net/http"
	"strings"
	"time"

	"bakery/frontend/internal/application"
)

type loginData struct {
	Next string
}

func (s *server) loginPage(w http.ResponseWriter, r *http.Request) {
	if viewer, _, err := s.viewer(r); err == nil && viewer != nil {
		s.redirect(w, r, "/orders")
		return
	}
	s.render(w, r, http.StatusOK, page{
		Title: "Вход",
		View:  "login",
		Error: queryMessage(r, "error"),
		Data:  loginData{Next: safeNext(r.URL.Query().Get("next"))},
	})
}

func (s *server) login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.render(w, r, http.StatusBadRequest, page{Title: "Вход", View: "login", Error: "Не удалось прочитать форму.", Data: loginData{Next: "/orders"}})
		return
	}
	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")
	next := safeNext(r.FormValue("next"))
	if username == "" || password == "" {
		s.render(w, r, http.StatusUnprocessableEntity, page{Title: "Вход", View: "login", Error: "Введите логин и пароль.", Data: loginData{Next: next}})
		return
	}
	session, err := s.commands.Login(r.Context(), username, password, strings.TrimSpace(r.FormValue("init_data")))
	if err != nil {
		s.logger.Warn("login failed", "user", username, "error", err)
		s.render(w, r, statusOr(err, http.StatusUnauthorized), page{
			Title: "Вход",
			View:  "login",
			Error: application.MessageOf(err, "Не удалось войти. Попробуйте ещё раз."),
			Data:  loginData{Next: next},
		})
		return
	}
	expires := time.Unix(session.ExpiresAt, 0)
	s.setSession(w, r, username, application.Credentials("Bearer "+session.Token), expires)
	s.logger.Info("login", "user", username)
	s.redirect(w, r, next)
}

func (s *server) telegramLogin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	initData := strings.TrimSpace(r.FormValue("init_data"))
	if initData == "" {
		http.Error(w, "telegram initData is missing", http.StatusBadRequest)
		return
	}
	cred := application.Credentials("tma " + initData)
	viewer, err := s.queries.Me(r.Context(), cred)
	if err != nil {
		s.logger.Warn("telegram login failed", "error", err)
		http.Error(w, application.MessageOf(err, "Telegram не подтвердил вход."), statusOr(err, http.StatusUnauthorized))
		return
	}
	s.setSession(w, r, viewer.TelegramUsername, cred, time.Now().Add(24*time.Hour))
	s.logger.Info("login", "user", viewer.TelegramUsername, "via", "telegram")
	w.Header().Set("HX-Location", safeNext(r.FormValue("next")))
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) logout(w http.ResponseWriter, r *http.Request) {
	// Mini App sessions cannot log out: the identity comes from Telegram, the
	// next request would auto-login again anyway. The UI hides the button; this
	// guards direct POSTs.
	if isMiniAppSession(r) {
		s.redirect(w, r, "/me")
		return
	}
	s.logger.Info("logout", "user", orAnon(sessionUsername(r)))
	s.clearSession(w, r)
	s.redirect(w, r, "/login")
}

func (s *server) mePage(w http.ResponseWriter, r *http.Request) {
	viewer, _, ok := s.requireViewer(w, r)
	if !ok {
		return
	}
	s.render(w, r, http.StatusOK, page{Title: "Профиль", View: "me", Viewer: viewer})
}

func statusOr(err error, fallback int) int {
	if status := application.StatusOf(err); status > 0 {
		return status
	}
	return fallback
}
