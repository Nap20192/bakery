package app

import (
	"bakery/internal/repo/sqlc"
	"context"
)

type AdminService struct {
	queries *sqlc.Queries
}

func NewAdminService(queries *sqlc.Queries) *AdminService {
	return &AdminService{queries: queries}
}

func (s *AdminService) AddGroup(ctx context.Context, name string) error {
	return nil
}
