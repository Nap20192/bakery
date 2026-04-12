package app

import "bakery/internal/adapter/outbound/db"

type ClientService struct {
	queries *db.Queries
}
