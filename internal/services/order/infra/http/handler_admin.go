package orderhttp

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"bakery/internal/inbound/api/httpx"
	orderdomain "bakery/internal/services/order/domain"
)

func (req dishWriteRequest) toDomain() orderdomain.DishCatalogItem {
	return orderdomain.DishCatalogItem{
		Code:      req.Code,
		Name:      req.Name,
		Theme:     req.Theme,
		SortOrder: req.SortOrder,
	}
}

func toDishResponse(item orderdomain.DishCatalogItem) dishResponse {
	return dishResponse{Code: item.Code, Name: item.Name, Theme: item.Theme, SortOrder: item.SortOrder}
}

type dishResponse struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	Theme     string `json:"theme"`
	SortOrder int64  `json:"sort_order"`
}

type dishWriteRequest struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	Theme     string `json:"theme"`
	SortOrder int64  `json:"sort_order"`
}

// RegisterAdminRoutes wires admin-only dish catalog management and order pins.
func (h *Handler) RegisterAdminRoutes(mux *http.ServeMux, auth *httpx.Authenticator) {
	mux.Handle("GET /admin/dishes", auth.RequireAdmin(http.HandlerFunc(h.handleListDishes)))
	mux.Handle("POST /admin/dishes", auth.RequireAdmin(http.HandlerFunc(h.handleCreateDish)))
	mux.Handle("PUT /admin/dishes/{code}", auth.RequireAdmin(http.HandlerFunc(h.handleUpdateDish)))
	mux.Handle("DELETE /admin/dishes/{code}", auth.RequireAdmin(http.HandlerFunc(h.handleDeleteDish)))
	mux.Handle("PATCH /orders/{id}/favorite", auth.RequireAdmin(http.HandlerFunc(h.handleSetFavorite)))
}

type favoriteRequest struct {
	Favorite bool `json:"favorite"`
}

func (h *Handler) handleSetFavorite(w http.ResponseWriter, r *http.Request) {
	number := httpx.Trim(r.PathValue("id"))
	var req favoriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Некорректные данные.")
		return
	}
	order, err := h.orderSvc.SetOrderFavorite(r.Context(), number, req.Favorite)
	if err != nil {
		httpx.WriteAppError(w, r, err, "Не удалось обновить заказ.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, h.presenter.BuildOrderResponse(r.Context(), order))
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
		out = append(out, toDishResponse(item))
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) handleCreateDish(w http.ResponseWriter, r *http.Request) {
	var req dishWriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Некорректные данные.")
		return
	}
	item, err := h.orderSvc.AddDishCatalogItem(r.Context(), req.toDomain())
	if err != nil {
		httpx.WriteAppError(w, r, err, "Не удалось добавить блюдо.")
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, toDishResponse(item))
}

func (h *Handler) handleUpdateDish(w http.ResponseWriter, r *http.Request) {
	code := httpx.Trim(r.PathValue("code"))
	var req dishWriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Некорректные данные.")
		return
	}
	item, err := h.orderSvc.UpdateDishCatalogItem(r.Context(), code, req.toDomain())
	if err != nil {
		httpx.WriteAppError(w, r, err, "Не удалось обновить блюдо.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toDishResponse(item))
}

func (h *Handler) handleDeleteDish(w http.ResponseWriter, r *http.Request) {
	code := httpx.Trim(r.PathValue("code"))
	if err := h.orderSvc.DeleteDishCatalogItem(r.Context(), code); err != nil {
		httpx.WriteAppError(w, r, err, "Не удалось удалить блюдо.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
