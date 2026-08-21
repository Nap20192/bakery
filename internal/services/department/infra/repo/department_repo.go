// Package departmentrepo is the persistence adapter of the department service.
// It implements departmentuc.Repository over sqlc and maps rows to the
// transport-agnostic departmentuc.Department.
package departmentrepo

import (
	"context"
	"fmt"
	"strings"

	sqlc "bakery/internal/outbound/db/sqlc"
	"bakery/internal/pkg/enum"
	departmentuc "bakery/internal/services/department/usecase/department"
)

type DepartmentRepository struct {
	queries *sqlc.Queries
}

var _ departmentuc.Repository = (*DepartmentRepository)(nil)

func New(queries *sqlc.Queries) *DepartmentRepository {
	return &DepartmentRepository{queries: queries}
}

func (r *DepartmentRepository) ListByType(ctx context.Context, departmentType enum.DepartmentType) ([]departmentuc.Department, error) {
	value := strings.TrimSpace(string(departmentType))
	var (
		rows []sqlc.Department
		err  error
	)
	if value == "" {
		rows, err = r.queries.ListDepartments(ctx, nil)
	} else {
		rows, err = r.queries.ListDepartments(ctx, &value)
	}
	if err != nil {
		return nil, fmt.Errorf("list departments: %w", err)
	}
	out := make([]departmentuc.Department, 0, len(rows))
	for _, row := range rows {
		out = append(out, toDepartment(row))
	}
	return out, nil
}

func (r *DepartmentRepository) GetByCode(ctx context.Context, code string) (departmentuc.Department, error) {
	row, err := r.queries.GetDepartmentByCode(ctx, strings.TrimSpace(code))
	if err != nil {
		return departmentuc.Department{}, fmt.Errorf("get department by code: %w", err)
	}
	return toDepartment(row), nil
}

func (r *DepartmentRepository) GetByID(ctx context.Context, id int64) (departmentuc.Department, error) {
	row, err := r.queries.GetDepartmentByID(ctx, id)
	if err != nil {
		return departmentuc.Department{}, fmt.Errorf("get department by id: %w", err)
	}
	return toDepartment(row), nil
}

func toDepartment(d sqlc.Department) departmentuc.Department {
	return departmentuc.Department{ID: d.ID, Code: d.Code, Name: d.Name, Type: d.Type}
}
