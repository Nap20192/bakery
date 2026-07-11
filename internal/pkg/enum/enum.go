package enum

import "strings"

type Role string

const (
	// RoleAdmin — полный доступ: iiko sync, управление пользователями, техкарты,
	// все заказы и мониторинг.
	RoleAdmin Role = "admin"
	// RoleShop — магазин: создаёт и редактирует заказы, видит только заказы
	// своего магазина.
	RoleShop Role = "shop"
	// RoleBaker — цех: видит все заказы и считает мониторинг по тесту.
	RoleBaker Role = "baker"
	// RoleUser — роль по умолчанию для нового пользователя без выданных прав.
	RoleUser Role = "user"
)

func NormalizeRole(role string) Role {
	return Role(strings.ToLower(strings.TrimSpace(role)))
}

func IsValidRole(role string) bool {
	switch NormalizeRole(role) {
	case RoleAdmin, RoleShop, RoleBaker, RoleUser:
		return true
	default:
		return false
	}
}

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
