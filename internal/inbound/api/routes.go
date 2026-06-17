package api

import "net/http"

func (s *Server) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("POST /login", s.handleLogin)
	mux.Handle("GET /me", s.requireAuth(http.HandlerFunc(s.handleMe)))
	mux.Handle("GET /users", s.requireAdmin(http.HandlerFunc(s.handleListUsers)))
	mux.Handle("POST /users", s.requireAdmin(http.HandlerFunc(s.handleCreateUser)))
	mux.Handle("PATCH /users/{id}", s.requireAdmin(http.HandlerFunc(s.handleUpdateUser)))
	mux.Handle("GET /admin/departments", s.requireAdmin(http.HandlerFunc(s.handleAdminDepartments)))
	mux.Handle("GET /catalog", s.requireMiniAppAuth(http.HandlerFunc(s.handleCatalog)))
	mux.Handle("GET /departments", s.requireMiniAppAuth(http.HandlerFunc(s.handleListDepartments)))
	mux.Handle("GET /orders", s.requireMiniAppAuth(http.HandlerFunc(s.handleListOrders)))
	mux.Handle("POST /orders", s.requireMiniAppAuth(http.HandlerFunc(s.handleCreateOrder)))
	mux.Handle("GET /orders/{id}", s.requireMiniAppAuth(http.HandlerFunc(s.handleOrderByID)))
	mux.Handle("PUT /orders/{id}", s.requireMiniAppAuth(http.HandlerFunc(s.handleUpdateOrder)))
	mux.Handle("GET /monitor/batch", s.requireMiniAppAuth(http.HandlerFunc(s.handleMonitorBatch)))
	mux.Handle("GET /monitor/{id}", s.requireMiniAppAuth(http.HandlerFunc(s.handleMonitorDefault)))
	mux.Handle("GET /monitor/{id}/{product_id}", s.requireMiniAppAuth(http.HandlerFunc(s.handleMonitorByProduct)))
}
