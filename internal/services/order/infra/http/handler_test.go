package orderhttp

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"bakery/internal/inbound/api/httpx"
	"bakery/internal/pkg/enum"
	authdomain "bakery/internal/services/auth/domain"
	orderdomain "bakery/internal/services/order/domain"
)

func TestDecodeOrderWriteRequest(t *testing.T) {
	tests := []struct {
		name string
		body string
		ok   bool
	}{
		{
			name: "valid names without codes",
			body: `{"fulfillment_date":"2026-05-28","items":[{"product_name":"Сосиска в тесте","quantity":2,"reserved_quantity":1}]}`,
			ok:   true,
		},
		{
			name: "duplicate product",
			body: `{"items":[{"product_name":"Кокрок","quantity":1},{"product_name":" кокрок ","quantity":2}]}`,
		},
		{
			name: "fractional quantity",
			body: `{"items":[{"product_name":"Кокрок","quantity":1.5}]}`,
		},
		{
			name: "trailing json rejected",
			body: `{"items":[{"product_name":"Кокрок","quantity":1}]} {}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest("POST", "/orders", strings.NewReader(tt.body))
			response := httptest.NewRecorder()
			got, ok := decodeOrderWriteRequest(response, request)
			if ok != tt.ok {
				t.Fatalf("decodeOrderWriteRequest() ok = %v, want %v; body = %s", ok, tt.ok, response.Body.String())
			}
			if ok && (len(got.items) != 1 || got.items[0].Code != "") {
				t.Fatalf("decoded items = %#v; client codes must not be accepted", got.items)
			}
		})
	}
}

func TestMiniAppOrderAuthor(t *testing.T) {
	dbUsername := "saved_user"
	tests := []struct {
		name string
		user httpx.MiniAppUser
		want string
	}{
		{name: "signed telegram username", user: httpx.MiniAppUser{TelegramID: 4, TelegramUser: "current_user"}, want: "current_user"},
		{name: "saved telegram username", user: httpx.MiniAppUser{TelegramID: 4, Auth: authdomain.AuthUser{TelegramUsername: &dbUsername}}, want: "saved_user"},
		{name: "telegram id fallback", user: httpx.MiniAppUser{TelegramID: 4}, want: "telegram_4"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := miniAppOrderAuthor(tt.user); got != tt.want {
				t.Fatalf("miniAppOrderAuthor() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestUserCanReadOrder(t *testing.T) {
	shopID := int64(3)
	otherShopID := int64(4)
	tests := []struct {
		name string
		user httpx.MiniAppUser
		from *int64
		want bool
	}{
		{name: "shop reads own order", user: httpx.MiniAppUser{DepartmentID: shopID, DepartmentType: string(enum.DepartmentTypeShop)}, from: &shopID, want: true},
		{name: "shop cannot read another order", user: httpx.MiniAppUser{DepartmentID: shopID, DepartmentType: string(enum.DepartmentTypeShop)}, from: &otherShopID, want: false},
		{name: "workshop reads any order", user: httpx.MiniAppUser{DepartmentType: string(enum.DepartmentTypeWorkshop)}, from: &otherShopID, want: true},
		{name: "unknown department denied", user: httpx.MiniAppUser{DepartmentType: "unknown"}, from: &shopID, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := httpx.WithMiniAppUser(context.Background(), tt.user)
			got := UserCanReadOrder(ctx, orderdomain.Order{FromDepartmentID: tt.from})
			if got != tt.want {
				t.Fatalf("UserCanReadOrder() = %v, want %v", got, tt.want)
			}
		})
	}
}
