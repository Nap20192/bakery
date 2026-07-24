package web

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"bakery/frontend/internal/application"
	"bakery/internal/inbound/api/contract"
)

type orderFiltersView struct {
	Page             int
	FromDepartmentID int64
	CategoryID       int64
	FulfillmentDate  string
	FulfillmentFrom  string
	FulfillmentTo    string
}

type matrixCell struct {
	Shop   contract.Department
	Orders []contract.Order
}

type matrixRow struct {
	Date  string
	Label string
	Tone  string
	Cells []matrixCell
}

type ordersData struct {
	Page       contract.OrdersPage
	Categories []contract.Category
	Shops      []contract.Department
	Filters    orderFiltersView
	Matrix     []matrixRow
	PrevFrom   string
	PrevTo     string
	NextFrom   string
	NextTo     string
}

type orderItemGroup struct {
	Name  string
	Items []contract.OrderItem
}

type orderDetailData struct {
	Order  contract.Order
	Groups []orderItemGroup
}

type editorItem struct {
	Dish     contract.Dish
	Quantity float64
	Reserved float64
	Comment  string
}

type editorGroup struct {
	Name  string
	Items []editorItem
}

type orderFormData struct {
	Mode             string
	Order            *contract.Order
	Categories       []contract.Category
	Shops            []contract.Department
	WorkshopSource   bool
	Groups           []editorGroup
	CategoryID       int64
	FromDepartmentID int64
	FulfillmentDate  string
	GeneralComment   string
}

type selectionData struct {
	Orders []contract.Order
	// Numbers feeds the monitor panel, which builds its request from plain
	// order numbers the same way the saved-sheet page does.
	Numbers []string
	Rows    []productionEditorRow
}

type monitorData struct {
	Reports      []contract.MonitorReport
	OrderReports []contract.BatchOrderMonitor
}

func (s *server) ordersPage(w http.ResponseWriter, r *http.Request) {
	viewer, cred, ok := s.requireViewer(w, r)
	if !ok {
		return
	}
	filters := parseOrderFilters(r)
	apiFilters := application.OrderFilters{
		Page:             filters.Page,
		Limit:            20,
		FromDepartmentID: filters.FromDepartmentID,
		CategoryID:       filters.CategoryID,
		FulfillmentDate:  filters.FulfillmentDate,
	}
	if application.CanUseProduction(viewer) {
		if filters.FulfillmentFrom == "" || filters.FulfillmentTo == "" {
			from := startOfDay(time.Now()).AddDate(0, 0, -1)
			filters.FulfillmentFrom = from.Format(time.DateOnly)
			filters.FulfillmentTo = from.AddDate(0, 0, 4).Format(time.DateOnly)
		}
		apiFilters.Limit = 100
		apiFilters.FulfillmentFrom = filters.FulfillmentFrom
		apiFilters.FulfillmentTo = filters.FulfillmentTo
	}

	ordersPage, err := s.queries.Orders(r.Context(), cred, apiFilters)
	if err != nil {
		s.renderError(w, r, statusOr(err, http.StatusBadGateway), application.MessageOf(err, "Не удалось загрузить заказы."))
		return
	}
	categories, err := s.queries.Categories(r.Context(), cred)
	if err != nil {
		s.renderError(w, r, statusOr(err, http.StatusBadGateway), application.MessageOf(err, "Не удалось загрузить типы заявок."))
		return
	}
	departmentType := "shop"
	if application.CanUseProduction(viewer) {
		departmentType = ""
	}
	shops, err := s.queries.Departments(r.Context(), cred, departmentType)
	if err != nil {
		s.renderError(w, r, statusOr(err, http.StatusBadGateway), application.MessageOf(err, "Не удалось загрузить отправителей."))
		return
	}
	if application.CanUseProduction(viewer) {
		shops = matrixShops(ordersPage.Items, shops)
	}
	data := ordersData{Page: ordersPage, Categories: categories, Shops: shops, Filters: filters}
	if application.CanUseProduction(viewer) {
		data.Matrix = buildOrderMatrix(ordersPage.Items, shops, filters.FulfillmentFrom, filters.FulfillmentTo)
		from, _ := time.Parse(time.DateOnly, filters.FulfillmentFrom)
		data.PrevFrom = from.AddDate(0, 0, -5).Format(time.DateOnly)
		data.PrevTo = from.AddDate(0, 0, -1).Format(time.DateOnly)
		data.NextFrom = from.AddDate(0, 0, 5).Format(time.DateOnly)
		data.NextTo = from.AddDate(0, 0, 9).Format(time.DateOnly)
	}
	s.render(w, r, http.StatusOK, page{Title: "Заказы", View: "orders", Viewer: viewer, Success: queryMessage(r, "success"), Data: data})
}

