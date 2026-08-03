// Package orderhttp is the HTTP delivery adapter of the order service.
package orderhttp

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"bakery/internal/inbound/api/contract"
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

func buildCommentsResponse(comments orderdomain.OrderComments) contract.Comments {
	out := contract.Comments{General: comments.General, Items: make([]contract.ItemComment, 0, len(comments.Items))}
	for _, c := range comments.Items {
		out.Items = append(out.Items, contract.ItemComment{ProductName: c.ProductName, Comment: c.Comment})
	}
	return out
}

func (p *OrderPresenter) BuildOrderResponses(ctx context.Context, orders []orderdomain.Order) []contract.Order {
	responses := make([]contract.Order, 0, len(orders))
	for _, order := range orders {
		responses = append(responses, p.BuildOrderResponse(ctx, order))
	}
	return responses
}

func (p *OrderPresenter) BuildOrderResponse(ctx context.Context, order orderdomain.Order) contract.Order {
	items := make([]contract.OrderItem, 0, len(order.Items))
	for _, item := range order.Items {
		items = append(items, contract.OrderItem{
			Code:               item.Code,
			ProductName:        item.ProductName,
			Quantity:           item.Quantity,
			ReservedQuantity:   item.ReservedQuantity,
			ProductionQuantity: item.ProductionQuantity(),
			ProducedQuantity:   item.ProducedQuantity,
			ProducedReason:     item.ProducedReason,
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

	return contract.Order{
		ID:                  order.ID,
		Number:              order.Number,
		Location:            order.Location,
		CreatedByUsername:   order.CreatedByUsername,
		FromDepartment:      p.departmentResponse(ctx, order.FromDepartmentID),
		ToDepartment:        p.departmentResponse(ctx, order.ToDepartmentID),
		Category:            buildCategoryResponse(order.Category),
		Items:               items,
		CreatedAt:           createdAt,
		FulfillmentDate:     fulfillmentDate,
		MonitorCommand:      fmt.Sprintf("/monitor %s", order.Number),
		Comments:            buildCommentsResponse(order.Comments),
		Favorite:            order.Favorite,
		Cancelled:           order.Cancelled,
		CancelledByUsername: order.CancelledByUsername,
		ProductionSheetID:   order.ProductionSheetID,
		History:             buildOrderHistoryResponse(order.History),
	}
}

func buildOrderHistoryResponse(history []orderdomain.OrderHistory) []contract.HistoryEntry {
	result := make([]contract.HistoryEntry, 0, len(history))
	for _, row := range history {
		items := make([]contract.HistoryItem, 0, len(row.Items))
		for _, item := range row.Items {
			items = append(items, contract.HistoryItem{
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
		result = append(result, contract.HistoryEntry{
			ID:                row.ID,
			ChangedByUsername: row.ChangedByUsername,
			ChangedAt:         changedAt,
			Items:             items,
		})
	}
	return result
}

func buildCategoryResponse(category *orderdomain.OrderCategory) *contract.Category {
	if category == nil {
		return nil
	}
	return &contract.Category{
		ID:           category.ID,
		Code:         category.Code,
		Letter:       category.Letter,
		Name:         category.Name,
		Color:        category.Color,
		SortOrder:    category.SortOrder,
		MonitorCodes: category.MonitorCodes,
	}
}

func (p *OrderPresenter) departmentResponse(ctx context.Context, id *int64) *contract.Department {
	if id == nil || p.departmentSvc == nil {
		return nil
	}
	department, err := p.departmentSvc.GetByID(ctx, *id)
	if err != nil {
		return &contract.Department{ID: *id, Name: strconv.FormatInt(*id, 10)}
	}
	return &contract.Department{
		ID:   department.ID,
		Code: department.Code,
		Name: department.Name,
		Type: department.Type,
	}
}
