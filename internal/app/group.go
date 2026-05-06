package app

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"bakery/internal/domain"
	"bakery/internal/repo/sqlc"
)

type GroupService struct {
	queries *sqlc.Queries
}

func NewGroupService(queries *sqlc.Queries) *GroupService {
	return &GroupService{queries: queries}
}

func (s *GroupService) AddGroupByProductCode(ctx context.Context, input domain.GroupInput) (domain.Group, error) {
	input.Code = strings.TrimSpace(input.Code)
	if input.Code == "" {
		return domain.Group{}, fmt.Errorf("code is required")
	}

	product, err := s.queries.GetIikoProductByCode(ctx, input.Code)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.Group{}, fmt.Errorf("product with code %s not found", input.Code)
		}
		return domain.Group{}, fmt.Errorf("get product by code: %w", err)
	}

	group, err := s.queries.UpsertGroup(ctx, sqlc.UpsertGroupParams{
		Code: product.Code,
		Name: product.Name,
	})
	if err != nil {
		return domain.Group{}, fmt.Errorf("upsert group: %w", err)
	}

	return domain.Group{
		ID:   product.ID,
		Code: group.Code,
		Name: group.Name,
	}, nil
}

func (s *GroupService) ListGroups(ctx context.Context) ([]domain.Group, error) {
	rows, err := s.queries.ListGroups(ctx)
	if err != nil {
		return nil, fmt.Errorf("list groups: %w", err)
	}
	groups := make([]domain.Group, 0, len(rows))
	for _, row := range rows {
		groups = append(groups, domain.Group{
			Code: row.Code,
			Name: row.Name,
		})
	}
	return groups, nil
}

func (s *GroupService) GetGroupByCode(ctx context.Context, code string) (domain.Group, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return domain.Group{}, fmt.Errorf("code is required")
	}
	row, err := s.queries.GetGroupByCode(ctx, code)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.Group{}, fmt.Errorf("group with code %s not found", code)
		}
		return domain.Group{}, fmt.Errorf("get group by code: %w", err)
	}
	return domain.Group{
		Code: row.Code,
		Name: row.Name,
	}, nil
}
