// Package authapp is the composition root of the auth service.
package authapp

import (
	sqlc "bakery/internal/outbound/db/sqlc"
	authrepo "bakery/internal/services/auth/infra/repo"
	authuc "bakery/internal/services/auth/usecase/auth"
)

func New(queries *sqlc.Queries) authuc.UseCase {
	return authuc.NewService(authrepo.New(queries))
}

func NewRBAC() *authuc.RBAC {
	return authuc.NewRBAC()
}