func (s *server) orderDetailPage(w http.ResponseWriter, r *http.Request) {
	viewer, cred, ok := s.requireViewer(w, r)
	if !ok {
		return
	}
	order, err := s.queries.Order(r.Context(), cred, r.PathValue("number"))
	if err != nil {
		s.renderError(w, r, statusOr(err, http.StatusBadGateway), application.MessageOf(err, "Не удалось загрузить заказ."))
		return
	}
	catalog, _ := s.queries.Catalog(r.Context(), cred)
	data := orderDetailData{Order: order, Groups: groupOrderItems(order.Items, catalog)}
	s.render(w, r, http.StatusOK, page{Title: order.Number, View: "order-detail", Viewer: viewer, Success: queryMessage(r, "success"), Data: data})
}

func (s *server) orderNewPage(w http.ResponseWriter, r *http.Request) {
	viewer, cred, ok := s.requireViewer(w, r)
	if !ok {
		return
	}
	if !application.CanManageOrders(viewer) {
		s.renderError(w, r, http.StatusForbidden, "Недостаточно прав для создания заказов.")
		return
	}
	var source *contract.Order
	if number := strings.TrimSpace(r.URL.Query().Get("duplicate")); number != "" {
		order, err := s.queries.Order(r.Context(), cred, number)
		if err != nil {
			s.renderError(w, r, statusOr(err, http.StatusBadGateway), application.MessageOf(err, "Не удалось открыть заказ для повтора."))
			return
		}
		source = &order
	}
	s.renderOrderForm(w, r, viewer, cred, http.StatusOK, "create", source, "")
}

func (s *server) orderEditPage(w http.ResponseWriter, r *http.Request) {
	viewer, cred, ok := s.requireViewer(w, r)
	if !ok {
		return
	}
	if !application.CanManageOrders(viewer) {
		s.renderError(w, r, http.StatusForbidden, "Недостаточно прав для редактирования заказа.")
		return
	}
	order, err := s.queries.Order(r.Context(), cred, r.PathValue("number"))
	if err != nil {
		s.renderError(w, r, statusOr(err, http.StatusBadGateway), application.MessageOf(err, "Не удалось загрузить заказ."))
		return
	}
	s.renderOrderForm(w, r, viewer, cred, http.StatusOK, "update", &order, "")
}

func (s *server) orderCreate(w http.ResponseWriter, r *http.Request) {
	viewer, cred, ok := s.requireViewer(w, r)
	if !ok {
		return
	}
	if !application.CanManageOrders(viewer) {
		s.renderError(w, r, http.StatusForbidden, "Недостаточно прав для создания заказов.")
		return
	}
	body, err := parseOrderWrite(r)
	if err != nil {
		s.renderOrderFormFromWrite(w, r, viewer, cred, http.StatusUnprocessableEntity, "create", nil, body, err.Error())
		return
	}
	created, err := s.commands.CreateOrder(r.Context(), cred, body)
	if err != nil {
		s.renderOrderFormFromWrite(w, r, viewer, cred, statusOr(err, http.StatusBadGateway), "create", nil, body, application.MessageOf(err, "Не удалось сохранить заказ."))
		return
	}
	s.redirect(w, r, "/orders/"+url.PathEscape(created.Number)+"?success="+url.QueryEscape("Заказ создан."))
}

