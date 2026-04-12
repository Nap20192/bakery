package http

import (
	"encoding/json"
	"net/http"

	"bakery/internal/app"
)

type Server struct {
	mux     *http.ServeMux
	kitchen *app.KitchenService
	client  *app.ClientService
	order   *app.OrderService
}

func NewServer(kitchen *app.KitchenService, client *app.ClientService, order *app.OrderService) *Server {
	s := &Server{
		mux:     http.NewServeMux(),
		kitchen: kitchen,
		client:  client,
		order:   order,
	}
	s.routes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	s.mux.Handle("GET /", http.FileServer(http.Dir("static")))

	s.mux.HandleFunc("GET /kitchens", s.listKitchens)
	s.mux.HandleFunc("POST /kitchens", s.createKitchen)
	s.mux.HandleFunc("GET /kitchens/{id}", s.getKitchen)
	s.mux.HandleFunc("PUT /kitchens/{id}", s.updateKitchen)
	s.mux.HandleFunc("DELETE /kitchens/{id}", s.deleteKitchen)

	s.mux.HandleFunc("GET /clients", s.listClients)
	s.mux.HandleFunc("POST /clients", s.createClient)
	s.mux.HandleFunc("GET /clients/{id}", s.getClient)
	s.mux.HandleFunc("PUT /clients/{id}", s.updateClient)
	s.mux.HandleFunc("DELETE /clients/{id}", s.deleteClient)

	s.mux.HandleFunc("GET /products", s.listProducts)
	s.mux.HandleFunc("POST /products", s.createProduct)
	s.mux.HandleFunc("GET /products/{id}", s.getProduct)

	s.mux.HandleFunc("GET /orders", s.listOrders)
	s.mux.HandleFunc("POST /orders", s.createOrder)
	s.mux.HandleFunc("GET /orders/{id}", s.getOrder)
	s.mux.HandleFunc("PATCH /orders/{id}/status", s.updateOrderStatus)
	s.mux.HandleFunc("DELETE /orders/{id}", s.cancelOrder)
	s.mux.HandleFunc("POST /orders/{id}/items", s.addOrderItem)
	s.mux.HandleFunc("GET /orders/{id}/items", s.listOrderItems)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func decodeJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}
