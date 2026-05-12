package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	monitoringdomain "bakery/internal/domain/monitoring"
	orderdomain "bakery/internal/domain/order"
)

var defaultMonitorCodes = []string{"17642", "17644", "17650", "19694"}

type departmentResponse struct {
	ID   int64  `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type orderItemResponse struct {
	Code               string  `json:"code"`
	ProductName        string  `json:"product_name"`
	Quantity           float64 `json:"quantity"`
	ReservedQuantity   float64 `json:"reserved_quantity"`
	ProductionQuantity float64 `json:"production_quantity"`
}

type orderResponse struct {
	ID              string              `json:"id"`
	Number          string              `json:"number"`
	Location        string              `json:"location"`
	FromDepartment  *departmentResponse `json:"from_department,omitempty"`
	ToDepartment    *departmentResponse `json:"to_department,omitempty"`
	Items           []orderItemResponse `json:"items"`
	CreatedAt       string              `json:"created_at"`
	FulfillmentDate string              `json:"fulfillment_date"`
	MonitorCommand  string              `json:"monitor_command"`
}

type monitorReportResponse struct {
	Code   string                            `json:"code"`
	Report monitoringdomain.IngredientReport `json:"report"`
}
type monitorResponse struct {
	Reports []monitorReportResponse `json:"reports"`
	Order   orderResponse           `json:"order"`
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

	writeJSON(w, http.StatusOK, s.buildOrderResponses(r.Context(), orders))
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

	writeJSON(w, http.StatusOK, s.buildOrderResponse(r.Context(), order))
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

	writeJSON(w, http.StatusOK, s.buildMonitorResponse(r.Context(), order, reports))
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

	writeJSON(w, http.StatusOK, s.buildMonitorResponse(r.Context(), order, []monitorReportResponse{
		{Code: productCode, Report: report},
	}))
}

func (s *Server) buildMonitorResponse(ctx context.Context, order orderdomain.Order, reports []monitorReportResponse) monitorResponse {
	return monitorResponse{
		Reports: reports,
		Order:   s.buildOrderResponse(ctx, order),
	}
}

func (s *Server) buildOrderResponses(ctx context.Context, orders []orderdomain.Order) []orderResponse {
	responses := make([]orderResponse, 0, len(orders))
	for _, order := range orders {
		responses = append(responses, s.buildOrderResponse(ctx, order))
	}
	return responses
}

func (s *Server) buildOrderResponse(ctx context.Context, order orderdomain.Order) orderResponse {
	items := make([]orderItemResponse, 0, len(order.Items))
	for _, item := range order.Items {
		items = append(items, orderItemResponse{
			Code:               item.Code,
			ProductName:        item.ProductName,
			Quantity:           item.Quantity,
			ReservedQuantity:   item.ReservedQuantity,
			ProductionQuantity: item.ProductionQuantity(),
		})
	}

	createdAt := ""
	if !order.CreatedAt.IsZero() {
		createdAt = order.CreatedAt.Format(time.RFC3339)
	}
	fulfillmentDate := ""
	if !order.FulfillmentDate.IsZero() {
		fulfillmentDate = order.FulfillmentDate.Format("2006-01-02")
	}

	return orderResponse{
		ID:              order.ID,
		Number:          order.Number,
		Location:        order.Location,
		FromDepartment:  s.departmentResponse(ctx, order.FromDepartmentID),
		ToDepartment:    s.departmentResponse(ctx, order.ToDepartmentID),
		Items:           items,
		CreatedAt:       createdAt,
		FulfillmentDate: fulfillmentDate,
		MonitorCommand:  fmt.Sprintf("/monitor %s", order.Number),
	}
}

func (s *Server) departmentResponse(ctx context.Context, id *int64) *departmentResponse {
	if id == nil || s.departmentSvc == nil {
		return nil
	}
	department, err := s.departmentSvc.GetByID(ctx, *id)
	if err != nil {
		return &departmentResponse{ID: *id, Name: strconv.FormatInt(*id, 10)}
	}
	return &departmentResponse{
		ID:   department.ID,
		Code: department.Code,
		Name: department.Name,
		Type: department.Type,
	}
}