func (s *server) orderUpdate(w http.ResponseWriter, r *http.Request) {
	viewer, cred, ok := s.requireViewer(w, r)
	if !ok {
		return
	}
	if !application.CanManageOrders(viewer) {
		s.renderError(w, r, http.StatusForbidden, "Недостаточно прав для редактирования заказа.")
		return
	}
	number := r.PathValue("number")
	existing, err := s.queries.Order(r.Context(), cred, number)
	if err != nil {
		s.renderError(w, r, statusOr(err, http.StatusBadGateway), application.MessageOf(err, "Не удалось загрузить заказ."))
		return
	}
	body, parseErr := parseOrderWrite(r)
	body.CategoryID = 0
	if parseErr != nil {
		s.renderOrderFormFromWrite(w, r, viewer, cred, http.StatusUnprocessableEntity, "update", &existing, body, parseErr.Error())
		return
	}
	updated, err := s.commands.UpdateOrder(r.Context(), cred, number, body)
	if err != nil {
		s.renderOrderFormFromWrite(w, r, viewer, cred, statusOr(err, http.StatusBadGateway), "update", &existing, body, application.MessageOf(err, "Не удалось сохранить изменения."))
		return
	}
	s.redirect(w, r, "/orders/"+url.PathEscape(updated.Number)+"?success="+url.QueryEscape("Изменения сохранены."))
}

func (s *server) orderCancel(w http.ResponseWriter, r *http.Request) {
	s.orderAction(w, r, "cancel")
}

func (s *server) orderRestore(w http.ResponseWriter, r *http.Request) {
	s.orderAction(w, r, "restore")
}

func (s *server) orderFavorite(w http.ResponseWriter, r *http.Request) {
	s.orderAction(w, r, "favorite")
}

func (s *server) orderAction(w http.ResponseWriter, r *http.Request, action string) {
	viewer, cred, ok := s.requireViewer(w, r)
	if !ok {
		return
	}
	number := r.PathValue("number")
	var err error
	switch action {
	case "cancel":
		if !application.CanManageOrders(viewer) {
			s.renderError(w, r, http.StatusForbidden, "Недостаточно прав для отмены заказа.")
			return
		}
		_, err = s.commands.CancelOrder(r.Context(), cred, number)
	case "restore":
		if !application.CanManageOrders(viewer) {
			s.renderError(w, r, http.StatusForbidden, "Недостаточно прав для восстановления заказа.")
			return
		}
		_, err = s.commands.RestoreOrder(r.Context(), cred, number)
	case "favorite":
		if !application.IsAdmin(viewer) {
			s.renderError(w, r, http.StatusForbidden, "Только администратор может менять избранное.")
			return
		}
		favorite, _ := strconv.ParseBool(r.FormValue("favorite"))
		_, err = s.commands.SetOrderFavorite(r.Context(), cred, number, favorite)
	}
	if err != nil {
		s.renderError(w, r, statusOr(err, http.StatusBadGateway), application.MessageOf(err, "Не удалось изменить заказ."))
		return
	}
	s.redirect(w, r, "/orders/"+url.PathEscape(number)+"?success="+url.QueryEscape("Заказ обновлён."))
}

func (s *server) orderSelectionPage(w http.ResponseWriter, r *http.Request) {
	viewer, cred, ok := s.requireProduction(w, r)
	if !ok {
		return
	}
	numbers := selectedOrderNumbers(r)
	orders := make([]contract.Order, 0, len(numbers))
	skipped := make([]string, 0)
	var categoryID int64
	for _, number := range numbers {
		order, err := s.queries.Order(r.Context(), cred, number)
		if err != nil {
			s.renderError(w, r, statusOr(err, http.StatusBadGateway), application.MessageOf(err, "Не удалось загрузить выбранные заказы."))
			return
		}
		// A stale pick would otherwise reach the API and fail there with
		// order.production_exists, after the baker has already filled the sheet.
		if order.Cancelled || order.ProductionSheetID != nil {
			skipped = append(skipped, order.Number)
			continue
		}
		currentCategoryID := int64(0)
		if order.Category != nil {
			currentCategoryID = order.Category.ID
		}
		if len(orders) > 0 && currentCategoryID != categoryID {
			s.renderError(w, r, http.StatusBadRequest, "В одну партию можно выбрать заявки только одного типа.")
			return
		}
		categoryID = currentCategoryID
		orders = append(orders, order)
	}
	notice := ""
	if len(skipped) > 0 {
		notice = "Не вошли в партию (уже отработаны или отменены): " + strings.Join(skipped, ", ") + "."
	}
	batch := make([]string, 0, len(orders))
	for _, order := range orders {
		batch = append(batch, order.Number)
	}
	s.render(w, r, http.StatusOK, page{Title: "Партия", View: "selection", Viewer: viewer, Error: notice, Data: selectionData{Orders: orders, Numbers: batch, Rows: buildProductionRows(orders, nil)}})
}

