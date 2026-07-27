package orderrepo

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	sqlc "bakery/internal/outbound/db/sqlc"
	"bakery/internal/pkg/helpers"
	orderdomain "bakery/internal/services/order/domain"
	orderuc "bakery/internal/services/order/usecase/order"
)

func (r *OrderRepository) SaveOrderDraft(ctx context.Context, input orderuc.SaveOrderDraftRepositoryInput) (orderdomain.OrderDraft, error) {
	fromDepartmentID := input.FromDepartmentID
	row, err := r.queries.UpsertOrderDraft(ctx, sqlc.UpsertOrderDraftParams{
		CreatedByUsername: strings.TrimSpace(input.CreatedByUsername),
		CategoryID:        input.CategoryID,
		FromDepartmentID:  &fromDepartmentID,
		FulfillmentDate:   helpers.DateOf(input.FulfillmentDate),
		Comments:          marshalComments(input.Comments),
		Items:             marshalDraftItems(input.Items),
	})
	if err != nil {
		return orderdomain.OrderDraft{}, fmt.Errorf("save order draft: %w", err)
	}
	return draftFromRow(row), nil
}

func (r *OrderRepository) GetOrderDraft(ctx context.Context, username string, categoryID int64) (orderdomain.OrderDraft, error) {
	row, err := r.queries.GetOrderDraft(ctx, sqlc.GetOrderDraftParams{CreatedByUsername: username, CategoryID: categoryID})
	if err != nil {
		return orderdomain.OrderDraft{}, err
	}
	return draftFromRow(row), nil
}

func (r *OrderRepository) ListOrderDrafts(ctx context.Context, username string) ([]orderdomain.OrderDraft, error) {
	rows, err := r.queries.ListOrderDrafts(ctx, username)
	if err != nil {
		return nil, err
	}
	drafts := make([]orderdomain.OrderDraft, 0, len(rows))
	for _, row := range rows {
		drafts = append(drafts, draftFromRow(row))
	}
	return drafts, nil
}

func (r *OrderRepository) DeleteOrderDraft(ctx context.Context, username string, categoryID int64) error {
	return r.queries.DeleteOrderDraft(ctx, sqlc.DeleteOrderDraftParams{CreatedByUsername: username, CategoryID: categoryID})
}

func (r *OrderRepository) DeleteOrderDraftsOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	return r.queries.DeleteOrderDraftsOlderThan(ctx, helpers.Timestamptz(cutoff))
}

func draftFromRow(row sqlc.OrderDraft) orderdomain.OrderDraft {
	draft := orderdomain.OrderDraft{
		CreatedByUsername: row.CreatedByUsername,
		CategoryID:        row.CategoryID,
		FromDepartmentID:  row.FromDepartmentID,
		Items:             parseDraftItems(row.Items),
		Comments:          parseComments(row.Comments),
		UpdatedAt:         row.UpdatedAt.Time,
	}
	if row.FulfillmentDate.Valid {
		draft.FulfillmentDate = row.FulfillmentDate.Time
	}
	return draft
}

// marshalDraftItems/parseDraftItems store a draft's items as a JSON array —
// unlike real orders, a draft has no order_items rows of its own.
func marshalDraftItems(items []orderdomain.OrderItem) []byte {
	data, err := json.Marshal(items)
	if err != nil {
		return []byte("[]")
	}
	return data
}

func parseDraftItems(raw []byte) []orderdomain.OrderItem {
	if len(raw) == 0 {
		return nil
	}
	var items []orderdomain.OrderItem
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil
	}
	return items
}
