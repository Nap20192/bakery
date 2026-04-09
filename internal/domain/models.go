package domain

import "github.com/google/uuid"

type Client struct {
	id   uuid.UUID
	name string
}
