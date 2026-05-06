package app

import (
	"bakery/internal/domain"
)

type AdminService struct{
	repo domain.OrderRepository
}

