package http

import (
	"time"

	"github.com/google/uuid"

	"bakery/internal/domain/client"
	"bakery/internal/domain/kitchen"
	"bakery/internal/domain/order"
	"bakery/internal/domain/product"
	"bakery/internal/domain/user"
)

// Агрегаты имеют приватные поля, поэтому JSON-представление описано явными DTO.

type UserDTO struct {
	ID        uuid.UUID `json:"id"`
	Login     string    `json:"login"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

func userDTO(u *user.User) UserDTO {
	return UserDTO{ID: u.ID(), Login: u.Login(), Role: string(u.Role()), CreatedAt: u.CreatedAt()}
}

type ClientDTO struct {
	ID             uuid.UUID  `json:"id"`
	UserID         *uuid.UUID `json:"user_id"`
	Name           string     `json:"name"`
	TelegramChatID *int64     `json:"telegram_chat_id"`
	Note           string     `json:"note"`
	CreatedAt      time.Time  `json:"created_at"`
}

func clientDTO(c *client.Client) ClientDTO {
	return ClientDTO{
		ID:             c.ID(),
		UserID:         c.UserID(),
		Name:           c.Name(),
		TelegramChatID: c.TelegramChatID(),
		Note:           c.Note(),
		CreatedAt:      c.CreatedAt(),
	}
}

func clientDTOs(cs []*client.Client) []ClientDTO {
	out := make([]ClientDTO, 0, len(cs))
	for _, c := range cs {
		out = append(out, clientDTO(c))
	}
	return out
}

type KitchenDTO struct {
	ID        uuid.UUID  `json:"id"`
	UserID    *uuid.UUID `json:"user_id"`
	Name      string     `json:"name"`
	CreatedAt time.Time  `json:"created_at"`
}

func kitchenDTO(k *kitchen.Kitchen) KitchenDTO {
	return KitchenDTO{ID: k.ID(), UserID: k.UserID(), Name: k.Name(), CreatedAt: k.CreatedAt()}
}

func kitchenDTOs(ks []*kitchen.Kitchen) []KitchenDTO {
	out := make([]KitchenDTO, 0, len(ks))
	for _, k := range ks {
		out = append(out, kitchenDTO(k))
	}
	return out
}

type ProductDTO struct {
	ID        uuid.UUID `json:"id"`
	IikoID    *string   `json:"iiko_id"`
	Name      string    `json:"name"`
	Unit      string    `json:"unit"`
	CreatedAt time.Time `json:"created_at"`
}

func productDTO(p *product.Product) ProductDTO {
	return ProductDTO{ID: p.ID(), IikoID: p.IikoID(), Name: p.Name(), Unit: string(p.Unit()), CreatedAt: p.CreatedAt()}
}

func productDTOs(ps []*product.Product) []ProductDTO {
	out := make([]ProductDTO, 0, len(ps))
	for _, p := range ps {
		out = append(out, productDTO(p))
	}
	return out
}

type OrderItemDTO struct {
	ID        uuid.UUID `json:"id"`
	OrderID   uuid.UUID `json:"order_id"`
	ProductID uuid.UUID `json:"product_id"`
	Quantity  float64   `json:"quantity"`
	Unit      string    `json:"unit"`
}

func orderItemDTO(i *order.Item) OrderItemDTO {
	return OrderItemDTO{
		ID:        i.ID(),
		OrderID:   i.OrderID(),
		ProductID: i.ProductID(),
		Quantity:  i.Quantity(),
		Unit:      string(i.Unit()),
	}
}

func orderItemDTOs(items []*order.Item) []OrderItemDTO {
	out := make([]OrderItemDTO, 0, len(items))
	for _, i := range items {
		out = append(out, orderItemDTO(i))
	}
	return out
}

type OrderDTO struct {
	ID        uuid.UUID      `json:"id"`
	ClientID  uuid.UUID      `json:"client_id"`
	KitchenID *uuid.UUID     `json:"kitchen_id"`
	OrderDate string         `json:"order_date"`
	Status    string         `json:"status"`
	RawText   string         `json:"raw_text"`
	Note      string         `json:"note"`
	Items     []OrderItemDTO `json:"items"`
	CreatedAt time.Time      `json:"created_at"`
}

func orderDTO(o *order.Order) OrderDTO {
	return OrderDTO{
		ID:        o.ID(),
		ClientID:  o.ClientID(),
		KitchenID: o.KitchenID(),
		OrderDate: o.OrderDate().Format("2006-01-02"),
		Status:    string(o.Status()),
		RawText:   o.RawText(),
		Note:      o.Note(),
		Items:     orderItemDTOs(o.Items()),
		CreatedAt: o.CreatedAt(),
	}
}

func orderDTOs(os []*order.Order) []OrderDTO {
	out := make([]OrderDTO, 0, len(os))
	for _, o := range os {
		out = append(out, orderDTO(o))
	}
	return out
}
