// Package orderhttp is the HTTP delivery adapter of the order service.
package orderhttp

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"bakery/internal/inbound/api/httpx"
	departmentuc "bakery/internal/services/department/usecase/department"
	orderdomain "bakery/internal/services/order/domain"
)

// OrderPresenter projects order domain objects to API responses, resolving
// referenced departments. It is shared with the monitor adapter, which embeds
// orders in its calculation responses.
type OrderPresenter struct {
	departmentSvc departmentuc.UseCase
}

func NewPresenter(departmentSvc departmentuc.UseCase) *OrderPresenter {
	return &OrderPresenter{departmentSvc: departmentSvc}
}

type orderItemResponse struct {
	Code               string  `json:"code"`
	ProductName        string  `json:"product_name"`
	Quantity           float64 `json:"quantity"`
	ReservedQuantity   float64 `json:"reserved_quantity"`
	ProductionQuantity float64 `json:"production_quantity"`
}

type orderHistoryItemResponse struct {
	ChangeType          string   `json:"change_type"`
	ProductCode         string   `json:"product_code"`
	ProductName         string   `json:"product_name"`
	OldQuantity         *float64 `json:"old_quantity,omitempty"`
	NewQuantity         *float64 `json:"new_quantity,omitempty"`
	OldReservedQuantity *float64 `json:"old_reserved_quantity,omitempty"`
	NewReservedQuantity *float64 `json:"new_reserved_quantity,omitempty"`
}

type orderHistoryResponse struct {
	ID                int64                      `json:"id"`
	ChangedByUsername string                     `json:"changed_by_username"`
	ChangedAt         string                     `json:"changed_at"`
	Items             []orderHistoryItemResponse `json:"items"`
}

// OrderResponse is the API projection of an order. Exported so other adapters
// (monitor) can embed it.
type OrderResponse struct {
	ID                string                    `json:"id"`
	Number            string                    `json:"number"`
	Location          string                    `json:"location"`
	CreatedByUsername string                    `json:"created_by_username"`
	FromDepartment    *httpx.DepartmentResponse `json:"from_department,omitempty"`
	ToDepartment      *httpx.DepartmentResponse `json:"to_department,omitempty"`
	Items             []orderItemResponse       `json:"items"`
	CreatedAt         string                    `json:"created_at"`
	FulfillmentDate   string                    `json:"fulfillment_date"`
	MonitorCommand    string                    `json:"monitor_command"`
	Comments          commentsResponse          `json:"comments"`
	Favorite          bool                      `json:"favorite"`
	History           []orderHistoryResponse    `json:"history,omitempty"`
}

type commentsResponse struct {
	General string                `json:"general"`
	Items   []itemCommentResponse `json:"items"`
}

type itemCommentResponse struct {
	ProductName string `json:"product_name"`
	Comment     string `json:"comment"`
}

func buildCommentsResponse(comments orderdomain.OrderComments) commentsResponse {
	out := commentsResponse{General: comments.General, Items: make([]itemCommentResponse, 0, len(comments.Items))}
	for _, c := range comments.Items {
		out.Items = append(out.Items, itemCommentResponse{ProductName: c.ProductName, Comment: c.Comment})
	}
	return out
}

type ordersPageResponse struct {
	Items      []OrderResponse `json:"items"`
	Page       int32           `json:"page"`
	Limit      int32           `json:"limit"`
	Offset     int32           `json:"offset"`
	Total      int64           `json:"total"`
	TotalPages int32           `json:"total_pages"`
}

func (p *OrderPresenter) BuildOrderResponses(ctx context.Context, orders []orderdomain.Order) []OrderResponse {
	responses := make([]OrderResponse, 0, len(orders))
	for _, order := range orders {
		responses = append(responses, p.BuildOrderResponse(ctx, order))
	}
	return responses
}

func (p *OrderPresenter) BuildOrderResponse(ctx context.Context, order orderdomain.Order) OrderResponse {
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

	return OrderResponse{
		ID:                order.ID,
		Number:            order.Number,
		Location:          order.Location,
		CreatedByUsername: order.CreatedByUsername,
		FromDepartment:    p.departmentResponse(ctx, order.FromDepartmentID),
		ToDepartment:      p.departmentResponse(ctx, order.ToDepartmentID),
		Items:             items,
		CreatedAt:         createdAt,
		FulfillmentDate:   fulfillmentDate,
		MonitorCommand:    fmt.Sprintf("/monitor %s", order.Number),
		Comments:          buildCommentsResponse(order.Comments),
		Favorite:          order.Favorite,
		History:           buildOrderHistoryResponse(order.History),
	}
}

func buildOrderHistoryResponse(history []orderdomain.OrderHistory) []orderHistoryResponse {
	result := make([]orderHistoryResponse, 0, len(history))
	for _, row := range history {
		items := make([]orderHistoryItemResponse, 0, len(row.Items))
		for _, item := range row.Items {
			items = append(items, orderHistoryItemResponse{
				ChangeType:          item.ChangeType,
				ProductCode:         item.ProductCode,
				ProductName:         item.ProductName,
				OldQuantity:         item.OldQuantity,
				NewQuantity:         item.NewQuantity,
				OldReservedQuantity: item.OldReservedQuantity,
				NewReservedQuantity: item.NewReservedQuantity,
			})
		}
		changedAt := ""
		if !row.ChangedAt.IsZero() {
			changedAt = row.ChangedAt.Format(time.RFC3339)
		}
		result = append(result, orderHistoryResponse{
			ID:                row.ID,
			ChangedByUsername: row.ChangedByUsername,
			ChangedAt:         changedAt,
			Items:             items,
		})
	}
	return result
}

func (p *OrderPresenter) departmentResponse(ctx context.Context, id *int64) *httpx.DepartmentResponse {
	if id == nil || p.departmentSvc == nil {
		return nil
	}
	department, err := p.departmentSvc.GetByID(ctx, *id)
	if err != nil {
		return &httpx.DepartmentResponse{ID: *id, Name: strconv.FormatInt(*id, 10)}
	}
	return &httpx.DepartmentResponse{
		ID:   department.ID,
		Code: department.Code,
		Name: department.Name,
		Type: department.Type,
	}
}
