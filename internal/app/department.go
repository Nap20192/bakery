package app

import (
	"context"
	"strings"

	sqlc "bakery/internal/outbound/db/sqlc"
	"bakery/internal/pkg/enum"
)

const (
	DepartmentTypeShop     = enum.DepartmentTypeShop
	DepartmentTypeWorkshop = enum.DepartmentTypeWorkshop
)

type DepartmentService struct {
	queries *sqlc.Queries
}

func NewDepartmentService(queries *sqlc.Queries) *DepartmentService {
	return &DepartmentService{queries: queries}
}

func (s *DepartmentService) ListByType(ctx context.Context, departmentType enum.DepartmentType) ([]sqlc.Department, error) {
	value := strings.TrimSpace(string(departmentType))
	if value == "" {
		return s.queries.ListDepartments(ctx, nil)
	}
	return s.queries.ListDepartments(ctx, &value)
}

func (s *DepartmentService) GetByCode(ctx context.Context, code string) (sqlc.Department, error) {
	return s.queries.GetDepartmentByCode(ctx, strings.TrimSpace(code))
}

func (s *DepartmentService) GetByID(ctx context.Context, id int64) (sqlc.Department, error) {
	return s.queries.GetDepartmentByID(ctx, id)
}
