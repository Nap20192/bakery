// Package adminapp is the composition root of the admin service. It adapts the
// auth and department services to the admin ports.
package adminapp

import (
	"context"

	adminuc "bakery/internal/services/admin/usecase/admin"
	authuc "bakery/internal/services/auth/usecase/auth"
	departmentuc "bakery/internal/services/department/usecase/department"
)

func New(users authuc.UseCase, departments departmentuc.UseCase) adminuc.UseCase {
	return adminuc.NewService(users, departmentsAdapter{svc: departments})
}

// departmentsAdapter adapts the department service to admin's Departments port.
type departmentsAdapter struct {
	svc departmentuc.UseCase
}

func (a departmentsAdapter) GetByCode(ctx context.Context, code string) (adminuc.Department, error) {
	d, err := a.svc.GetByCode(ctx, code)
	if err != nil {
		return adminuc.Department{}, err
	}
	return adminuc.Department{ID: d.ID, Code: d.Code, Name: d.Name, Type: d.Type}, nil
}

func (a departmentsAdapter) ListAll(ctx context.Context) ([]adminuc.Department, error) {
	rows, err := a.svc.ListByType(ctx, "")
	if err != nil {
		return nil, err
	}
	out := make([]adminuc.Department, 0, len(rows))
	for _, d := range rows {
		out = append(out, adminuc.Department{ID: d.ID, Code: d.Code, Name: d.Name, Type: d.Type})
	}
	return out, nil
}