func (s *server) monitorFragment(w http.ResponseWriter, r *http.Request) {
	_, cred, ok := s.requireProduction(w, r)
	if !ok {
		return
	}
	numbers := selectedOrderNumbers(r)
	if len(numbers) == 0 {
		s.renderError(w, r, http.StatusBadRequest, "Выберите хотя бы один заказ.")
		return
	}
	data := monitorData{}
	if len(numbers) == 1 {
		result, err := s.queries.OrderMonitor(r.Context(), cred, numbers[0])
		if err != nil {
			s.renderMonitorError(w, err)
			return
		}
		data.Reports = result.Reports
		data.OrderReports = []contract.BatchOrderMonitor{{Order: result.Order, Reports: result.Reports}}
	} else {
		result, err := s.queries.BatchMonitor(r.Context(), cred, numbers)
		if err != nil {
			s.renderMonitorError(w, err)
			return
		}
		data.Reports = result.TotalReports
		data.OrderReports = result.Orders
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates.ExecuteTemplate(w, "monitor", data); err != nil {
		s.logger.Error("render monitor", "error", err)
	}
}

func (s *server) orderPreviewFragment(w http.ResponseWriter, r *http.Request) {
	_, cred, ok := s.requireViewer(w, r)
	if !ok {
		return
	}
	order, err := s.queries.Order(r.Context(), cred, r.PathValue("number"))
	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(statusOr(err, http.StatusBadGateway))
		_ = s.templates.ExecuteTemplate(w, "inline-error", application.MessageOf(err, "Не удалось открыть обзор заказа."))
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates.ExecuteTemplate(w, "order-preview", order); err != nil {
		s.logger.Error("render order preview", "error", err)
	}
}

func (s *server) renderMonitorError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(statusOr(err, http.StatusBadGateway))
	_ = s.templates.ExecuteTemplate(w, "inline-error", application.MessageOf(err, "Не удалось рассчитать тесто."))
}

func (s *server) renderOrderForm(w http.ResponseWriter, r *http.Request, viewer *contract.Me, cred application.Credentials, status int, mode string, order *contract.Order, message string) {
	body := orderWriteFromOrder(order)
	if mode == "create" && order != nil {
		body.FulfillmentDate = ""
	}
	s.renderOrderFormFromWrite(w, r, viewer, cred, status, mode, order, body, message)
}

func (s *server) renderOrderFormFromWrite(w http.ResponseWriter, r *http.Request, viewer *contract.Me, cred application.Credentials, status int, mode string, order *contract.Order, body contract.OrderWrite, message string) {
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
	departmentType := "shop"
	if mode == "update" && application.CanUseProduction(viewer) {
		departmentType = ""
	}
	shops, err := s.queries.Departments(r.Context(), cred, departmentType)
	if err != nil {
		s.renderError(w, r, statusOr(err, http.StatusBadGateway), application.MessageOf(err, "Не удалось загрузить отправителей."))
		return
	}
	categoryID := body.CategoryID
	if order != nil && order.Category != nil {
		categoryID = order.Category.ID
	}
	fromDepartmentID := int64(0)
	if body.FromDepartmentID != nil {
		fromDepartmentID = *body.FromDepartmentID
	} else if order != nil && order.FromDepartment != nil {
		fromDepartmentID = order.FromDepartment.ID
	} else if viewer.DepartmentType == "shop" {
		fromDepartmentID = viewer.DepartmentID
	}
	data := orderFormData{
		Mode:             mode,
		Order:            order,
		Categories:       categories,
		Shops:            shops,
		WorkshopSource:   mode == "create" && viewer.Role == "baker",
		Groups:           buildEditorGroups(catalog, body),
		CategoryID:       categoryID,
		FromDepartmentID: fromDepartmentID,
		FulfillmentDate:  body.FulfillmentDate,
		GeneralComment:   body.Comments.General,
	}
	title := "Новая заявка"
	if mode == "update" {
		title = "Изменить заявку"
	}
	s.render(w, r, status, page{Title: title, View: "order-form", Viewer: viewer, Error: message, Data: data})
}

