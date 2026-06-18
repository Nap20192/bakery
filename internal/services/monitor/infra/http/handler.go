// Package monitorhttp is the HTTP delivery adapter of the monitoring service.
package monitorhttp

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"bakery/internal/inbound/api/httpx"
	monitoringdomain "bakery/internal/services/monitor/domain"
	monitoruc "bakery/internal/services/monitor/usecase/monitor"
	orderdomain "bakery/internal/services/order/domain"
	orderhttp "bakery/internal/services/order/infra/http"
	orderuc "bakery/internal/services/order/usecase/order"
)

// Handler is the monitoring service HTTP delivery adapter. It reuses the order
// presenter to embed orders in calculation responses.
type Handler struct {
	monitorSvc monitoruc.UseCase
	orderSvc   orderuc.UseCase
	presenter  *orderhttp.OrderPresenter
}

func New(monitorSvc monitoruc.UseCase, orderSvc orderuc.UseCase, presenter *orderhttp.OrderPresenter) *Handler {
	return &Handler{monitorSvc: monitorSvc, orderSvc: orderSvc, presenter: presenter}
}

// RegisterRoutes wires the monitor endpoints behind Mini App authentication.
func (h *Handler) RegisterRoutes(mux *http.ServeMux, auth *httpx.Authenticator) {
	mux.Handle("GET /monitor/batch", auth.RequireMiniAppAuth(http.HandlerFunc(h.handleMonitorBatch)))
	mux.Handle("GET /monitor/{id}", auth.RequireMiniAppAuth(http.HandlerFunc(h.handleMonitorDefault)))
	mux.Handle("GET /monitor/{id}/{product_id}", auth.RequireMiniAppAuth(http.HandlerFunc(h.handleMonitorByProduct)))
}

type monitorReportResponse struct {
	Code   string                            `json:"code"`
	Report monitoringdomain.IngredientReport `json:"report"`
}

type monitorResponse struct {
	Reports []monitorReportResponse `json:"reports"`
	Order   orderhttp.OrderResponse `json:"order"`
}

type batchMonitorOrderResponse struct {
	Order   orderhttp.OrderResponse `json:"order"`
	Reports []monitorReportResponse `json:"reports"`
}

type batchMonitorResponse struct {
	Orders       []batchMonitorOrderResponse `json:"orders"`
	TotalReports []monitorReportResponse     `json:"total_reports"`
}

func (h *Handler) handleMonitorDefault(w http.ResponseWriter, r *http.Request) {
	if h.monitorSvc == nil || h.orderSvc == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "Сервис калькуляции временно недоступен.")
		return
	}

	orderID := httpx.Trim(r.PathValue("id"))
	if orderID == "" {
		httpx.WriteError(w, http.StatusBadRequest, "Укажите номер заказа.")
		return
	}

	order, err := h.orderSvc.GetOrderByNumber(r.Context(), orderID)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, fmt.Sprintf("Заказ %s не найден.", orderID))
		return
	}
	if !orderhttp.UserCanReadOrder(r.Context(), order) {
		httpx.WriteError(w, http.StatusNotFound, fmt.Sprintf("Заказ %s не найден.", orderID))
		return
	}

	reports := make([]monitorReportResponse, 0, len(monitoringdomain.DefaultMonitorCodes))
	for _, code := range monitoringdomain.DefaultMonitorCodes {
		report, err := h.monitorSvc.GetIngredientsByCode(r.Context(), code, order)
		if err != nil {
			slog.WarnContext(r.Context(), "default monitor calculation failed", "order_number", orderID, "code", code, "error", err)
			httpx.WriteError(w, http.StatusBadRequest, "Не удалось посчитать калькуляцию. Проверьте заказ и техкарты.")
			return
		}
		reports = append(reports, monitorReportResponse{Code: code, Report: report})
	}

	httpx.WriteJSON(w, http.StatusOK, h.buildMonitorResponse(r.Context(), order, reports))
}

