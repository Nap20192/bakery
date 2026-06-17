package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	adminuc "bakery/internal/services/admin/usecase/admin"
	authuc "bakery/internal/services/auth/usecase/auth"
	departmentuc "bakery/internal/services/department/usecase/department"
	monitoruc "bakery/internal/services/monitor/usecase/monitor"
	orderuc "bakery/internal/services/order/usecase/order"
)

type ServerConfig struct {
	Addr           string
	AllowedOrigins string
	BotToken       string
}

type Server struct {
	orderSvc      orderuc.UseCase
	monitorSvc    monitoruc.UseCase
	departmentSvc departmentuc.UseCase
	authSvc       authuc.UseCase
	adminSvc      adminuc.UseCase
	config        ServerConfig
	server        *http.Server
}

func NewServer(
	orderSvc orderuc.UseCase,
	monitorSvc monitoruc.UseCase,
	departmentSvc departmentuc.UseCase,
	authSvc authuc.UseCase,
	adminSvc adminuc.UseCase,
	config ServerConfig,
) *Server {
	return &Server{
		orderSvc:      orderSvc,
		monitorSvc:    monitorSvc,
		departmentSvc: departmentSvc,
		authSvc:       authSvc,
		adminSvc:      adminSvc,
		config:        config,
	}
}

func (s *Server) Start() error {
	mux := http.NewServeMux()
	s.registerRoutes(mux)
	s.server = &http.Server{
		Addr:              s.config.Addr,
		Handler:           s.withMiddleware(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}
	if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s == nil || s.server == nil {
		return nil
	}
	return s.server.Shutdown(ctx)
}

func parseRequestDate(raw string) (time.Time, error) {
	raw = trim(raw)
	if raw == "" {
		return time.Time{}, nil
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t, nil
	}
	if t, err := time.Parse("2006-01-02", raw); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("invalid date format: %q", raw)
}
