package web

import (
	"fmt"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"bakery/frontend/internal/application"
	"bakery/internal/inbound/api/contract"
)

type productionEditorShare struct {
	OrderNumber      string
	OrderedQuantity  float64
	ProducedQuantity float64
}

type productionEditorRow struct {
	ProductName      string
	OrderedQuantity  float64
	LoadedQuantity   float64
	ProducedQuantity float64
	Reason           string
	ReasonConflict   bool
	Linked           bool
	Shares           []productionEditorShare
}

type productionEditorData struct {
	Sheet    *contract.ProductionSheet
	Orders   []contract.Order
	Rows     []productionEditorRow
	Overview productionOverview
}

// productionOverview is the read-only «Обзор»: a pivot with dishes down the side,
// orders across the top. The order columns show the plain заявка (ordered) — the
// produced fact is NOT split back across orders. Only «Итого» (per dish and grand)
// carries the produced fact with its +/- deviation.
type productionPivotCell struct {
	Produced float64 // факт (выпечено)
	Delta    float64 // produced − ordered; 0 when the fact matched the order
}

type productionPivotRow struct {
	Name  string
	Cells []float64           // заявка по заказу, aligned with OrderNumbers
	Total productionPivotCell // итого по позиции: факт + отклонение
}

type productionOverview struct {
	OrderNumbers []string             // column headers, in batch order
	Rows         []productionPivotRow // one per dish
	ColumnTotals []float64            // сумма заявки по заказу, aligned with OrderNumbers
	GrandTotal   productionPivotCell  // итог: факт + отклонение
}

// buildProductionOverview pivots the batch: each editor row is a dish, its Shares
// place the заявка under the matching order column. The produced fact only lands in
// «Итого» (per dish and grand), so no fact ever gets re-apportioned across orders.
func buildProductionOverview(orders []contract.Order, rows []productionEditorRow) productionOverview {
	overview := productionOverview{
		OrderNumbers: make([]string, len(orders)),
		Rows:         make([]productionPivotRow, 0, len(rows)),
		ColumnTotals: make([]float64, len(orders)),
	}
	columnOf := make(map[string]int, len(orders))
	for index, order := range orders {
		overview.OrderNumbers[index] = order.Number
		columnOf[order.Number] = index
	}
	var grandOrdered, grandProduced float64
	for _, row := range rows {
		pivot := productionPivotRow{Name: row.ProductName, Cells: make([]float64, len(orders))}
		for _, share := range row.Shares {
			if column, ok := columnOf[share.OrderNumber]; ok {
				pivot.Cells[column] += share.OrderedQuantity
				overview.ColumnTotals[column] += share.OrderedQuantity
			}
		}
		pivot.Total = productionPivotCell{Produced: row.ProducedQuantity, Delta: row.ProducedQuantity - row.OrderedQuantity}
		grandOrdered += row.OrderedQuantity
		grandProduced += row.ProducedQuantity
		overview.Rows = append(overview.Rows, pivot)
	}
	overview.GrandTotal = productionPivotCell{Produced: grandProduced, Delta: grandProduced - grandOrdered}
	return overview
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
	data := productionJournalData{Categories: categories}
	data.Rows = s.buildProductionJournal(r, cred, sheets, data.Categories)
	for _, row := range data.Rows {
		for _, cell := range row.Cells {
			data.Count += len(cell)
		}
	}
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
	rows := buildProductionRows(orders, sheet.Items)
	data := productionEditorData{Sheet: &sheet, Orders: orders, Rows: rows, Overview: buildProductionOverview(orders, rows)}
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
		if len(sheet.OrderNumbers) == 0 {
			continue
		}
		order, err := s.queries.Order(r.Context(), cred, sheet.OrderNumbers[0])
		if err != nil || order.Category == nil {
			continue
		}
		index, exists := categoryIndex[order.Category.ID]
		if !exists {
			continue
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
		row.cells[index] = append(row.cells[index], productionSheetView{Sheet: sheet, Category: order.Category})
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
		byKey[productionItemKey(item.OrderNumber, item.ProductName)] = item
	}
	type rowBuilder struct {
		row     productionEditorRow
		reasons map[string]struct{}
	}
	builders := make([]rowBuilder, 0)
	byProduct := make(map[string]int)
	for _, order := range orders {
		for _, item := range order.Items {
			productKey := strings.ToLower(strings.TrimSpace(item.ProductName))
			index, ok := byProduct[productKey]
			if !ok {
				index = len(builders)
				byProduct[productKey] = index
				builders = append(builders, rowBuilder{
					row:     productionEditorRow{ProductName: item.ProductName, Linked: true},
					reasons: make(map[string]struct{}),
				})
			}

			ordered := item.ProductionQuantity
			loaded := ordered
			produced := effectiveQuantity(item)
			reason := item.ProducedReason
			if value, exists := byKey[productionItemKey(order.Number, item.ProductName)]; exists {
				loaded = value.LoadedQuantity
				produced = value.ProducedQuantity
				reason = value.Reason
			}

			builder := &builders[index]
			builder.row.OrderedQuantity += ordered
			builder.row.LoadedQuantity += loaded
			builder.row.ProducedQuantity += produced
			builder.row.Linked = builder.row.Linked && loaded == ordered && produced == ordered
			builder.row.Shares = append(builder.row.Shares, productionEditorShare{
				OrderNumber: order.Number, OrderedQuantity: ordered, ProducedQuantity: produced,
			})
			if produced != ordered {
				builder.reasons[strings.TrimSpace(reason)] = struct{}{}
			}
		}
	}

	rows := make([]productionEditorRow, 0, len(builders))
	for _, builder := range builders {
		if len(builder.reasons) == 1 {
			for reason := range builder.reasons {
				builder.row.Reason = reason
			}
		} else if len(builder.reasons) > 1 {
			builder.row.ReasonConflict = true
		}
		rows = append(rows, builder.row)
	}
	return rows
}

