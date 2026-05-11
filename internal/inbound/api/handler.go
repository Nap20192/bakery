package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	orderdomain "bakery/internal/domain/order"
	monitoringdomain "bakery/internal/domain/monitoring"
)

var defaultMonitorCodes = []string{"17642", "17644", "17650", "19694"}

type monitorReportResponse struct {
	Code   string                            `json:"code"`
	Report monitoringdomain.IngredientReport `json:"report"`
}
type monitorReportWithMetaData struct {
	Reports      []monitorReportResponse `json:"reports"`
	OrderID      string                  `json:"order_id"`
	LoactionFrom string                  `json:"loaction_from"`
	LoactionTo   string                  `json:"loaction_to"`
	Date         string                  `json:"date"`
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleListOrders(w http.ResponseWriter, r *http.Request) {
	if s.orderSvc == nil {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "order service unavailable"})
		return
	}

	limit := int32(10)
	if raw := trim(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid limit"})
			return
		}
		limit = int32(parsed)
	}

	orders, err := s.orderSvc.ListOrders(r.Context(), limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, orders)
}

func (s *Server) handleOrderByID(w http.ResponseWriter, r *http.Request) {
	if s.orderSvc == nil {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "order service unavailable"})
		return
	}

	id := trim(r.PathValue("id"))
	if id == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "order id is required"})
		return
	}

	order, err := s.orderSvc.GetOrderByNumber(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: fmt.Sprintf("order %s not found", id)})
		return
	}

	writeJSON(w, http.StatusOK, order)
}

func (s *Server) handleMonitorDefault(w http.ResponseWriter, r *http.Request) {
	if s.monitorSvc == nil || s.orderSvc == nil {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "monitor service unavailable"})
		return
	}

	orderID := trim(r.PathValue("id"))
	if orderID == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "order id is required"})
		return
	}

	order, err := s.orderSvc.GetOrderByNumber(r.Context(), orderID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: fmt.Sprintf("order %s not found", orderID)})
		return
	}

	reports := make([]monitorReportResponse, 0, len(defaultMonitorCodes))
	for _, code := range defaultMonitorCodes {
		report, err := s.monitorSvc.GetIngredientsByCode(r.Context(), code, order)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
			return
		}
		reports = append(reports, monitorReportResponse{Code: code, Report: report})
	}

	writeJSON(w, http.StatusOK, buildMonitorMetadata(order, reports))
}

func (s *Server) handleMonitorByProduct(w http.ResponseWriter, r *http.Request) {
	if s.monitorSvc == nil || s.orderSvc == nil {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "monitor service unavailable"})
		return
	}

	orderID := trim(r.PathValue("id"))
	productCode := trim(r.PathValue("product_id"))
	if orderID == "" || productCode == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "order id and product id are required"})
		return
	}

	order, err := s.orderSvc.GetOrderByNumber(r.Context(), orderID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: fmt.Sprintf("order %s not found", orderID)})
		return
	}

	report, err := s.monitorSvc.GetIngredientsByCode(r.Context(), productCode, order)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, buildMonitorMetadata(order, []monitorReportResponse{
		{Code: productCode, Report: report},
	}))
}

func buildMonitorMetadata(order orderdomain.Order, reports []monitorReportResponse) monitorReportWithMetaData {
	locationFrom := order.Location
	if order.FromDepartmentID != nil {
		locationFrom = strconv.FormatInt(*order.FromDepartmentID, 10)
	}
	locationTo := ""
	if order.ToDepartmentID != nil {
		locationTo = strconv.FormatInt(*order.ToDepartmentID, 10)
	}
	date := ""
	if !order.CreatedAt.IsZero() {
		date = order.CreatedAt.Format(time.RFC3339)
	}
	return monitorReportWithMetaData{
		Reports:      reports,
		OrderID:      order.Number,
		LoactionFrom: locationFrom,
		LoactionTo:   locationTo,
		Date:         date,
	}
}
