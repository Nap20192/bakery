package authuc

import (
	"strings"

	"bakery/internal/pkg/enum"
)

// RBAC is the stateless role→permission policy of the auth service.
type RBAC struct {
	rolePermissions map[enum.Role]map[enum.Permission]struct{}
}

func NewRBAC() *RBAC {
	return &RBAC{
		rolePermissions: map[enum.Role]map[enum.Permission]struct{}{
			// admin — всё.
			enum.RoleAdmin: {
				enum.PermissionSync:         {},
				enum.PermissionTechCard:     {},
				enum.PermissionManageUsers:  {},
				enum.PermissionOrderWrite:   {},
				enum.PermissionOrderViewAll: {},
				enum.PermissionMonitorDough: {},
			},
			// shop — создаёт/редактирует заказы; нет order:view_all, поэтому
			// видимость ограничивается своим магазином (по department_id).
			enum.RoleShop: {
				enum.PermissionOrderWrite: {},
			},
			// baker — видит все заказы и считает мониторинг по тесту.
			enum.RoleBaker: {
				enum.PermissionOrderViewAll: {},
				enum.PermissionMonitorDough: {},
			},
		},
	}
}

func (s *RBAC) HasPermission(role string, permission enum.Permission) bool {
	if s == nil {
		return false
	}
	normalizedRole := enum.NormalizeRole(role)
	normalizedPermission := enum.Permission(strings.TrimSpace(string(permission)))
	perms, ok := s.rolePermissions[normalizedRole]
	if !ok {
		return false
	}
	_, ok = perms[normalizedPermission]
	return ok
}
