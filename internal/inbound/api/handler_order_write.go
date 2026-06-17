package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	orderdomain "bakery/internal/domain/order"
	orderuc "bakery/internal/services/order/usecase/order"
)

const (
	workshopDepartmentCode = "pekari"
	maxOrderItems          = 250
	maxItemQuantity        = 100000
)

type orderWriteRequest struct {
	Items           []orderWriteItem `json:"items"`
	FulfillmentDate string           `json:"fulfillment_date"`
}

type orderWriteItem struct {
	ProductName      string  `json:"product_name"`
	Quantity         float64 `json:"quantity"`
	ReservedQuantity float64 `json:"reserved_quantity"`
}

func (s *Server) handleCatalog(w http.ResponseWriter, r *http.Request) {
	if s.orderSvc == nil {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "Сервис заказов временно недоступен."})
		return
	}
	user, _ := miniAppUserFromContext(r.Context())
	if !isShopMiniAppUser(user) {
		writeJSON(w, http.StatusForbidden, errorResponse{Error: "Каталог нужен только магазину для создания заказа."})
		return
	}
	items, err := s.orderSvc.ListDishCatalog(r.Context())
	if err != nil {
		slog.ErrorContext(r.Context(), "list dish catalog failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "Не удалось загрузить список блюд."})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleCreateOrder(w http.ResponseWriter, r *http.Request) {
	if s.orderSvc == nil || s.departmentSvc == nil {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "Сервис заказов временно недоступен."})
		return
	}
	user, ok := s.shopOrderWriter(w, r)
	if !ok {
		return
	}
	input, ok := decodeOrderWriteRequest(w, r)
	if !ok {
		return
	}

	toDepartmentID, ok := s.workshopDepartmentID(w, r)
	if !ok {
		return
	}
	fromDepartmentID := user.DepartmentID
	order, err := s.orderSvc.CreateOrder(r.Context(), orderdomain.CreateOrderInput{
		Items:             input.items,
		FromDepartmentID:  &fromDepartmentID,
		ToDepartmentID:    &toDepartmentID,
		CreatedByUsername: miniAppOrderAuthor(user),
		FulfillmentDate:   input.fulfillmentDate,
	})
	if err != nil {
		slog.WarnContext(r.Context(), "mini app create order failed", "error", err, "telegram_id", user.TelegramID)
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Не удалось создать заказ. Проверьте позиции и количества."})
		return
	}
	writeJSON(w, http.StatusCreated, s.buildOrderResponse(r.Context(), order))
}

func (s *Server) handleUpdateOrder(w http.ResponseWriter, r *http.Request) {
	if s.orderSvc == nil {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "Сервис заказов временно недоступен."})
		return
	}
	user, ok := s.shopOrderWriter(w, r)
	if !ok {
		return
	}
	number := trim(r.PathValue("id"))
	if number == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Укажите номер заказа."})
		return
	}
	existing, err := s.orderSvc.GetOrderByNumber(r.Context(), number)
	if err != nil || !miniAppUserCanReadOrder(r.Context(), existing) {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: fmt.Sprintf("Заказ %s не найден.", number)})
		return
	}
	input, ok := decodeOrderWriteRequest(w, r)
	if !ok {
		return
	}

	fromDepartmentID := user.DepartmentID
	order, err := s.orderSvc.UpdateOrder(r.Context(), orderuc.UpdateOrderInput{
		Number:            number,
		Items:             input.items,
		FromDepartmentID:  &fromDepartmentID,
		ToDepartmentID:    existing.ToDepartmentID,
		CreatedByUsername: miniAppOrderAuthor(user),
		FulfillmentDate:   input.fulfillmentDate,
	})
	if err != nil {
		slog.WarnContext(r.Context(), "mini app update order failed", "error", err, "telegram_id", user.TelegramID)
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Не удалось обновить заказ. Проверьте позиции и количества."})
		return
	}
	writeJSON(w, http.StatusOK, s.buildOrderResponse(r.Context(), order))
}

type validatedOrderWrite struct {
	items           []orderdomain.OrderItem
	fulfillmentDate time.Time
}

func decodeOrderWriteRequest(w http.ResponseWriter, r *http.Request) (validatedOrderWrite, bool) {
	var request orderWriteRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Проверьте формат заказа."})
		return validatedOrderWrite{}, false
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Проверьте формат заказа."})
		return validatedOrderWrite{}, false
	}
	if len(request.Items) == 0 || len(request.Items) > maxOrderItems {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Добавьте от 1 до 250 позиций заказа."})
		return validatedOrderWrite{}, false
	}

	fulfillmentDate, err := parseRequestDate(request.FulfillmentDate)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Дата выполнения должна быть в формате YYYY-MM-DD."})
		return validatedOrderWrite{}, false
	}
	items := make([]orderdomain.OrderItem, 0, len(request.Items))
	seen := make(map[string]struct{}, len(request.Items))
	for _, item := range request.Items {
		name := strings.TrimSpace(item.ProductName)
		key := strings.ToLower(name)
		if name == "" {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "У каждой позиции должно быть название блюда."})
			return validatedOrderWrite{}, false
		}
		if _, ok := seen[key]; ok {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: fmt.Sprintf("Позиция %q добавлена повторно.", name)})
			return validatedOrderWrite{}, false
		}
		seen[key] = struct{}{}
		if !validOrderQuantity(item.Quantity) || !validOrderQuantity(item.ReservedQuantity) ||
			item.Quantity+item.ReservedQuantity <= 0 {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: fmt.Sprintf("Укажите целое количество для позиции %q.", name)})
			return validatedOrderWrite{}, false
		}
		items = append(items, orderdomain.OrderItem{
			ProductName:      name,
			Quantity:         item.Quantity,
			ReservedQuantity: item.ReservedQuantity,
		})
	}
	return validatedOrderWrite{items: items, fulfillmentDate: fulfillmentDate}, true
}

func validOrderQuantity(quantity float64) bool {
	return quantity >= 0 && quantity <= maxItemQuantity && !math.IsNaN(quantity) && !math.IsInf(quantity, 0) &&
		quantity == math.Trunc(quantity)
}

func (s *Server) shopOrderWriter(w http.ResponseWriter, r *http.Request) (miniAppUser, bool) {
	user, ok := miniAppUserFromContext(r.Context())
	if !ok || !isShopMiniAppUser(user) {
		writeJSON(w, http.StatusForbidden, errorResponse{Error: "Только магазин может создавать и изменять заказы."})
		return miniAppUser{}, false
	}
	return user, true
}

func (s *Server) workshopDepartmentID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	department, err := s.departmentSvc.GetByCode(r.Context(), workshopDepartmentCode)
	if err != nil {
		slog.ErrorContext(r.Context(), "workshop department lookup failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "Цех не настроен. Не удалось создать заказ."})
		return 0, false
	}
	return department.ID, true
}

func miniAppOrderAuthor(user miniAppUser) string {
	if name := strings.TrimSpace(user.TelegramUser); name != "" {
		return name
	}
	if user.Auth.TelegramUsername != nil && strings.TrimSpace(*user.Auth.TelegramUsername) != "" {
		return strings.TrimSpace(*user.Auth.TelegramUsername)
	}
	if name := strings.TrimSpace(user.Auth.Username); name != "" {
		return name
	}
	return "telegram_" + strconv.FormatInt(user.TelegramID, 10)
}
