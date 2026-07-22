package web

import (
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"bakery/frontend/internal/application"
	"bakery/internal/inbound/api/contract"
)

type tableColumn struct {
	Date      string
	Label     string
	Tone      string
	ActionURL string
}

type tableCell struct {
	Quantity float64
	Delta    float64
	Produced bool
}

type tableRow struct {
	Dish  contract.Dish
	Cells []tableCell
}

type ordersTableData struct {
	Categories       []contract.Category
	ActiveCategoryID int64
	Columns          []tableColumn
	Rows             []tableRow
	Totals           []float64
	PrevStart        string
	NextStart        string
	TodayStart       string
}

func (s *server) ordersTablePage(w http.ResponseWriter, r *http.Request) {
	viewer, cred, ok := s.requireProduction(w, r)
	if !ok {
		return
	}
	start, err := time.Parse(time.DateOnly, r.URL.Query().Get("start"))
	if err != nil {
		start = startOfDay(time.Now()).AddDate(0, 0, -1)
	}
	end := start.AddDate(0, 0, 7)
	ordersPage, err := s.queries.Orders(r.Context(), cred, application.OrderFilters{
		Page: 1, Limit: 100, FulfillmentFrom: start.Format(time.DateOnly), FulfillmentTo: end.Format(time.DateOnly),
	})
	if err != nil {
		s.renderError(w, r, statusOr(err, http.StatusBadGateway), application.MessageOf(err, "Не удалось загрузить сводную таблицу."))
		return
	}
	catalog, err := s.queries.Catalog(r.Context(), cred)
	if err != nil {
		s.renderError(w, r, statusOr(err, http.StatusBadGateway), application.MessageOf(err, "Не удалось загрузить каталог."))
		return
	}
	categories, err := s.queries.Categories(r.Context(), cred)
	if err != nil {
		s.renderError(w, r, statusOr(err, http.StatusBadGateway), application.MessageOf(err, "Не удалось загрузить типы заявок."))
		return
	}
	for _, order := range ordersPage.Items {
		if order.Category == nil {
			categories = append(categories, contract.Category{Name: "Без типа", Color: "stone"})
			break
		}
	}
	activeID := parseCategoryID(r.URL.Query().Get("category_id"))
	if activeID == 0 && len(categories) > 0 {
		activeID = categories[0].ID
	}
	data := buildOrdersTable(ordersPage.Items, catalog, categories, activeID, start, end)
	data.PrevStart = start.AddDate(0, 0, -8).Format(time.DateOnly)
	data.NextStart = start.AddDate(0, 0, 8).Format(time.DateOnly)
	data.TodayStart = startOfDay(time.Now()).AddDate(0, 0, -1).Format(time.DateOnly)
	s.render(w, r, http.StatusOK, page{Title: "Таблица", View: "orders-table", Viewer: viewer, Data: data})
}

func buildOrdersTable(orders []contract.Order, catalog []contract.Dish, categories []contract.Category, activeID int64, start, end time.Time) ordersTableData {
	data := ordersTableData{Categories: categories, ActiveCategoryID: activeID}
	for day := start; !day.After(end); day = day.AddDate(0, 0, 1) {
		data.Columns = append(data.Columns, tableColumn{Date: day.Format(time.DateOnly), Label: weekdayLabel(day), Tone: dayTone(day)})
	}
	filteredOrders := make([]contract.Order, 0)
	for _, order := range orders {
		if order.Cancelled || categoryIDOf(order) != activeID {
			continue
		}
		filteredOrders = append(filteredOrders, order)
	}
	for columnIndex := range data.Columns {
		var pending []string
		var sheetID int64
		sameSheet := true
		for _, order := range filteredOrders {
			if order.FulfillmentDate != data.Columns[columnIndex].Date {
				continue
			}
			if order.ProductionSheetID == nil {
				pending = append(pending, order.Number)
				continue
			}
			if sheetID == 0 {
				sheetID = *order.ProductionSheetID
			} else if sheetID != *order.ProductionSheetID {
				sameSheet = false
			}
		}
		if len(pending) > 0 {
			params := url.Values{}
			for _, number := range pending {
				params.Add("order", number)
			}
			data.Columns[columnIndex].ActionURL = "/orders/selection?" + params.Encode()
		} else if sheetID > 0 && sameSheet {
			data.Columns[columnIndex].ActionURL = "/production/" + formatQuantity(sheetID)
		}
	}
	byName := make(map[string]contract.Dish, len(catalog))
	for _, dish := range catalog {
		if dish.CategoryID == nil || *dish.CategoryID == activeID {
			byName[dish.Name] = dish
		}
	}
	for _, order := range filteredOrders {
		for _, item := range order.Items {
			if _, ok := byName[item.ProductName]; !ok {
				byName[item.ProductName] = contract.Dish{Name: item.ProductName, SortOrder: 1 << 30}
			}
		}
	}
	dishes := make([]contract.Dish, 0, len(byName))
	for _, dish := range byName {
		dishes = append(dishes, dish)
	}
	sort.SliceStable(dishes, func(i, j int) bool {
		if dishes[i].SortOrder == dishes[j].SortOrder {
			return strings.ToLower(dishes[i].Name) < strings.ToLower(dishes[j].Name)
		}
		return dishes[i].SortOrder < dishes[j].SortOrder
	})
	data.Totals = make([]float64, len(data.Columns))
	for _, dish := range dishes {
		row := tableRow{Dish: dish, Cells: make([]tableCell, len(data.Columns))}
		for columnIndex, column := range data.Columns {
			contributors := 0
			produced := 0
			for _, order := range filteredOrders {
				if order.FulfillmentDate != column.Date {
					continue
				}
				for _, item := range order.Items {
					if item.ProductName != dish.Name {
						continue
					}
					contributors++
					ordered := item.ProductionQuantity
					effective := effectiveQuantity(item)
					row.Cells[columnIndex].Quantity += effective
					row.Cells[columnIndex].Delta += effective - ordered
					if order.ProductionSheetID != nil {
						produced++
					}
				}
			}
			row.Cells[columnIndex].Produced = contributors > 0 && contributors == produced
			data.Totals[columnIndex] += row.Cells[columnIndex].Quantity
		}
		data.Rows = append(data.Rows, row)
	}
	return data
}
