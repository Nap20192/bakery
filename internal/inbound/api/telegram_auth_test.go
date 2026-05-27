package api

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	orderdomain "bakery/internal/domain/order"
	"bakery/internal/pkg/enum"
)

func TestValidateMiniAppInitData(t *testing.T) {
	now := time.Date(2026, 5, 27, 10, 0, 0, 0, time.UTC)
	const botToken = "123456:bot-token"

	tests := []struct {
		name    string
		build   func() string
		wantID  int64
		wantErr bool
	}{
		{
			name: "valid data",
			build: func() string {
				return signedInitData(botToken, now.Unix(), `{"id":711,"username":"shop"}`)
			},
			wantID: 711,
		},
		{
			name: "tampered user",
			build: func() string {
				values, _ := url.ParseQuery(signedInitData(botToken, now.Unix(), `{"id":711}`))
				values.Set("user", `{"id":999}`)
				return values.Encode()
			},
			wantErr: true,
		},
		{
			name: "expired data",
			build: func() string {
				return signedInitData(botToken, now.Add(-miniAppAuthMaxAge-time.Second).Unix(), `{"id":711}`)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := validateMiniAppInitData(tt.build(), botToken, now, miniAppAuthMaxAge)
			if tt.wantErr {
				if err == nil {
					t.Fatal("validateMiniAppInitData() returned nil error")
				}
				return
			}
			if err != nil {
				t.Fatalf("validateMiniAppInitData() error = %v", err)
			}
			if user.ID != tt.wantID {
				t.Fatalf("user.ID = %d, want %d", user.ID, tt.wantID)
			}
		})
	}
}

func TestMiniAppAuthorizationData(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   string
		ok     bool
	}{
		{name: "valid scheme", header: "tma auth_date=1&hash=a", want: "auth_date=1&hash=a", ok: true},
		{name: "case insensitive scheme", header: "TMA data", want: "data", ok: true},
		{name: "wrong scheme", header: "Bearer data", ok: false},
		{name: "empty data", header: "tma ", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := miniAppAuthorizationData(tt.header)
			if got != tt.want || ok != tt.ok {
				t.Fatalf("miniAppAuthorizationData() = (%q, %v), want (%q, %v)", got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestMiniAppUserCanReadOrder(t *testing.T) {
	shopID := int64(3)
	otherShopID := int64(4)
	tests := []struct {
		name string
		user miniAppUser
		from *int64
		want bool
	}{
		{name: "shop reads own order", user: miniAppUser{DepartmentID: shopID, DepartmentType: string(enum.DepartmentTypeShop)}, from: &shopID, want: true},
		{name: "shop cannot read another order", user: miniAppUser{DepartmentID: shopID, DepartmentType: string(enum.DepartmentTypeShop)}, from: &otherShopID, want: false},
		{name: "workshop reads any order", user: miniAppUser{DepartmentType: string(enum.DepartmentTypeWorkshop)}, from: &otherShopID, want: true},
		{name: "unknown department denied", user: miniAppUser{DepartmentType: "unknown"}, from: &shopID, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), miniAppUserContextKey{}, tt.user)
			got := miniAppUserCanReadOrder(ctx, orderdomain.Order{FromDepartmentID: tt.from})
			if got != tt.want {
				t.Fatalf("miniAppUserCanReadOrder() = %v, want %v", got, tt.want)
			}
		})
	}
}

func signedInitData(botToken string, authDate int64, user string) string {
	values := url.Values{
		"auth_date": {strconv.FormatInt(authDate, 10)},
		"query_id":  {"query"},
		"user":      {user},
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		lines = append(lines, key+"="+values.Get(key))
	}
	secretMAC := hmac.New(sha256.New, []byte("WebAppData"))
	_, _ = secretMAC.Write([]byte(botToken))
	signatureMAC := hmac.New(sha256.New, secretMAC.Sum(nil))
	_, _ = signatureMAC.Write([]byte(strings.Join(lines, "\n")))
	values.Set("hash", hex.EncodeToString(signatureMAC.Sum(nil)))
	return values.Encode()
}
