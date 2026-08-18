package web

import (
	"crypto/rand"
	"embed"
	"encoding/base64"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"bakery/frontend/internal/application"
	"bakery/internal/inbound/api/contract"
)

const (
	sessionCookie = "bakery_session"
	csrfCookie    = "bakery_csrf"
)

//go:embed templates static
var frontendFiles embed.FS

type server struct {
	queries   application.Queries
	commands  application.Commands
	logger    *slog.Logger
	templates *template.Template
	static    http.Handler
}

type page struct {
	Title       string
	View        string
	CurrentPath string
	Today       string
	Viewer      *contract.Me
	CSRF        string
	Error       string
	Success     string
	MiniApp     bool
	Data        any
}

func New(queries application.Queries, commands application.Commands, logger *slog.Logger) (http.Handler, error) {
	server, err := newServer(queries, commands, logger)
	if err != nil {
		return nil, err
	}
	return server.routes(), nil
}

func newServer(queries application.Queries, commands application.Commands, logger *slog.Logger) (*server, error) {
	templates, err := parseTemplates()
	if err != nil {
		return nil, err
	}
	staticFS, err := fs.Sub(frontendFiles, "static")
	if err != nil {
		return nil, fmt.Errorf("open static filesystem: %w", err)
	}
	return &server{
		queries:   queries,
		commands:  commands,
		logger:    logger,
		templates: templates,
		static:    http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))),
	}, nil
}

func (s *server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /static/", s.static)
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("GET /api/health", s.health)
	mux.HandleFunc("GET /login", s.loginPage)
	mux.HandleFunc("POST /session/login", s.login)
	mux.HandleFunc("POST /session/telegram", s.telegramLogin)
	mux.HandleFunc("POST /session/logout", s.logout)

	mux.HandleFunc("GET /", s.home)
	mux.HandleFunc("GET /orders", s.ordersPage)
	mux.HandleFunc("GET /orders/new", s.orderNewPage)
	mux.HandleFunc("POST /orders", s.orderCreate)
	mux.HandleFunc("POST /orders/draft", s.orderSaveDraft)
	mux.HandleFunc("POST /orders/draft/{categoryId}/delete", s.orderDeleteDraft)
	mux.HandleFunc("GET /drafts", s.draftsPage)
	mux.HandleFunc("GET /orders/selection", s.orderSelectionPage)
	mux.HandleFunc("GET /orders/table", s.ordersTablePage)
	mux.HandleFunc("GET /orders/{number}", s.orderDetailPage)
	mux.HandleFunc("GET /orders/{number}/edit", s.orderEditPage)
	mux.HandleFunc("POST /orders/{number}/edit", s.orderUpdate)
	mux.HandleFunc("POST /orders/{number}/cancel", s.orderCancel)
	mux.HandleFunc("POST /orders/{number}/restore", s.orderRestore)
	mux.HandleFunc("POST /orders/{number}/favorite", s.orderFavorite)

	mux.HandleFunc("GET /fragments/monitor", s.monitorFragment)
	mux.HandleFunc("GET /fragments/orders/{number}", s.orderPreviewFragment)
	mux.HandleFunc("GET /calculator", s.calculatorPage)
	mux.HandleFunc("POST /calculator", s.calculateDough)

	mux.HandleFunc("GET /production", s.productionPage)
	mux.HandleFunc("POST /production", s.productionCreate)
	mux.HandleFunc("GET /production/{id}", s.productionDetailPage)
	mux.HandleFunc("POST /production/{id}", s.productionUpdate)
	mux.HandleFunc("POST /production/{id}/delete", s.productionDelete)

	mux.HandleFunc("GET /admin/users", s.adminUsersPage)
	mux.HandleFunc("POST /admin/users", s.adminUserCreate)
	mux.HandleFunc("POST /admin/users/{id}", s.adminUserUpdate)
	mux.HandleFunc("POST /admin/users/{id}/delete", s.adminUserDelete)
	mux.HandleFunc("GET /admin/dishes", s.adminDishesPage)
	mux.HandleFunc("GET /admin/techcards", s.adminTechCardsPage)
	mux.HandleFunc("POST /admin/dishes", s.adminDishCreate)
	mux.HandleFunc("POST /admin/dishes/{code}", s.adminDishUpdate)
	mux.HandleFunc("POST /admin/dishes/{code}/delete", s.adminDishDelete)
	mux.HandleFunc("POST /admin/dishes/reorder", s.adminDishReorder)
	mux.HandleFunc("POST /admin/iiko/sync", s.adminIikoSync)
	mux.HandleFunc("POST /admin/categories", s.adminCategoryCreate)
	mux.HandleFunc("POST /admin/categories/{id}", s.adminCategoryUpdate)
	mux.HandleFunc("POST /admin/categories/{id}/delete", s.adminCategoryDelete)
	mux.HandleFunc("GET /me", s.mePage)

	return s.securityHeaders(s.requestLogger(s.csrfProtection(mux)))
}

