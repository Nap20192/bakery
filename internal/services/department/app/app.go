// Package departmentapp is the composition root of the department service.
package departmentapp

import (
	sqlc "bakery/internal/outbound/db/sqlc"
	departmentrepo "bakery/internal/services/department/infra/repo"
	departmentuc "bakery/internal/services/department/usecase/department"
)

func New(queries *sqlc.Queries) departmentuc.UseCase {
	return departmentuc.NewService(departmentrepo.New(queries))
}
