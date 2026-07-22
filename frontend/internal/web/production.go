package web

import (
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"bakery/frontend/internal/application"
	"bakery/internal/inbound/api/contract"
)

type productionEditorRow struct {
	OrderNumber      string
	ProductName      string
	OrderedQuantity  float64
	LoadedQuantity   float64
	ProducedQuantity float64
	Reason           string
	Linked           bool
}

type productionEditorData struct {
	Sheet  *contract.ProductionSheet
	Orders []contract.Order
	Rows   []productionEditorRow
}

type productionSheetView struct {
	Sheet    contract.ProductionSheet
	Category *contract.Category
}

type productionJournalRow struct {
	Date  string
	Cells [][]productionSheetView
}

type productionJournalData struct {
	Categories []contract.Category
	Rows       []productionJournalRow
	Count      int
}

func (s *server) productionPage(w http.ResponseWriter, r *http.Request) {
	viewer, cred, ok := s.requireProduction(w, r)
	if !ok {
		return
	}
	sheets, err := s.queries.ProductionSheets(r.Context(), cred)
	if err != nil {
		s.renderError(w, r, statusOr(err, http.StatusBadGateway), application.MessageOf(err, "Не удалось загрузить журнал отработок."))
		return
	}
	categories, err := s.queries.Categories(r.Context(), cred)
	if err != nil {
		s.renderError(w, r, statusOr(err, http.StatusBadGateway), application.MessageOf(err, "Не удалось загрузить типы заявок."))
		return
	}
	data := productionJournalData{Categories: append(categories, contract.Category{Name: "Без типа", Color: "stone"}), Count: len(sheets)}
	data.Rows = s.buildProductionJournal(r, cred, sheets, data.Categories)
	s.render(w, r, http.StatusOK, page{Title: "Отработки", View: "production", Viewer: viewer, Success: queryMessage(r, "success"), Data: data})
}

func (s *server) productionDetailPage(w http.ResponseWriter, r *http.Request) {
	viewer, cred, ok := s.requireProduction(w, r)
	if !ok {
		return
	}
	id, err := parseID(r)
	if err != nil {
		s.renderError(w, r, http.StatusBadRequest, "Некорректный номер отработки.")
		return
	}
	sheet, err := s.queries.ProductionSheet(r.Context(), cred, id)
	if err != nil {
		s.renderError(w, r, statusOr(err, http.StatusBadGateway), application.MessageOf(err, "Не удалось загрузить отработку."))
		return
	}
	orders, err := s.loadOrders(r, cred, sheet.OrderNumbers)
	if err != nil {
		s.renderError(w, r, statusOr(err, http.StatusBadGateway), application.MessageOf(err, "Не удалось загрузить заказы партии."))
		return
	}
	data := productionEditorData{Sheet: &sheet, Orders: orders, Rows: buildProductionRows(orders, sheet.Items)}
	s.render(w, r, http.StatusOK, page{Title: fmt.Sprintf("Отработка №%d", sheet.ID), View: "production-detail", Viewer: viewer, Success: queryMessage(r, "success"), Data: data})
}

func (s *server) productionCreate(w http.ResponseWriter, r *http.Request) {
	_, cred, ok := s.requireProduction(w, r)
	if !ok {
		return
	}
	body, err := parseProductionWrite(r)
	if err != nil {
		s.renderError(w, r, http.StatusUnprocessableEntity, err.Error())
		return
	}
	sheet, err := s.commands.CreateProductionSheet(r.Context(), cred, body)
	if err != nil {
		s.renderError(w, r, statusOr(err, http.StatusBadGateway), application.MessageOf(err, "Не удалось сохранить отработку."))
		return
	}
	s.redirect(w, r, fmt.Sprintf("/production/%d?success=%s", sheet.ID, url.QueryEscape("Отработка сохранена.")))
}

func (s *server) productionUpdate(w http.ResponseWriter, r *http.Request) {
	_, cred, ok := s.requireProduction(w, r)
	if !ok {
		return
	}
	id, err := parseID(r)
	if err != nil {
		s.renderError(w, r, http.StatusBadRequest, "Некорректный номер отработки.")
		return
	}
	body, err := parseProductionWrite(r)
	if err != nil {
		s.renderError(w, r, http.StatusUnprocessableEntity, err.Error())
		return
	}
	sheet, err := s.commands.UpdateProductionSheet(r.Context(), cred, id, body)
	if err != nil {
		s.renderError(w, r, statusOr(err, http.StatusBadGateway), application.MessageOf(err, "Не удалось сохранить изменения."))
		return
	}
	s.redirect(w, r, fmt.Sprintf("/production/%d?success=%s", sheet.ID, url.QueryEscape("Изменения сохранены.")))
}

func (s *server) productionDelete(w http.ResponseWriter, r *http.Request) {
	_, cred, ok := s.requireProduction(w, r)
	if !ok {
		return
	}
	id, err := parseID(r)
	if err != nil {
		s.renderError(w, r, http.StatusBadRequest, "Некорректный номер отработки.")
		return
	}
	if err := s.commands.DeleteProductionSheet(r.Context(), cred, id); err != nil {
		s.renderError(w, r, statusOr(err, http.StatusBadGateway), application.MessageOf(err, "Не удалось удалить отработку."))
		return
	}
	s.redirect(w, r, "/production?success="+url.QueryEscape("Отработка удалена."))
}