func (s *server) home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		s.renderError(w, r, http.StatusNotFound, "Страница не найдена.")
		return
	}
	http.Redirect(w, r, "/orders", http.StatusSeeOther)
}

func (s *server) health(w http.ResponseWriter, r *http.Request) {
	if err := s.queries.Health(r.Context()); err != nil {
		http.Error(w, "backend unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func (s *server) render(w http.ResponseWriter, r *http.Request, status int, value page) {
	if value.Title == "" {
		value.Title = "Bakery"
	}
	value.CurrentPath = r.URL.Path
	value.Today = time.Now().Format(time.DateOnly)
	value.CSRF = s.ensureCSRF(w, r)
	value.MiniApp = isMiniAppSession(r)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Vary", "HX-Request")
	w.WriteHeader(status)
	if err := s.templates.ExecuteTemplate(w, "layout", value); err != nil {
		s.logger.Error("render template", "view", value.View, "error", err)
	}
}

func (s *server) renderError(w http.ResponseWriter, r *http.Request, status int, message string) {
	viewer, _, _ := s.viewer(r)
	s.render(w, r, status, page{Title: "Ошибка", View: "error", Viewer: viewer, Error: message})
}

func (s *server) redirect(w http.ResponseWriter, r *http.Request, target string) {
	target = safeNext(target)
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Location", target)
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.Redirect(w, r, target, http.StatusSeeOther) //nolint:gosec // safeNext restricts redirects to local paths.
}

func (s *server) viewer(r *http.Request) (*contract.Me, application.Credentials, error) {
	cred := credentials(r)
	if cred == "" {
		return nil, "", errUnauthenticated
	}
	viewer, err := s.queries.Me(r.Context(), cred)
	if err != nil {
		return nil, cred, err
	}
	return &viewer, cred, nil
}

func (s *server) requireViewer(w http.ResponseWriter, r *http.Request) (*contract.Me, application.Credentials, bool) {
	viewer, cred, err := s.viewer(r)
	if err == nil {
		return viewer, cred, true
	}
	if application.StatusOf(err) == http.StatusUnauthorized || errors.Is(err, errUnauthenticated) {
		s.clearSession(w, r)
		if r.Header.Get("HX-Request") == "true" {
			w.Header().Set("HX-Redirect", "/login?next="+url.QueryEscape(safeNext(r.URL.RequestURI())))
			w.WriteHeader(http.StatusNoContent)
			return nil, "", false
		}
		s.render(w, r, http.StatusUnauthorized, page{
			Title: "Вход",
			View:  "login",
			Error: queryMessage(r, "error"),
			Data:  loginData{Next: safeNext(r.URL.RequestURI())},
		})
		return nil, "", false
	}
	s.renderError(w, r, http.StatusBadGateway, application.MessageOf(err, "Backend временно недоступен."))
	return nil, "", false
}

func (s *server) requireAdmin(w http.ResponseWriter, r *http.Request) (*contract.Me, application.Credentials, bool) {
	viewer, cred, ok := s.requireViewer(w, r)
	if !ok {
		return nil, "", false
	}
	if !application.IsAdmin(viewer) {
		s.renderError(w, r, http.StatusForbidden, "Недостаточно прав для этого раздела.")
		return nil, "", false
	}
	return viewer, cred, true
}

func (s *server) requireProduction(w http.ResponseWriter, r *http.Request) (*contract.Me, application.Credentials, bool) {
	viewer, cred, ok := s.requireViewer(w, r)
	if !ok {
		return nil, "", false
	}
	if !application.CanUseProduction(viewer) {
		s.renderError(w, r, http.StatusForbidden, "Раздел доступен пекарю и администратору.")
		return nil, "", false
	}
	return viewer, cred, true
}

func (s *server) requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)
		s.logger.Info("frontend request",
			"user", orAnon(sessionUsername(r)),
			"method", r.Method,
			"path", r.URL.Path,
			"status", recorder.status,
			"duration_ms", time.Since(started).Milliseconds(),
		)
	})
}