func parseOrderFilters(r *http.Request) orderFiltersView {
	query := r.URL.Query()
	pageNumber, _ := strconv.Atoi(query.Get("page"))
	if pageNumber < 1 {
		pageNumber = 1
	}
	fromID, _ := strconv.ParseInt(query.Get("from_department_id"), 10, 64)
	categoryID, _ := strconv.ParseInt(query.Get("category_id"), 10, 64)
	return orderFiltersView{
		Page:             pageNumber,
		FromDepartmentID: fromID,
		CategoryID:       categoryID,
		FulfillmentDate:  query.Get("fulfillment_date"),
		FulfillmentFrom:  query.Get("fulfillment_from"),
		FulfillmentTo:    query.Get("fulfillment_to"),
	}
}

func parseOrderWrite(r *http.Request) (contract.OrderWrite, error) {
	if err := r.ParseForm(); err != nil {
		return contract.OrderWrite{}, fmt.Errorf("не удалось прочитать форму")
	}
	categoryID, _ := strconv.ParseInt(r.FormValue("category_id"), 10, 64)
	fromDepartmentID, _ := strconv.ParseInt(r.FormValue("from_department_id"), 10, 64)
	body := contract.OrderWrite{
		CategoryID:      categoryID,
		FulfillmentDate: strings.TrimSpace(r.FormValue("fulfillment_date")),
		Comments:        contract.Comments{General: strings.TrimSpace(r.FormValue("general_comment"))},
	}
	if fromDepartmentID > 0 {
		body.FromDepartmentID = &fromDepartmentID
	}
	names := r.Form["item_name"]
	quantities := r.Form["quantity"]
	reserves := r.Form["reserved_quantity"]
	comments := r.Form["item_comment"]
	for index, name := range names {
		quantity := valueAtFloat(quantities, index)
		reserved := valueAtFloat(reserves, index)
		if quantity < 0 || reserved < 0 {
			return body, fmt.Errorf("количество не может быть отрицательным")
		}
		if quantity+reserved <= 0 {
			continue
		}
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		body.Items = append(body.Items, contract.OrderItemWrite{ProductName: name, Quantity: quantity, ReservedQuantity: reserved})
		comment := strings.TrimSpace(valueAt(comments, index))
		if comment != "" {
			body.Comments.Items = append(body.Comments.Items, contract.ItemComment{ProductName: name, Comment: comment})
		}
	}
	if body.FulfillmentDate == "" {
		return body, fmt.Errorf("укажите дату выполнения")
	}
	if body.CategoryID < 1 && r.PathValue("number") == "" {
		return body, fmt.Errorf("выберите тип заявки")
	}
	if len(body.Items) == 0 {
		return body, fmt.Errorf("укажите количество хотя бы для одной позиции")
	}
	return body, nil
}

func orderWriteFromOrder(order *contract.Order) contract.OrderWrite {
	if order == nil {
		return contract.OrderWrite{}
	}
	body := contract.OrderWrite{FulfillmentDate: order.FulfillmentDate, Comments: order.Comments}
	if order.Category != nil {
		body.CategoryID = order.Category.ID
	}
	if order.FromDepartment != nil {
		id := order.FromDepartment.ID
		body.FromDepartmentID = &id
	}
	for _, item := range order.Items {
		body.Items = append(body.Items, contract.OrderItemWrite{ProductName: item.ProductName, Quantity: item.Quantity, ReservedQuantity: item.ReservedQuantity})
	}
	return body
}

func buildEditorGroups(catalog []contract.Dish, body contract.OrderWrite) []editorGroup {
	values := make(map[string]contract.OrderItemWrite, len(body.Items))
	for _, item := range body.Items {
		values[item.ProductName] = item
	}
	groups := make([]editorGroup, 0)
	indexes := make(map[string]int)
	for _, dish := range catalog {
		index, ok := indexes[dish.Theme]
		if !ok {
			index = len(groups)
			indexes[dish.Theme] = index
			groups = append(groups, editorGroup{Name: fallback(dish.Theme, "Без группы")})
		}
		value := values[dish.Name]
		groups[index].Items = append(groups[index].Items, editorItem{
			Dish: dish, Quantity: value.Quantity, Reserved: value.ReservedQuantity, Comment: commentFor(dish.Name, body.Comments.Items),
		})
	}
	return groups
}

