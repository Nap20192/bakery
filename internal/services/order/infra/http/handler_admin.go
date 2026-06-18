package orderhttp

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"bakery/internal/inbound/api/httpx"
)

type dishResponse struct {
	Code  string `json:"code"`
	Name  string `json:"name"`
	Theme string `json:"theme"`
}

type dishCreateRequest struct {
	Name  string `json:"name"`
	Theme string `json:"theme"`
}

// RegisterAdminRoutes wires admin-only dish catalog management.
func (h *Handler) RegisterAdminRoutes(mux *http.ServeMux, auth *httpx.Authenticator) {
	mux.Handle("GET /admin/dishes", auth.RequireAdmin(http.HandlerFunc(h.handleListDishes)))
	mux.Handle("POST /admin/dishes", auth.RequireAdmin(http.HandlerFunc(h.handleCreateDish)))
	mux.Handle("DELETE /admin/dishes/{code}", auth.RequireAdmin(http.HandlerFunc(h.handleDeleteDish)))
}

func (h *Handler) handleListDishes(w http.ResponseWriter, r *http.Request) {
	items, err := h.orderSvc.ListDishCatalog(r.Context())
	if err != nil {
		slog.ErrorContext(r.Context(), "admin list dishes failed", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "Не удалось получить блюда.")
		return
	}
	out := make([]dishResponse, 0, len(items))
	for _, item := range items {
		out = append(out, dishResponse{Code: item.Code, Name: item.Name, Theme: item.Theme})
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) handleCreateDish(w http.ResponseWriter, r *http.Request) {
	var req dishCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Некорректные данные.")
		return
	}
	item, err := h.orderSvc.AddDishCatalogItem(r.Context(), req.Name, req.Theme)
	if err != nil {
		httpx.WriteAppError(w, r, err, "Не удалось добавить блюдо.")
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, dishResponse{Code: item.Code, Name: item.Name, Theme: item.Theme})
}

func (h *Handler) handleDeleteDish(w http.ResponseWriter, r *http.Request) {
	code := httpx.Trim(r.PathValue("code"))
	if err := h.orderSvc.DeleteDishCatalogItem(r.Context(), code); err != nil {
		httpx.WriteAppError(w, r, err, "Не удалось удалить блюдо.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
