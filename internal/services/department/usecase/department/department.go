package departmentuc

import (
	"context"
	"strings"

	"bakery/internal/pkg/enum"
)

// Service is the department use case. It depends only on the Repository port.
type Service struct {
	repo Repository
}

var _ UseCase = (*Service)(nil)

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListByType(ctx context.Context, departmentType enum.DepartmentType) ([]Department, error) {
	return s.repo.ListByType(ctx, departmentType)
}

func (s *Service) GetByCode(ctx context.Context, code string) (Department, error) {
	return s.repo.GetByCode(ctx, strings.TrimSpace(code))
}

func (s *Service) GetByID(ctx context.Context, id int64) (Department, error) {
	return s.repo.GetByID(ctx, id)
}
