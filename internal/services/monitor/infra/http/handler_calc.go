package monitorhttp

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"strings"
	"time"

	"bakery/internal/inbound/api/contract"
	"bakery/internal/inbound/api/httpx"
	orderdomain "bakery/internal/services/order/domain"
)

const (
	calcMaxItems    = 200
	calcMaxQuantity = 100000
)

// handleMonitorCalc — калькулятор теста: расход по введённым вручную позициям,
// без создания заказа или каких-либо записей. Синтетический «заказ» живёт
// только внутри запроса; коды теста берутся у выбранного типа заявки
// (category_id) или из дефолтного набора.
func (h *Handler) handleMonitorCalc(w http.ResponseWriter, r *http.Request) {
	if h.monitorSvc == nil || h.orderSvc == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "Сервис калькуляции временно недоступен.")
		return
	}

	var req contract.DoughCalcRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Некорректные данные калькулятора.")
		return
	}
	if len(req.Items) == 0 {
		httpx.WriteError(w, http.StatusBadRequest, "Добавьте хотя бы одну позицию с количеством.")
		return
	}
	if len(req.Items) > calcMaxItems {
		httpx.WriteError(w, http.StatusBadRequest, fmt.Sprintf("Слишком много позиций (до %d).", calcMaxItems))
		return
	}

	items := make([]orderdomain.OrderItem, 0, len(req.Items))
	for _, item := range req.Items {
		name := strings.TrimSpace(item.ProductName)
		code := strings.TrimSpace(item.Code)
		if code == "" {
			httpx.WriteError(w, http.StatusBadRequest, fmt.Sprintf("У позиции %q нет кода блюда.", name))
			return
		}
		if !(item.Quantity > 0) || math.IsInf(item.Quantity, 0) || item.Quantity > calcMaxQuantity {
			httpx.WriteError(w, http.StatusBadRequest, fmt.Sprintf("Количество для %q должно быть положительным числом.", name))
			return
		}
		items = append(items, orderdomain.OrderItem{Code: code, ProductName: name, Quantity: item.Quantity})
	}

	// Техкарты датированы — считаем по картам, действующим сегодня.
	now := time.Now().UTC()
	order := orderdomain.Order{Items: items, CreatedAt: now, FulfillmentDate: now}
	if req.CategoryID > 0 {
		categories, err := h.orderSvc.ListOrderCategories(r.Context())
		if err != nil {
			slog.ErrorContext(r.Context(), "calc: list categories failed", "error", err)
			httpx.WriteError(w, http.StatusInternalServerError, "Не удалось загрузить типы заявок.")
			return
		}
		for i := range categories {
			if categories[i].ID == req.CategoryID {
				order.Category = &categories[i]
				break
			}
		}
		if order.Category == nil {
			httpx.WriteError(w, http.StatusBadRequest, "Тип заявки не найден.")
			return
		}
	}
	codes, ok := h.monitorCodesFor(w, order)
	if !ok {
		return
	}

	reports := make([]contract.MonitorReport, 0, len(codes))
	for _, code := range codes {
		report, err := h.monitorSvc.GetIngredientsByCode(r.Context(), code, order)
		if err != nil {
			slog.WarnContext(r.Context(), "dough calculator failed", "code", code, "error", err)
			httpx.WriteError(w, http.StatusBadRequest, "Не удалось посчитать тесто. Проверьте позиции и техкарты.")
			return
		}
		reports = append(reports, contract.MonitorReport{Code: code, Report: report})
	}
	httpx.WriteJSON(w, http.StatusOK, contract.DoughCalcResponse{Reports: reports})
}
