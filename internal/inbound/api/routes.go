package api

import "net/http"

func (s *Server) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /departments", s.handleListDepartments)
	mux.HandleFunc("GET /orders", s.handleListOrders)
	mux.HandleFunc("GET /orders/{id}", s.handleOrderByID)
	mux.HandleFunc("GET /monitor/batch", s.handleMonitorBatch)
	mux.HandleFunc("GET /monitor/{id}", s.handleMonitorDefault)
	mux.HandleFunc("GET /monitor/{id}/{product_id}", s.handleMonitorByProduct)
}