func (s *server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' https://telegram.org; style-src 'self'; img-src 'self' data:; connect-src 'self'; font-src 'self'; frame-ancestors 'self' https://web.telegram.org https://*.telegram.org; base-uri 'self'; form-action 'self'")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(w, r)
	})
}

func (s *server) csrfProtection(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.URL.Path == "/session/login" || r.URL.Path == "/session/telegram" {
			next.ServeHTTP(w, r)
			return
		}
		cookie, err := r.Cookie(csrfCookie)
		requestToken := r.Header.Get("X-CSRF-Token")
		if requestToken == "" {
			requestToken = r.FormValue("_csrf")
		}
		if err != nil || cookie.Value == "" || requestToken != cookie.Value {
			s.renderError(w, r, http.StatusForbidden, "Сессия формы устарела. Обновите страницу.")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *server) ensureCSRF(w http.ResponseWriter, r *http.Request) string {
	if cookie, err := r.Cookie(csrfCookie); err == nil && cookie.Value != "" {
		return cookie.Value
	}
	buffer := make([]byte, 24)
	if _, err := rand.Read(buffer); err != nil {
		s.logger.Error("generate csrf token", "error", err)
		return ""
	}
	token := base64.RawURLEncoding.EncodeToString(buffer)
	// Secure follows the actual request scheme so local HTTP development still works.
	//nolint:gosec
	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookie,
		Value:    token,
		Path:     "/",
		MaxAge:   7 * 24 * 60 * 60,
		Secure:   secureRequest(r),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	return token
}

func (s *server) setSession(w http.ResponseWriter, r *http.Request, username string, cred application.Credentials, expires time.Time) {
	// Store the login alongside the credentials so request logs can name the
	// acting user without an extra backend round-trip. Separator is a newline,
	// which never appears in a bearer token or tma initData.
	value := base64.RawURLEncoding.EncodeToString([]byte(username + "\n" + string(cred)))
	// Secure follows the actual request scheme so local HTTP development still works.
	//nolint:gosec
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    value,
		Path:     "/",
		Expires:  expires,
		MaxAge:   int(time.Until(expires).Seconds()),
		Secure:   secureRequest(r),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func (s *server) clearSession(w http.ResponseWriter, r *http.Request) {
	// Secure follows the actual request scheme so local HTTP development still works.
	//nolint:gosec
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Path:     "/",
		MaxAge:   -1,
		Secure:   secureRequest(r),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func credentials(r *http.Request) application.Credentials {
	_, cred := parseSession(r)
	return cred
}

// isMiniAppSession reports whether the session carries Telegram initData
// credentials. Such sessions have no logout: the identity comes from Telegram
// itself and the next request would auto-login again.
func isMiniAppSession(r *http.Request) bool {
	return strings.HasPrefix(string(credentials(r)), "tma ")
}

// sessionUsername returns the login stored in the session cookie, or "" for
// anonymous requests or legacy cookies written before the login was stored.
func sessionUsername(r *http.Request) string {
	user, _ := parseSession(r)
	return user
}

func parseSession(r *http.Request) (string, application.Credentials) {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil {
		return "", ""
	}
	decoded, err := base64.RawURLEncoding.DecodeString(cookie.Value)
	if err != nil {
		return "", ""
	}
	// New cookies are "<username>\n<credentials>"; legacy cookies are just the
	// credentials, so a missing separator means the whole value is the token.
	if user, cred, ok := strings.Cut(string(decoded), "\n"); ok {
		return user, application.Credentials(cred)
	}
	return "", application.Credentials(decoded)
}

// orAnon labels requests with no logged-in user so logs read clearly.
func orAnon(user string) string {
	if user == "" {
		return "anonymous"
	}
	return user
}

func secureRequest(r *http.Request) bool {
	return r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

func safeNext(value string) string {
	parsed, err := url.Parse(value)
	if err != nil || parsed.IsAbs() || !strings.HasPrefix(parsed.Path, "/") || strings.HasPrefix(parsed.Path, "//") {
		return "/orders"
	}
	return parsed.RequestURI()
}

func queryMessage(r *http.Request, key string) string {
	return strings.TrimSpace(r.URL.Query().Get(key))
}

func parseID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		return 0, fmt.Errorf("invalid id")
	}
	return id, nil
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (w *statusRecorder) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

var errUnauthenticated = errors.New("unauthenticated")
