package orderhttp

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"bakery/internal/inbound/api/contract"
	"bakery/internal/inbound/api/httpx"
	orderdomain "bakery/internal/services/order/domain"
)

func dishWriteToDomain(req contract.DishWrite) orderdomain.DishCatalogItem {
	return orderdomain.DishCatalogItem{
		Code:       req.Code,
		Name:       req.Name,
		Theme:      req.Theme,
		CategoryID: req.CategoryID,
		SortOrder:  req.SortOrder,
	}
}

func toDishResponse(item orderdomain.DishCatalogItem) contract.Dish {
	return contract.Dish{Code: item.Code, Name: item.Name, Theme: item.Theme, CategoryID: item.CategoryID, SortOrder: item.SortOrder}
}

func categoryWriteToDomain(req contract.CategoryWrite) orderdomain.OrderCategory {
	return orderdomain.OrderCategory{
		Code:         req.Code,
		Letter:       req.Letter,
		Name:         req.Name,
		Color:        req.Color,
		SortOrder:    req.SortOrder,
		MonitorCodes: req.MonitorCodes,
	}
}

// RegisterAdminRoutes wires admin-only dish catalog management, тип заявки
// management and order pins.
func (h *Handler) RegisterAdminRoutes(mux *http.ServeMux, auth *httpx.Authenticator) {
	mux.Handle("GET /admin/dishes", auth.RequireAdmin(http.HandlerFunc(h.handleListDishes)))
	mux.Handle("GET /admin/dishes/techcards", auth.RequireAdmin(http.HandlerFunc(h.handleListDishTechCards)))
	mux.Handle("GET /admin/dishes/available", auth.RequireAdmin(http.HandlerFunc(h.handleListAvailableDishes)))
	mux.Handle("POST /admin/dishes", auth.RequireAdmin(http.HandlerFunc(h.handleCreateDish)))
	mux.Handle("PUT /admin/dishes/reorder", auth.RequireAdmin(http.HandlerFunc(h.handleReorderDishes)))
	mux.Handle("PUT /admin/dishes/{code}", auth.RequireAdmin(http.HandlerFunc(h.handleUpdateDish)))
	mux.Handle("DELETE /admin/dishes/{code}", auth.RequireAdmin(http.HandlerFunc(h.handleDeleteDish)))
	mux.Handle("POST /admin/categories", auth.RequireAdmin(http.HandlerFunc(h.handleCreateCategory)))
	mux.Handle("PUT /admin/categories/{id}", auth.RequireAdmin(http.HandlerFunc(h.handleUpdateCategory)))
	mux.Handle("DELETE /admin/categories/{id}", auth.RequireAdmin(http.HandlerFunc(h.handleDeleteCategory)))
	mux.Handle("PATCH /orders/{id}/favorite", auth.RequireAdmin(http.HandlerFunc(h.handleSetFavorite)))
}

func (h *Handler) handleCreateCategory(w http.ResponseWriter, r *http.Request) {
	var req contract.CategoryWrite
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Некорректные данные.")
		return
	}
	category, err := h.orderSvc.CreateOrderCategory(r.Context(), categoryWriteToDomain(req))
	if err != nil {
		httpx.WriteAppError(w, r, err, "Не удалось создать тип заявки.")
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, buildCategoryResponse(&category))
}

