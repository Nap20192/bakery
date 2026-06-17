package departmentsvc

import (
	"context"
	"strings"

	sqlc "bakery/internal/outbound/db/sqlc"
	"bakery/internal/pkg/enum"
)

type Service struct {
	queries *sqlc.Queries
}

func New(queries *sqlc.Queries) *Service {
	return &Service{queries: queries}
}

func (s *Service) ListByType(ctx context.Context, departmentType enum.DepartmentType) ([]sqlc.Department, error) {
	value := strings.TrimSpace(string(departmentType))
	if value == "" {
		return s.queries.ListDepartments(ctx, nil)
	}
	return s.queries.ListDepartments(ctx, &value)
}

func (s *Service) GetByCode(ctx context.Context, code string) (sqlc.Department, error) {
	return s.queries.GetDepartmentByCode(ctx, strings.TrimSpace(code))
}

func (s *Service) GetByID(ctx context.Context, id int64) (sqlc.Department, error) {
	return s.queries.GetDepartmentByID(ctx, id)
}