func (s *server) buildProductionJournal(r *http.Request, cred application.Credentials, sheets []contract.ProductionSheet, categories []contract.Category) []productionJournalRow {
	type rowBuilder struct {
		date  string
		cells [][]productionSheetView
	}
	rows := make(map[string]*rowBuilder)
	categoryIndex := make(map[int64]int)
	for index, category := range categories {
		categoryIndex[category.ID] = index
	}
	for _, sheet := range sheets {
		categoryID := int64(0)
		var category *contract.Category
		if len(sheet.OrderNumbers) > 0 {
			order, err := s.queries.Order(r.Context(), cred, sheet.OrderNumbers[0])
			if err == nil && order.Category != nil {
				categoryID = order.Category.ID
				category = order.Category
			}
		}
		date := sheet.CreatedAt
		if len(date) >= 10 {
			date = date[:10]
		}
		row := rows[date]
		if row == nil {
			row = &rowBuilder{date: date, cells: make([][]productionSheetView, len(categories))}
			rows[date] = row
		}
		index, exists := categoryIndex[categoryID]
		if !exists {
			index = len(categories) - 1
		}
		row.cells[index] = append(row.cells[index], productionSheetView{Sheet: sheet, Category: category})
	}
	dates := make([]string, 0, len(rows))
	for date := range rows {
		dates = append(dates, date)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(dates)))
	result := make([]productionJournalRow, 0, len(dates))
	for _, date := range dates {
		result = append(result, productionJournalRow{Date: date, Cells: rows[date].cells})
	}
	return result
}

func (s *server) loadOrders(r *http.Request, cred application.Credentials, numbers []string) ([]contract.Order, error) {
	orders := make([]contract.Order, 0, len(numbers))
	for _, number := range numbers {
		order, err := s.queries.Order(r.Context(), cred, number)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	return orders, nil
}

func buildProductionRows(orders []contract.Order, saved []contract.ProductionSheetItem) []productionEditorRow {
	byKey := make(map[string]contract.ProductionSheetItem, len(saved))
	for _, item := range saved {
		byKey[item.OrderNumber+"\x00"+item.ProductName] = item
	}
	rows := make([]productionEditorRow, 0)
	for _, order := range orders {
		for _, item := range order.Items {
			ordered := item.ProductionQuantity
			row := productionEditorRow{
				OrderNumber: order.Number, ProductName: item.ProductName, OrderedQuantity: ordered,
				LoadedQuantity: ordered, ProducedQuantity: effectiveQuantity(item), Reason: item.ProducedReason, Linked: item.ProducedQuantity == nil,
			}
			if value, ok := byKey[order.Number+"\x00"+item.ProductName]; ok {
				row.LoadedQuantity = value.LoadedQuantity
				row.ProducedQuantity = value.ProducedQuantity
				row.Reason = value.Reason
				row.Linked = value.LoadedQuantity == ordered && value.ProducedQuantity == ordered
			}
			rows = append(rows, row)
		}
	}
	return rows
}

func parseProductionWrite(r *http.Request) (contract.ProductionWrite, error) {
	if err := r.ParseForm(); err != nil {
		return contract.ProductionWrite{}, fmt.Errorf("не удалось прочитать форму")
	}
	numbers := r.Form["order_number"]
	names := r.Form["product_name"]
	loaded := r.Form["loaded_quantity"]
	produced := r.Form["produced_quantity"]
	reasons := r.Form["reason"]
	if len(numbers) == 0 || len(names) != len(numbers) {
		return contract.ProductionWrite{}, fmt.Errorf("в партии нет позиций")
	}
	orderIndexes := make(map[string]int)
	body := contract.ProductionWrite{}
	for index, number := range numbers {
		number = strings.TrimSpace(number)
		name := strings.TrimSpace(valueAt(names, index))
		loadedQuantity := valueAtFloat(loaded, index)
		producedQuantity := valueAtFloat(produced, index)
		reason := strings.TrimSpace(valueAt(reasons, index))
		if number == "" || name == "" {
			continue
		}
		if loadedQuantity < 0 || producedQuantity < 0 {
			return body, fmt.Errorf("закладка и выход не могут быть отрицательными")
		}
		if len([]rune(reason)) > 200 {
			return body, fmt.Errorf("обоснование должно быть не длиннее 200 символов")
		}
		orderIndex, ok := orderIndexes[number]
		if !ok {
			orderIndex = len(body.Orders)
			orderIndexes[number] = orderIndex
			body.Orders = append(body.Orders, contract.ProductionOrderWrite{Number: number})
		}
		body.Orders[orderIndex].Items = append(body.Orders[orderIndex].Items, contract.ProductionItemWrite{
			ProductName: name, LoadedQuantity: &loadedQuantity, ProducedQuantity: producedQuantity, Reason: reason,
		})
	}
	if len(body.Orders) == 0 {
		return body, fmt.Errorf("выберите хотя бы один заказ")
	}
	return body, nil
}

func parseInt64(value string) int64 {
	result, _ := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	return result
}