func (h *Handler) handleUpdateCategory(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(httpx.Trim(r.PathValue("id")), 10, 64)
	if err != nil || id <= 0 {
		httpx.WriteError(w, http.StatusBadRequest, "Укажите тип заявки.")
		return
	}
	var req contract.CategoryWrite
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Некорректные данные.")
		return
	}
	category, err := h.orderSvc.UpdateOrderCategory(r.Context(), id, categoryWriteToDomain(req))
	if err != nil {
		httpx.WriteAppError(w, r, err, "Не удалось обновить тип заявки.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, buildCategoryResponse(&category))
}

func (h *Handler) handleDeleteCategory(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(httpx.Trim(r.PathValue("id")), 10, 64)
	if err != nil || id <= 0 {
		httpx.WriteError(w, http.StatusBadRequest, "Укажите тип заявки.")
		return
	}
	if err := h.orderSvc.DeleteOrderCategory(r.Context(), id); err != nil {
		httpx.WriteAppError(w, r, err, "Не удалось удалить тип заявки.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleSetFavorite(w http.ResponseWriter, r *http.Request) {
	number := httpx.Trim(r.PathValue("id"))
	var req contract.FavoriteUpdate
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
	out := make([]contract.Dish, 0, len(items))
	for _, item := range items {
		out = append(out, toDishResponse(item))
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

// handleListDishTechCards returns every catalog dish's tech card
// (ingredient list), grouped by order category, for admin review.
func (h *Handler) handleListDishTechCards(w http.ResponseWriter, r *http.Request) {
	dishes, err := h.orderSvc.ListDishCatalog(r.Context())
	if err != nil {
		slog.ErrorContext(r.Context(), "admin list dish techcards: list dishes failed", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "Не удалось получить блюда.")
		return
	}
	categories, err := h.orderSvc.ListOrderCategories(r.Context())
	if err != nil {
		slog.ErrorContext(r.Context(), "admin list dish techcards: list categories failed", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "Не удалось получить типы заявок.")
		return
	}

	byCategory := make(map[int64]*contract.TechCardCategory, len(categories))
	out := make([]contract.TechCardCategory, len(categories))
	for i, cat := range categories {
		out[i] = contract.TechCardCategory{ID: cat.ID, Name: cat.Name, Dishes: []contract.TechCardDish{}}
		byCategory[cat.ID] = &out[i]
	}

	now := time.Now()
	for _, dish := range dishes {
		if dish.CategoryID == nil {
			continue
		}
		cat, ok := byCategory[*dish.CategoryID]
		if !ok {
			continue
		}
		cat.Dishes = append(cat.Dishes, h.buildDishTechCard(r, dish, now))
	}

	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) buildDishTechCard(r *http.Request, dish orderdomain.DishCatalogItem, date time.Time) contract.TechCardDish {
	out := contract.TechCardDish{Code: dish.Code, Name: dish.Name, Ingredients: []contract.TechCardIngredient{}}
	if h.techcardSvc == nil {
		out.Error = "техкарты недоступны"
		return out
	}
	card, err := h.techcardSvc.GetByCode(r.Context(), dish.Code, date)
	if err != nil {
		out.Error = "техкарта не найдена"
		return out
	}
	out.Unit = card.Unit

	switch {
	case card.Assembly != nil:
		for _, item := range card.Assembly.Items {
			product := card.Products[item.ProductID]
			out.Ingredients = append(out.Ingredients, contract.TechCardIngredient{
				Code: product.Code, Name: product.Name, Unit: product.Unit, Amount: item.AmountIn,
			})
		}
	case card.Prepared != nil:
		for _, item := range card.Prepared.Items {
			product := card.Products[item.ProductID]
			out.Ingredients = append(out.Ingredients, contract.TechCardIngredient{
				Code: product.Code, Name: product.Name, Unit: product.Unit, Amount: item.Amount,
			})
		}
	default:
		out.Error = "техкарта пуста"
	}
	return out
}

func (h *Handler) handleListAvailableDishes(w http.ResponseWriter, r *http.Request) {
	query := httpx.Trim(r.URL.Query().Get("q"))
	dishes, err := h.orderSvc.SearchAvailableDishes(r.Context(), query, 20)
	if err != nil {
		slog.ErrorContext(r.Context(), "admin search available dishes failed", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "Не удалось получить блюда.")
		return
	}
	out := make([]contract.AvailableDish, 0, len(dishes))
	for _, dish := range dishes {
		out = append(out, contract.AvailableDish{Code: dish.Code, Name: dish.Name, Unit: dish.Unit})
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) handleReorderDishes(w http.ResponseWriter, r *http.Request) {
	var req contract.ReorderDishes
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Некорректные данные.")
		return
	}
	if err := h.orderSvc.ReorderDishCatalog(r.Context(), req.Codes); err != nil {
		httpx.WriteAppError(w, r, err, "Не удалось изменить порядок.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleCreateDish(w http.ResponseWriter, r *http.Request) {
	var req contract.DishWrite
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Некорректные данные.")
		return
	}
	item, err := h.orderSvc.AddDishCatalogItem(r.Context(), dishWriteToDomain(req))
	if err != nil {
		httpx.WriteAppError(w, r, err, "Не удалось добавить блюдо.")
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, toDishResponse(item))
}

func (h *Handler) handleUpdateDish(w http.ResponseWriter, r *http.Request) {
	code := httpx.Trim(r.PathValue("code"))
	var req contract.DishWrite
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Некорректные данные.")
		return
	}
	item, err := h.orderSvc.UpdateDishCatalogItem(r.Context(), code, dishWriteToDomain(req))
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
