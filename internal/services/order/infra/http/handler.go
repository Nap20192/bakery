package orderhttp

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"strconv"

	"bakery/internal/inbound/api/httpx"
	departmentuc "bakery/internal/services/department/usecase/department"
	orderdomain "bakery/internal/services/order/domain"
	orderuc "bakery/internal/services/order/usecase/order"
)

// Handler is the order service HTTP delivery adapter.
type Handler struct {
	orderSvc      orderuc.UseCase
	departmentSvc departmentuc.UseCase
	presenter     *OrderPresenter
}

func New(orderSvc orderuc.UseCase, departmentSvc departmentuc.UseCase, presenter *OrderPresenter) *Handler {
	return &Handler{orderSvc: orderSvc, departmentSvc: departmentSvc, presenter: presenter}
}

// RegisterRoutes wires the order endpoints behind Mini App authentication.
func (h *Handler) RegisterRoutes(mux *http.ServeMux, auth *httpx.Authenticator) {
	mux.Handle("GET /catalog", auth.RequireMiniAppAuth(http.HandlerFunc(h.handleCatalog)))
	mux.Handle("GET /orders", auth.RequireMiniAppAuth(http.HandlerFunc(h.handleListOrders)))
	mux.Handle("POST /orders", auth.RequireMiniAppAuth(http.HandlerFunc(h.handleCreateOrder)))
	mux.Handle("GET /orders/{id}", auth.RequireMiniAppAuth(http.HandlerFunc(h.handleOrderByID)))
	mux.Handle("PUT /orders/{id}", auth.RequireMiniAppAuth(http.HandlerFunc(h.handleUpdateOrder)))
	h.RegisterAdminRoutes(mux, auth)
}

func (h *Handler) handleListOrders(w http.ResponseWriter, r *http.Request) {
	if h.orderSvc == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "Сервис заказов временно недоступен.")
		return
	}

	user, _ := httpx.MiniAppUserFromContext(r.Context())
	limit := int32(10)
	if raw := httpx.Trim(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 32)
		if err != nil || parsed <= 0 {
			httpx.WriteError(w, http.StatusBadRequest, "Параметр limit должен быть положительным числом.")
			return
		}
		limit = int32(parsed)
	}
	page := int32(1)
	if raw := httpx.Trim(r.URL.Query().Get("page")); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 32)
		if err != nil || parsed <= 0 {
			httpx.WriteError(w, http.StatusBadRequest, "Параметр page должен быть положительным числом.")
			return
		}
		page = int32(parsed)
	}
	offset := (page - 1) * limit
	var fromDepartmentID *int64
	if httpx.IsShopUser(user) {
		fromDepartmentID = &user.DepartmentID
	} else if raw := httpx.Trim(r.URL.Query().Get("from_department_id")); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || parsed <= 0 {
			httpx.WriteError(w, http.StatusBadRequest, "Параметр from_department_id должен быть положительным числом.")
			return
		}
		fromDepartmentID = &parsed
	}
	fulfillmentDate, err := httpx.ParseRequestDate(r.URL.Query().Get("fulfillment_date"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Дата выполнения должна быть в формате YYYY-MM-DD.")
		return
	}

	result, err := h.orderSvc.ListOrders(r.Context(), orderuc.ListOrdersInput{
		Limit:            limit,
		Offset:           offset,
		FromDepartmentID: fromDepartmentID,
		FulfillmentDate:  fulfillmentDate,
	})
	if err != nil {
		slog.ErrorContext(r.Context(), "list orders failed", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "Не удалось получить список заказов.")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, ordersPageResponse{
		Items:      h.presenter.BuildOrderResponses(r.Context(), result.Orders),
		Page:       page,
		Limit:      result.Limit,
		Offset:     result.Offset,
		Total:      result.Total,
		TotalPages: int32(math.Ceil(float64(result.Total) / float64(result.Limit))),
	})
}

func (h *Handler) handleOrderByID(w http.ResponseWriter, r *http.Request) {
	if h.orderSvc == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "Сервис заказов временно недоступен.")
		return
	}

	id := httpx.Trim(r.PathValue("id"))
	if id == "" {
		httpx.WriteError(w, http.StatusBadRequest, "Укажите номер заказа.")
		return
	}

	order, err := h.orderSvc.GetOrderByNumber(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, fmt.Sprintf("Заказ %s не найден.", id))
		return
	}
	if !UserCanReadOrder(r.Context(), order) {
		httpx.WriteError(w, http.StatusNotFound, fmt.Sprintf("Заказ %s не найден.", id))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, h.presenter.BuildOrderResponse(r.Context(), order))
}

// UserCanReadOrder reports whether the request's viewer may read the order:
// workshops see everything, shops only their own orders.
func UserCanReadOrder(ctx context.Context, order orderdomain.Order) bool {
	user, ok := httpx.MiniAppUserFromContext(ctx)
	if !ok {
		return false
	}
	if !httpx.IsShopUser(user) {
		return httpx.IsWorkshopUser(user)
	}
	return order.FromDepartmentID != nil && *order.FromDepartmentID == user.DepartmentID
}
