package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"bakery/internal/app"
)

type ServerConfig struct {
	Addr           string
	AllowedOrigins string
}

type Server struct {
	orderSvc      *app.OrderService
	monitorSvc    *app.MonitorService
	departmentSvc *app.DepartmentService
	config        ServerConfig
	server        *http.Server
}

func NewServer(
	orderSvc *app.OrderService,
	monitorSvc *app.MonitorService,
	departmentSvc *app.DepartmentService,
	config ServerConfig,
) *Server {
	return &Server{
		orderSvc:      orderSvc,
		monitorSvc:    monitorSvc,
		departmentSvc: departmentSvc,
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
