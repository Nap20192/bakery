package app

import (
	"strings"

	"bakery/internal/pkg/enum"
)

const (
	PermissionTechCard       = enum.PermissionTechCard
	PermissionSync           = enum.PermissionSync
	PermissionTemplateManage = enum.PermissionTemplateManage
)

type RbacService struct {
	rolePermissions map[enum.Role]map[enum.Permission]struct{}
}

func NewRbacService() *RbacService {
	return &RbacService{
		rolePermissions: map[enum.Role]map[enum.Permission]struct{}{
			enum.RoleAdmin: {
				PermissionTechCard:       {},
				PermissionSync:           {},
				PermissionTemplateManage: {},
			},
		},
	}
}

func (s *RbacService) HasPermission(role string, permission enum.Permission) bool {
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
