package application

import (
	"errors"
	"fmt"
	"strings"

	"bakery/internal/inbound/api/contract"
)

// Credentials is the Authorization header value used for a backend request.
type Credentials string

type Error struct {
	Status  int
	Message string
}

func (e *Error) Error() string {
	return fmt.Sprintf("backend: %d: %s", e.Status, e.Message)
}

func StatusOf(err error) int {
	var backendErr *Error
	if errors.As(err, &backendErr) {
		return backendErr.Status
	}
	return 0
}

func MessageOf(err error, fallback string) string {
	var backendErr *Error
	if errors.As(err, &backendErr) && strings.TrimSpace(backendErr.Message) != "" {
		return backendErr.Message
	}
	return fallback
}

type OrderFilters struct {
	Page             int
	Limit            int
	FromDepartmentID int64
	CategoryID       int64
	FulfillmentDate  string
	FulfillmentFrom  string
	FulfillmentTo    string
}

func IsAdmin(viewer *contract.Me) bool { return viewer != nil && viewer.Role == "admin" }

func CanUseProduction(viewer *contract.Me) bool {
	return viewer != nil && (viewer.Role == "admin" || viewer.Role == "baker")
}

func CanManageOrders(viewer *contract.Me) bool {
	return viewer != nil && (viewer.Role == "admin" || viewer.Role == "shop")
}
