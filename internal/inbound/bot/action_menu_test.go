package bot

import (
	"reflect"
	"testing"
	"time"

	"bakery/internal/app"
)

func TestActionMenuSnapshotRows(t *testing.T) {
	filterDate := time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		snap actionMenuSnapshot
		want [][]string
	}{
		{
			name: "guest chooses location",
			snap: actionMenuSnapshot{state: actionStateGuest},
			want: [][]string{{actionChooseShop, actionChooseWorkshop}},
		},
		{
			name: "shop idle",
			snap: actionMenuSnapshot{state: actionStateShopIdle, departmentType: string(app.DepartmentTypeShop)},
			want: [][]string{{actionTemplates, actionOrders}},
		},
		{
			name: "shop creates order",
			snap: actionMenuSnapshot{state: actionStateShopCreate, departmentType: string(app.DepartmentTypeShop), orderItems: 3},
			want: [][]string{
				{actionTemplates, actionOrders},
				{"Создается заказ: 3 поз."},
				{actionSubmitOrder},
				{actionCancelOrder},
			},
		},
		{
			name: "shop updates order",
			snap: actionMenuSnapshot{state: actionStateShopUpdate, departmentType: string(app.DepartmentTypeShop), orderItems: 2, editOrder: "Г.24.05.26.001"},
			want: [][]string{
				{actionTemplates, actionOrders},
				{"Редактируется: Г.24.05.26.001"},
				{actionUpdateOrder},
				{actionCancelOrder},
			},
		},
		{
			name: "workshop idle shows filters",
			snap: actionMenuSnapshot{state: actionStateWorkshopIdle, departmentType: string(app.DepartmentTypeWorkshop), filterShop: orderFilterAllShopsText},
			want: [][]string{
				{actionOrders},
				{"Фильтр: Все магазины / Все даты"},
				{orderFilterTodayText, orderFilterTomorrowText, orderFilterAllDatesText},
			},
		},
		{
			name: "workshop filter state",
			snap: actionMenuSnapshot{state: actionStateWorkshopFilter, departmentType: string(app.DepartmentTypeWorkshop), filterShop: "Магазин Гагарина", filterDate: filterDate},
			want: [][]string{
				{actionOrders},
				{"Фильтр: Магазин Гагарина / 25.05.2026"},
				{orderFilterTodayText, orderFilterTomorrowText, orderFilterAllDatesText},
			},
		},
		{
			name: "admin creates shop order",
			snap: actionMenuSnapshot{state: actionStateAdminCreate, admin: true, departmentType: string(app.DepartmentTypeShop), orderItems: 1},
			want: [][]string{
				{actionTemplates, actionOrders},
				{actionSync},
				{"Создается заказ: 1 поз."},
				{actionSubmitOrder},
				{actionCancelOrder},
			},
		},
		{
			name: "admin workshop filter",
			snap: actionMenuSnapshot{state: actionStateAdminWorkshopFilt, admin: true, departmentType: string(app.DepartmentTypeWorkshop), filterShop: "Магазин Сарыарка"},
			want: [][]string{
				{actionOrders},
				{actionSync},
				{"Фильтр: Магазин Сарыарка / Все даты"},
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
		{name: "shop idle", snap: actionMenuSnapshot{departmentType: string(app.DepartmentTypeShop), filterShop: orderFilterAllShopsText}, want: actionStateShopIdle},
		{name: "shop create", snap: actionMenuSnapshot{departmentType: string(app.DepartmentTypeShop), orderItems: 1, filterShop: orderFilterAllShopsText}, want: actionStateShopCreate},
		{name: "shop update", snap: actionMenuSnapshot{departmentType: string(app.DepartmentTypeShop), orderItems: 1, editOrder: "Г.1", filterShop: orderFilterAllShopsText}, want: actionStateShopUpdate},
		{name: "workshop idle", snap: actionMenuSnapshot{departmentType: string(app.DepartmentTypeWorkshop), filterShop: orderFilterAllShopsText}, want: actionStateWorkshopIdle},
		{name: "workshop filtered by shop", snap: actionMenuSnapshot{departmentType: string(app.DepartmentTypeWorkshop), filterShop: "Магазин Гагарина"}, want: actionStateWorkshopFilter},
		{name: "workshop filtered by date", snap: actionMenuSnapshot{departmentType: string(app.DepartmentTypeWorkshop), filterShop: orderFilterAllShopsText, filterDate: time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC)}, want: actionStateWorkshopFilter},
		{name: "admin idle", snap: actionMenuSnapshot{admin: true, departmentType: string(app.DepartmentTypeShop), filterShop: orderFilterAllShopsText}, want: actionStateAdminIdle},
		{name: "admin create", snap: actionMenuSnapshot{admin: true, departmentType: string(app.DepartmentTypeShop), orderItems: 1, filterShop: orderFilterAllShopsText}, want: actionStateAdminCreate},
		{name: "admin update", snap: actionMenuSnapshot{admin: true, departmentType: string(app.DepartmentTypeShop), orderItems: 1, editOrder: "Г.1", filterShop: orderFilterAllShopsText}, want: actionStateAdminUpdate},
		{name: "admin workshop", snap: actionMenuSnapshot{admin: true, departmentType: string(app.DepartmentTypeWorkshop), filterShop: orderFilterAllShopsText}, want: actionStateAdminWorkshop},
		{name: "admin workshop filter", snap: actionMenuSnapshot{admin: true, departmentType: string(app.DepartmentTypeWorkshop), filterShop: "Магазин Шолохова"}, want: actionStateAdminWorkshopFilt},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.snap.resolveState(); got != tt.want {
				t.Fatalf("resolveState = %s, want %s", got, tt.want)
			}
		})
	}
}
