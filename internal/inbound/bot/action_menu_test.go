package bot

import (
	"reflect"
	"testing"
	"time"

	"bakery/internal/pkg/enum"
)

func TestActionMenuSnapshotRows(t *testing.T) {
	filterDate := time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		snap actionMenuSnapshot
		want [][]string
	}{
		{
			name: "guest has no menu",
			snap: actionMenuSnapshot{state: actionStateGuest},
			want: nil,
		},
		{
			name: "shop idle",
			snap: actionMenuSnapshot{state: actionStateShopIdle, departmentType: string(enum.DepartmentTypeShop)},
			want: [][]string{{actionTemplates, actionOrders}},
		},
		{
			name: "shop creates order",
			snap: actionMenuSnapshot{state: actionStateShopCreate, departmentType: string(enum.DepartmentTypeShop), orderItems: 3},
			want: [][]string{
				{actionTemplates, actionOrders},
				{actionSubmitOrder, actionCancelOrder},
			},
		},
		{
			name: "shop updates order",
			snap: actionMenuSnapshot{state: actionStateShopUpdate, departmentType: string(enum.DepartmentTypeShop), orderItems: 2, editOrder: "Г.24.05.26.001"},
			want: [][]string{
				{actionTemplates, actionOrders},
				{actionUpdateOrder, actionCancelOrder},
			},
		},
		{
			name: "workshop idle shows filters",
			snap: actionMenuSnapshot{state: actionStateWorkshopIdle, departmentType: string(enum.DepartmentTypeWorkshop), filterShop: orderFilterAllShopsText},
			want: [][]string{
				{actionOrders},
				{orderFilterTodayText, orderFilterTomorrowText, orderFilterAllDatesText},
			},
		},
		{
			name: "workshop filter state",
			snap: actionMenuSnapshot{state: actionStateWorkshopFilter, departmentType: string(enum.DepartmentTypeWorkshop), filterShop: "Магазин Гагарина", filterDate: filterDate},
			want: [][]string{
				{actionOrders},
				{orderFilterTodayText, orderFilterTomorrowText, orderFilterAllDatesText},
			},
		},
		{
			name: "admin creates shop order",
			snap: actionMenuSnapshot{state: actionStateAdminCreate, admin: true, departmentType: string(enum.DepartmentTypeShop), orderItems: 1},
			want: [][]string{
				{actionTemplates, actionOrders},
				{actionSync},
				{actionSubmitOrder, actionCancelOrder},
			},
		},
		{
			name: "admin workshop filter",
			snap: actionMenuSnapshot{state: actionStateAdminWorkshopFilt, admin: true, departmentType: string(enum.DepartmentTypeWorkshop), filterShop: "Магазин Сарыарка"},
			want: [][]string{
				{actionOrders},
				{actionSync},
				{orderFilterTodayText, orderFilterTomorrowText, orderFilterAllDatesText},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.snap.rows(); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("rows = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestActionMenuSnapshotResolveState(t *testing.T) {
	tests := []struct {
		name string
		snap actionMenuSnapshot
		want actionMenuState
	}{
		{name: "shop idle", snap: actionMenuSnapshot{departmentType: string(enum.DepartmentTypeShop), filterShop: orderFilterAllShopsText}, want: actionStateShopIdle},
		{name: "shop create", snap: actionMenuSnapshot{departmentType: string(enum.DepartmentTypeShop), orderItems: 1, filterShop: orderFilterAllShopsText}, want: actionStateShopCreate},
		{name: "shop update", snap: actionMenuSnapshot{departmentType: string(enum.DepartmentTypeShop), orderItems: 1, editOrder: "Г.1", filterShop: orderFilterAllShopsText}, want: actionStateShopUpdate},
		{name: "workshop idle", snap: actionMenuSnapshot{departmentType: string(enum.DepartmentTypeWorkshop), filterShop: orderFilterAllShopsText}, want: actionStateWorkshopIdle},
		{name: "workshop filtered by shop", snap: actionMenuSnapshot{departmentType: string(enum.DepartmentTypeWorkshop), filterShop: "Магазин Гагарина"}, want: actionStateWorkshopFilter},
		{name: "workshop filtered by date", snap: actionMenuSnapshot{departmentType: string(enum.DepartmentTypeWorkshop), filterShop: orderFilterAllShopsText, filterDate: time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC)}, want: actionStateWorkshopFilter},
		{name: "admin idle", snap: actionMenuSnapshot{admin: true, departmentType: string(enum.DepartmentTypeShop), filterShop: orderFilterAllShopsText}, want: actionStateAdminIdle},
		{name: "admin create", snap: actionMenuSnapshot{admin: true, departmentType: string(enum.DepartmentTypeShop), orderItems: 1, filterShop: orderFilterAllShopsText}, want: actionStateAdminCreate},
		{name: "admin update", snap: actionMenuSnapshot{admin: true, departmentType: string(enum.DepartmentTypeShop), orderItems: 1, editOrder: "Г.1", filterShop: orderFilterAllShopsText}, want: actionStateAdminUpdate},
		{name: "admin workshop", snap: actionMenuSnapshot{admin: true, departmentType: string(enum.DepartmentTypeWorkshop), filterShop: orderFilterAllShopsText}, want: actionStateAdminWorkshop},
		{name: "admin workshop filter", snap: actionMenuSnapshot{admin: true, departmentType: string(enum.DepartmentTypeWorkshop), filterShop: "Магазин Шолохова"}, want: actionStateAdminWorkshopFilt},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.snap.resolveState(); got != tt.want {
				t.Fatalf("resolveState = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestDepartmentTypeForRole(t *testing.T) {
	tests := []struct {
		name string
		role string
		want string
	}{
		{name: "shop role uses shop mode", role: string(enum.RoleShop), want: string(enum.DepartmentTypeShop)},
		{name: "baker role uses workshop mode", role: string(enum.RoleBaker), want: string(enum.DepartmentTypeWorkshop)},
		{name: "admin role uses workshop mode without assigned department", role: string(enum.RoleAdmin), want: string(enum.DepartmentTypeWorkshop)},
		{name: "unknown role has no mode", role: string(enum.RoleUser), want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := departmentTypeForRole(tt.role); got != tt.want {
				t.Fatalf("departmentTypeForRole(%q) = %q, want %q", tt.role, got, tt.want)
			}
		})
	}
}