func (h *Handler) handleMonitorByProduct(w http.ResponseWriter, r *http.Request) {
	if h.monitorSvc == nil || h.orderSvc == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "Сервис калькуляции временно недоступен.")
		return
	}

	orderID := httpx.Trim(r.PathValue("id"))
	productCode := httpx.Trim(r.PathValue("product_id"))
	if orderID == "" || productCode == "" {
		httpx.WriteError(w, http.StatusBadRequest, "Укажите номер заказа и код продукта.")
		return
	}

	order, err := h.orderSvc.GetOrderByNumber(r.Context(), orderID)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, fmt.Sprintf("Заказ %s не найден.", orderID))
		return
	}
	if !orderhttp.UserCanReadOrder(r.Context(), order) {
		httpx.WriteError(w, http.StatusNotFound, fmt.Sprintf("Заказ %s не найден.", orderID))
		return
	}

	report, err := h.monitorSvc.GetIngredientsByCode(r.Context(), productCode, order)
	if err != nil {
		slog.WarnContext(r.Context(), "monitor calculation failed", "order_number", orderID, "code", productCode, "error", err)
		httpx.WriteError(w, http.StatusBadRequest, "Не удалось посчитать калькуляцию по этому продукту. Проверьте код продукта и техкарту.")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, h.buildMonitorResponse(r.Context(), order, []monitorReportResponse{
		{Code: productCode, Report: report},
	}))
}

func (h *Handler) handleMonitorBatch(w http.ResponseWriter, r *http.Request) {
	if h.monitorSvc == nil || h.orderSvc == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "Сервис калькуляции временно недоступен.")
		return
	}

	orderNumbers := httpx.UniqueQueryValues(r.URL.Query()["orders"])
	if len(orderNumbers) == 0 {
		httpx.WriteError(w, http.StatusBadRequest, "Укажите хотя бы один заказ в параметре orders.")
		return
	}
	if len(orderNumbers) > 50 {
		httpx.WriteError(w, http.StatusBadRequest, "Слишком много заказов. Выберите не больше 50.")
		return
	}

	orders := make([]orderdomain.Order, 0, len(orderNumbers))
	for _, number := range orderNumbers {
		order, err := h.orderSvc.GetOrderByNumber(r.Context(), number)
		if err != nil {
			httpx.WriteError(w, http.StatusNotFound, fmt.Sprintf("Заказ %s не найден.", number))
			return
		}
		if !orderhttp.UserCanReadOrder(r.Context(), order) {
			httpx.WriteError(w, http.StatusNotFound, fmt.Sprintf("Заказ %s не найден.", number))
			return
		}
		orders = append(orders, order)
	}

	report, err := h.monitorSvc.GetBatchIngredientsByCodes(r.Context(), monitoringdomain.DefaultMonitorCodes, orders)
	if err != nil {
		slog.WarnContext(r.Context(), "batch monitor calculation failed", "orders", orderNumbers, "error", err)
		httpx.WriteError(w, http.StatusBadRequest, "Не удалось посчитать калькуляцию по выбранным заказам. Проверьте заказы и техкарты.")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, h.buildBatchMonitorResponse(r.Context(), orders, report))
}

func (h *Handler) buildMonitorResponse(ctx context.Context, order orderdomain.Order, reports []monitorReportResponse) monitorResponse {
	return monitorResponse{
		Reports: reports,
		Order:   h.presenter.BuildOrderResponse(ctx, order),
	}
}

func (h *Handler) buildBatchMonitorResponse(
	ctx context.Context,
	orders []orderdomain.Order,
	report monitoringdomain.BatchMonitoringReport,
) batchMonitorResponse {
	orderByNumber := make(map[string]orderdomain.Order, len(orders))
	for _, order := range orders {
		orderByNumber[order.Number] = order
	}

	response := batchMonitorResponse{
		Orders:       make([]batchMonitorOrderResponse, 0, len(report.Orders)),
		TotalReports: make([]monitorReportResponse, 0, len(report.TotalReports)),
	}
	for i, total := range report.TotalReports {
		code := ""
		if i < len(monitoringdomain.DefaultMonitorCodes) {
			code = monitoringdomain.DefaultMonitorCodes[i]
		}
		response.TotalReports = append(response.TotalReports, monitorReportResponse{
			Code:   code,
			Report: total,
		})
	}
	for _, orderReport := range report.Orders {
		order := orderByNumber[orderReport.OrderNumber]
		reports := make([]monitorReportResponse, 0, len(orderReport.Reports))
		for i, item := range orderReport.Reports {
			code := ""
			if i < len(monitoringdomain.DefaultMonitorCodes) {
				code = monitoringdomain.DefaultMonitorCodes[i]
			}
			reports = append(reports, monitorReportResponse{
				Code:   code,
				Report: item,
			})
		}
		response.Orders = append(response.Orders, batchMonitorOrderResponse{
			Order:   h.presenter.BuildOrderResponse(ctx, order),
			Reports: reports,
		})
	}
	return response
}
