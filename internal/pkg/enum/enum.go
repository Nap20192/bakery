package enum

import "strings"

type Role string

const (
	RoleAdmin Role = "admin"
)

func NormalizeRole(role string) Role {
	return Role(strings.ToLower(strings.TrimSpace(role)))
}

func IsValidRole(role string) bool {
	switch NormalizeRole(role) {
	case RoleAdmin:
		return true
	default:
		return false
	}
}

type Permission string

const (
	PermissionTechCard       Permission = "techcard:view"
	PermissionSync           Permission = "sync:run"
	PermissionTemplateManage Permission = "template:manage"
)

type DepartmentType string

const (
	DepartmentTypeShop     DepartmentType = "shop"
	DepartmentTypeWorkshop DepartmentType = "workshop"
)

type IikoProductType string

const (
	IikoProductTypeDish     IikoProductType = "DISH"
	IikoProductTypePrepared IikoProductType = "PREPARED"
)

func IsIikoProductType(value string, productType IikoProductType) bool {
	return strings.EqualFold(strings.TrimSpace(value), string(productType))
}

type IikoSyncSource string

const (
	IikoSyncSourceGetAll IikoSyncSource = "getAll"
)

type SyncStatus string

const (
	SyncStatusRunning SyncStatus = "running"
	SyncStatusOK      SyncStatus = "ok"
	SyncStatusError   SyncStatus = "error"
)
