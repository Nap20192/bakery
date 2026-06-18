package access

import (
	"time"

	"bakery/internal/pkg/enum"
)

const (
	// RoleAdmin — полный доступ к заказам, калькуляции, iiko sync и управлению пользователями.
	RoleAdmin = string(enum.RoleAdmin)
)

// AuthUser — доменная модель пользователя авторизации.
// Используется middleware и сервисами для проверки роли и разрешений.
type AuthUser struct {
	ID               int64
	TelegramID       *int64
	TelegramUsername *string
	DepartmentID     *int64
	Username         string
	MetadataJSON     string
	Role             string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// PasswordAuthUserInput — входные данные для пользователя с username/password.
type PasswordAuthUserInput struct {
	Username         string
	Password         string
	TelegramUsername string
	DepartmentID     *int64
	MetadataJSON     string
	Role             string
}

// NormalizeRole приводит роль к каноничному виду (trim + lower-case),
// чтобы сравнение прав было стабильным независимо от регистра и пробелов.
func NormalizeRole(role string) string {
	return string(enum.NormalizeRole(role))
}

// IsValidRole проверяет, что роль входит в поддерживаемый набор RBAC.
func IsValidRole(role string) bool {
	return enum.IsValidRole(role)
}
