package api

import (
	"net/http"

	"bakery/internal/inbound/api/httpx"
	adminhttp "bakery/internal/services/admin/infra/http"
	adminuc "bakery/internal/services/admin/usecase/admin"
	authhttp "bakery/internal/services/auth/infra/http"
	authuc "bakery/internal/services/auth/usecase/auth"
	departmenthttp "bakery/internal/services/department/infra/http"
	departmentuc "bakery/internal/services/department/usecase/department"
	monitorhttp "bakery/internal/services/monitor/infra/http"
	monitoruc "bakery/internal/services/monitor/usecase/monitor"
	orderhttp "bakery/internal/services/order/infra/http"
	orderuc "bakery/internal/services/order/usecase/order"
	synchttp "bakery/internal/services/sync/infra/http"
	syncuc "bakery/internal/services/sync/usecase/sync"
	techcarduc "bakery/internal/services/techcard/usecase/techcard"
)

type ServerConfig struct {
	BotToken string
}

// routeRegistrar is implemented by each service's HTTP delivery adapter.
type routeRegistrar interface {
	RegisterRoutes(mux *http.ServeMux, auth *httpx.Authenticator)
}

type Server struct {
	auth     *httpx.Authenticator
	handlers []routeRegistrar
}

func NewServer(
	orderSvc orderuc.UseCase,
	monitorSvc monitoruc.UseCase,
	departmentSvc departmentuc.UseCase,
	authSvc authuc.UseCase,
	adminSvc adminuc.UseCase,
	techcardSvc techcarduc.UseCase,
	syncSvc syncuc.UseCase,
	config ServerConfig,
) *Server {
	presenter := orderhttp.NewPresenter(departmentSvc)
	return &Server{
		auth: httpx.NewAuthenticator(authSvc, departmentSvc, config.BotToken),
		handlers: []routeRegistrar{
			authhttp.New(authSvc, departmentSvc, config.BotToken),
			adminhttp.New(adminSvc),
			departmenthttp.New(departmentSvc),
			orderhttp.New(orderSvc, departmentSvc, techcardSvc, presenter),
			monitorhttp.New(monitorSvc, orderSvc, presenter),
			synchttp.New(syncSvc),
		},
	}
}

// Handler returns the API with its middleware. It is mounted only on the
// in-process transport the web UI uses, never on a public listener.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.handleHealth)
	for _, h := range s.handlers {
		h.RegisterRoutes(mux, s.auth)
	}
	return s.withMiddleware(mux)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
