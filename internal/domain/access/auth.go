package access

import (
	"strings"
	"time"
)

const (
	// RoleAdmin — полный доступ: управление пользователями, группами и операциями по заказам.
	RoleAdmin = "admin"
	// RoleBaker — операционная роль производства: просмотр отчётов/заказов и изменение статуса заказов.
	RoleBaker = "baker"
	// RoleClient — клиентская роль: создание заказов.
	RoleClient = "client"
)

// AuthUser — доменная модель пользователя авторизации.
// Используется middleware и сервисами для проверки роли и разрешений.
type AuthUser struct {
	ID           int64
	TelegramID   *int64
	Username     string
	MetadataJSON string
	Role         string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// PasswordAuthUserInput — входные данные для пользователя с username/password.
type PasswordAuthUserInput struct {
	Username     string
	Password     string
	MetadataJSON string
	Role         string
}

// NormalizeRole приводит роль к каноничному виду (trim + lower-case),
// чтобы сравнение прав было стабильным независимо от регистра и пробелов.
func NormalizeRole(role string) string {
	return strings.ToLower(strings.TrimSpace(role))
}

// IsValidRole проверяет, что роль входит в поддерживаемый набор RBAC.
func IsValidRole(role string) bool {
	switch NormalizeRole(role) {
	case RoleAdmin, RoleBaker, RoleClient:
		return true
	default:
		return false
	}
}
