// Package departmentuc is the application layer of the department service.
// It declares the boundary (UseCase) and persistence port (Repository) and a
// transport-agnostic Department value, so delivery never depends on sqlc types.
package departmentuc

import (
	"context"

	"bakery/internal/pkg/enum"
)

type UseCase interface {
	ListByType(ctx context.Context, departmentType enum.DepartmentType) ([]Department, error)
	GetByCode(ctx context.Context, code string) (Department, error)
	GetByID(ctx context.Context, id int64) (Department, error)
}

type Repository interface {
	ListByType(ctx context.Context, departmentType enum.DepartmentType) ([]Department, error)
	GetByCode(ctx context.Context, code string) (Department, error)
	GetByID(ctx context.Context, id int64) (Department, error)
}

type Department struct {
	ID   int64
	Code string
	Name string
	Type string
}
