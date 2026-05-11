package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"bakery/internal/app"
)

type Server struct {
	orderSvc   *app.OrderService
	monitorSvc *app.MonitorService
	server     *http.Server
}

func NewOrderBot(
	orderSvc *app.OrderService,
	monitorSvc *app.MonitorService,
) *Server {
	return &Server{
		orderSvc:   orderSvc,
		monitorSvc: monitorSvc,
	}
}

func NewServer(orderSvc *app.OrderService, monitorSvc *app.MonitorService) *Server {
	return NewOrderBot(orderSvc, monitorSvc)
}

func (s *Server) Start() error {
	mux := http.NewServeMux()
	s.registerRoutes(mux)
	s.server = &http.Server{
		Addr:    resolveHTTPAddr(),
		Handler: s.withMiddleware(mux),
	}
	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s == nil || s.server == nil {
		return nil
	}
	return s.server.Shutdown(ctx)
}

func resolveHTTPAddr() string {
	if addr := os.Getenv("HTTP_ADDR"); addr != "" {
		return addr
	}
	port := 8080
	if raw := os.Getenv("HTTP_PORT"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			port = parsed
		}
	}
	return fmt.Sprintf(":%d", port)
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
