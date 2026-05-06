package app

import "bakery/internal/repo/sqlc"

type AdminService struct {
	queries *sqlc.Queries
}

func NewAdminService(queries *sqlc.Queries) *AdminService {
	return &AdminService{queries: queries}
}
