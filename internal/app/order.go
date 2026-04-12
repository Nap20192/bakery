package app

import "bakery/internal/adapter/outbound/db"

type OrderService struct {
	queries *db.Queries
}