func groupOrderItems(items []contract.OrderItem, catalog []contract.Dish) []orderItemGroup {
	themeByName := make(map[string]string, len(catalog))
	for _, dish := range catalog {
		themeByName[dish.Name] = fallback(dish.Theme, "Без группы")
	}
	groups := make([]orderItemGroup, 0)
	indexes := make(map[string]int)
	for _, item := range items {
		theme := fallback(themeByName[item.ProductName], "Без группы")
		index, ok := indexes[theme]
		if !ok {
			index = len(groups)
			indexes[theme] = index
			groups = append(groups, orderItemGroup{Name: theme})
		}
		groups[index].Items = append(groups[index].Items, item)
	}
	return groups
}

func buildOrderMatrix(orders []contract.Order, shops []contract.Department, fromValue, toValue string) []matrixRow {
	from, err := time.Parse(time.DateOnly, fromValue)
	if err != nil {
		return nil
	}
	to, err := time.Parse(time.DateOnly, toValue)
	if err != nil {
		return nil
	}
	byDateShop := make(map[string]map[int64][]contract.Order)
	knownShops := make(map[int64]struct{}, len(shops))
	for _, shop := range shops {
		knownShops[shop.ID] = struct{}{}
	}
	for _, order := range orders {
		shopID := int64(0)
		if order.FromDepartment != nil {
			shopID = order.FromDepartment.ID
		}
		if _, ok := knownShops[shopID]; !ok {
			shopID = 0
		}
		if byDateShop[order.FulfillmentDate] == nil {
			byDateShop[order.FulfillmentDate] = make(map[int64][]contract.Order)
		}
		byDateShop[order.FulfillmentDate][shopID] = append(byDateShop[order.FulfillmentDate][shopID], order)
	}
	rows := make([]matrixRow, 0, 5)
	for day := to; !day.Before(from); day = day.AddDate(0, 0, -1) {
		key := day.Format(time.DateOnly)
		row := matrixRow{Date: key, Label: weekdayLabel(day), Tone: dayTone(day)}
		for _, shop := range shops {
			row.Cells = append(row.Cells, matrixCell{Shop: shop, Orders: byDateShop[key][shop.ID]})
		}
		rows = append(rows, row)
	}
	return rows
}

func matrixShops(orders []contract.Order, shops []contract.Department) []contract.Department {
	known := make(map[int64]struct{}, len(shops))
	for _, shop := range shops {
		known[shop.ID] = struct{}{}
	}
	for _, order := range orders {
		if order.FromDepartment == nil {
			return append(shops, contract.Department{Name: "Прочее"})
		}
		if _, ok := known[order.FromDepartment.ID]; !ok {
			return append(shops, contract.Department{Name: "Прочее"})
		}
	}
	return shops
}

// weekdayLabel names the day. Today and tomorrow keep their own colour через
// tone-today/tone-tomorrow, so the label itself stays uniform.
func weekdayLabel(day time.Time) string {
	return [...]string{"Вс", "Пн", "Вт", "Ср", "Чт", "Пт", "Сб"}[int(day.Weekday())]
}

func dayTone(day time.Time) string {
	today := startOfDay(time.Now())
	switch {
	case day.Equal(today):
		return "today"
	case day.Equal(today.AddDate(0, 0, 1)):
		return "tomorrow"
	}
	return ""
}

func selectedOrderNumbers(r *http.Request) []string {
	values := append([]string(nil), r.URL.Query()["order"]...)
	if joined := r.URL.Query().Get("orders"); joined != "" {
		values = append(values, strings.Split(joined, ",")...)
	}
	seen := make(map[string]struct{})
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func startOfDay(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, value.Location())
}

func valueAt(values []string, index int) string {
	if index < 0 || index >= len(values) {
		return ""
	}
	return values[index]
}

func valueAtFloat(values []string, index int) float64 {
	value := strings.ReplaceAll(strings.TrimSpace(valueAt(values, index)), ",", ".")
	number, _ := strconv.ParseFloat(value, 64)
	return number
}

func fallback(value, fallbackValue string) string {
	if strings.TrimSpace(value) == "" {
		return fallbackValue
	}
	return value
}