func productionItemKey(orderNumber, productName string) string {
	return strings.TrimSpace(orderNumber) + "\x00" + strings.ToLower(strings.TrimSpace(productName))
}

func parseProductionWrite(r *http.Request) (contract.ProductionWrite, error) {
	if err := r.ParseForm(); err != nil {
		return contract.ProductionWrite{}, fmt.Errorf("не удалось прочитать форму")
	}
	names := r.Form["product_name"]
	loaded := r.Form["loaded_quantity"]
	produced := r.Form["produced_quantity"]
	reasons := r.Form["reason"]
	shareCounts := r.Form["share_count"]
	numbers := r.Form["order_number"]
	ordered := r.Form["ordered_quantity"]
	if len(names) == 0 || len(loaded) != len(names) || len(produced) != len(names) ||
		len(reasons) != len(names) || len(shareCounts) != len(names) || len(numbers) != len(ordered) {
		return contract.ProductionWrite{}, fmt.Errorf("в партии нет позиций")
	}
	orderIndexes := make(map[string]int)
	body := contract.ProductionWrite{}
	shareOffset := 0
	for index, nameValue := range names {
		name := strings.TrimSpace(nameValue)
		loadedTotal := valueAtFloat(loaded, index)
		producedTotal := valueAtFloat(produced, index)
		reason := strings.TrimSpace(valueAt(reasons, index))
		shareCount, err := strconv.Atoi(strings.TrimSpace(shareCounts[index]))
		if err != nil || name == "" || shareCount <= 0 || shareOffset+shareCount > len(numbers) {
			return body, fmt.Errorf("проверьте состав партии")
		}
		if loadedTotal < 0 || producedTotal < 0 ||
			math.IsNaN(loadedTotal) || math.IsInf(loadedTotal, 0) ||
			math.IsNaN(producedTotal) || math.IsInf(producedTotal, 0) {
			return body, fmt.Errorf("закладка и выход не могут быть отрицательными")
		}
		if len([]rune(reason)) > 200 {
			return body, fmt.Errorf("обоснование должно быть не длиннее 200 символов")
		}

		rowNumbers := numbers[shareOffset : shareOffset+shareCount]
		rowOrdered := make([]float64, shareCount)
		for shareIndex := range rowOrdered {
			rowNumbers[shareIndex] = strings.TrimSpace(rowNumbers[shareIndex])
			rowOrdered[shareIndex] = valueAtFloat(ordered, shareOffset+shareIndex)
			if rowNumbers[shareIndex] == "" || rowOrdered[shareIndex] <= 0 ||
				math.IsNaN(rowOrdered[shareIndex]) || math.IsInf(rowOrdered[shareIndex], 0) {
				return body, fmt.Errorf("проверьте количество по заявкам для позиции %q", name)
			}
		}
		loadedShares := distributeProductionQuantity(loadedTotal, rowOrdered)
		producedShares := distributeProductionQuantity(producedTotal, rowOrdered)
		for shareIndex, number := range rowNumbers {
			orderIndex, ok := orderIndexes[number]
			if !ok {
				orderIndex = len(body.Orders)
				orderIndexes[number] = orderIndex
				body.Orders = append(body.Orders, contract.ProductionOrderWrite{Number: number})
			}
			loadedQuantity := loadedShares[shareIndex]
			body.Orders[orderIndex].Items = append(body.Orders[orderIndex].Items, contract.ProductionItemWrite{
				ProductName: name, LoadedQuantity: &loadedQuantity, ProducedQuantity: producedShares[shareIndex], Reason: reason,
			})
		}
		shareOffset += shareCount
	}
	if shareOffset != len(numbers) {
		return body, fmt.Errorf("проверьте состав партии")
	}
	if len(body.Orders) == 0 {
		return body, fmt.Errorf("выберите хотя бы один заказ")
	}
	return body, nil
}

func distributeProductionQuantity(total float64, ordered []float64) []float64 {
	result := make([]float64, len(ordered))
	var orderedTotal float64
	for _, quantity := range ordered {
		orderedTotal += quantity
	}
	var allocated float64
	for index := range ordered {
		if index == len(ordered)-1 {
			result[index] = roundProductionQuantity(total - allocated)
			break
		}
		result[index] = roundProductionQuantity(total * ordered[index] / orderedTotal)
		allocated += result[index]
	}
	return result
}

func roundProductionQuantity(value float64) float64 {
	value = math.Round(value*10) / 10
	if math.Abs(value) < 0.000001 {
		return 0
	}
	return value
}

func parseInt64(value string) int64 {
	result, _ := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	return result
}
