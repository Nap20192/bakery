package http

import (
	"encoding/json"
	"net/http"

	"bakery/internal/app"
	"bakery/internal/domain/user"
)

type Server struct {
	mux     *http.ServeMux
	auth    *app.AuthService
	kitchen *app.KitchenService
	client  *app.ClientService
	order   *app.OrderService
	product *app.ProductService
}

func NewServer(
	auth *app.AuthService,
	kitchen *app.KitchenService,
	client *app.ClientService,
	order *app.OrderService,
	product *app.ProductService,
) *Server {
	s := &Server{
		mux:     http.NewServeMux(),
		auth:    auth,
		kitchen: kitchen,
		client:  client,
		order:   order,
		product: product,
	}
	s.routes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	authMW := AuthMiddleware(s.auth)
	protect := func(h http.HandlerFunc) http.Handler {
		return authMW(http.HandlerFunc(RequireAuth(h)))
	}
	role := func(h http.HandlerFunc, roles ...user.Role) http.Handler {
		return authMW(http.HandlerFunc(RequireRole(roles...)(h)))
	}

	s.mux.Handle("GET /", http.FileServer(http.Dir("static")))

	// auth (public)
	s.mux.HandleFunc("POST /auth/register/client", s.registerClient)
	s.mux.HandleFunc("POST /auth/register/kitchen", s.registerKitchen)
	s.mux.HandleFunc("POST /auth/login", s.login)

	// kitchens
	s.mux.Handle("GET /kitchens", role(s.listKitchens, user.RoleAdmin, user.RoleKitchen))
	s.mux.Handle("POST /kitchens", role(s.createKitchen, user.RoleAdmin))
	s.mux.Handle("GET /kitchens/{id}", role(s.getKitchen, user.RoleAdmin, user.RoleKitchen))
	s.mux.Handle("PUT /kitchens/{id}", role(s.updateKitchen, user.RoleAdmin))
	s.mux.Handle("DELETE /kitchens/{id}", role(s.deleteKitchen, user.RoleAdmin))

	// clients
	s.mux.Handle("GET /clients", role(s.listClients, user.RoleAdmin, user.RoleKitchen))
	s.mux.Handle("POST /clients", role(s.createClient, user.RoleAdmin))
	s.mux.Handle("GET /clients/{id}", role(s.getClient, user.RoleAdmin, user.RoleKitchen, user.RoleClient))
	s.mux.Handle("PUT /clients/{id}", role(s.updateClient, user.RoleAdmin))
	s.mux.Handle("DELETE /clients/{id}", role(s.deleteClient, user.RoleAdmin))

	// products — каталог читают все авторизованные, пишет admin
	s.mux.Handle("GET /products", protect(s.listProducts))
	s.mux.Handle("POST /products", role(s.createProduct, user.RoleAdmin))
	s.mux.Handle("GET /products/{id}", protect(s.getProduct))

	// orders — любой авторизованный, ownership проверяется в хендлерах
	s.mux.Handle("GET /orders", protect(s.listOrders))
	s.mux.Handle("POST /orders", protect(s.createOrder))
	s.mux.Handle("GET /orders/{id}", protect(s.getOrder))
	s.mux.Handle("PATCH /orders/{id}/status", role(s.updateOrderStatus, user.RoleAdmin, user.RoleKitchen))
	s.mux.Handle("DELETE /orders/{id}", protect(s.cancelOrder))
	s.mux.Handle("POST /orders/{id}/items", protect(s.addOrderItem))
	s.mux.Handle("GET /orders/{id}/items", protect(s.listOrderItems))
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
