package web

import (
	"net/http"
	"strconv"
	"strings"

	"bakery/frontend/internal/application"
	"bakery/internal/inbound/api/contract"
)

type calculatorItem struct {
	Dish     contract.Dish
	Quantity float64
}

type calculatorGroup struct {
	Name  string
	Items []calculatorItem
}

type calculatorData struct {
	Categories []contract.Category
	Groups     []calculatorGroup
	CategoryID int64
	Reports    []contract.MonitorReport
}

func (s *server) calculatorPage(w http.ResponseWriter, r *http.Request) {
	viewer, cred, ok := s.requireProduction(w, r)
	if !ok {
		return
	}
	s.renderCalculator(w, r, viewer, cred, http.StatusOK, contract.DoughCalcRequest{}, "")
}

func (s *server) calculateDough(w http.ResponseWriter, r *http.Request) {
	viewer, cred, ok := s.requireProduction(w, r)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		s.renderError(w, r, http.StatusBadRequest, "Не удалось прочитать форму.")
		return
	}
	request := contract.DoughCalcRequest{CategoryID: parseInt64(r.FormValue("category_id"))}
	codes := r.Form["code"]
	names := r.Form["product_name"]
	quantities := r.Form["quantity"]
	for index, code := range codes {
		quantity := valueAtFloat(quantities, index)
		if quantity <= 0 {
			continue
		}
		request.Items = append(request.Items, contract.DoughCalcItem{Code: code, ProductName: valueAt(names, index), Quantity: quantity})
	}
	if request.CategoryID < 1 {
		s.renderCalculator(w, r, viewer, cred, http.StatusUnprocessableEntity, request, "Выберите тип заявки.")
		return
	}
	if len(request.Items) == 0 {
		s.renderCalculator(w, r, viewer, cred, http.StatusUnprocessableEntity, request, "Укажите количество хотя бы для одного блюда.")
		return
	}
	reports, err := s.commands.CalculateDough(r.Context(), cred, request)
	if err != nil {
		s.renderCalculator(w, r, viewer, cred, statusOr(err, http.StatusBadGateway), request, application.MessageOf(err, "Не удалось рассчитать тесто."))
		return
	}
	s.renderCalculatorWithReports(w, r, viewer, cred, http.StatusOK, request, reports, "")
}

func (s *server) renderCalculator(w http.ResponseWriter, r *http.Request, viewer *contract.Me, cred application.Credentials, status int, request contract.DoughCalcRequest, message string) {
	s.renderCalculatorWithReports(w, r, viewer, cred, status, request, nil, message)
}

func (s *server) renderCalculatorWithReports(w http.ResponseWriter, r *http.Request, viewer *contract.Me, cred application.Credentials, status int, request contract.DoughCalcRequest, reports []contract.MonitorReport, message string) {
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
	quantities := make(map[string]float64, len(request.Items))
	for _, item := range request.Items {
		quantities[item.Code] = item.Quantity
	}
	groups := make([]calculatorGroup, 0)
	indexes := make(map[string]int)
	for _, dish := range catalog {
		index, ok := indexes[dish.Theme]
		if !ok {
			index = len(groups)
			indexes[dish.Theme] = index
			groups = append(groups, calculatorGroup{Name: fallback(dish.Theme, "Без группы")})
		}
		groups[index].Items = append(groups[index].Items, calculatorItem{Dish: dish, Quantity: quantities[dish.Code]})
	}
	data := calculatorData{Categories: categories, Groups: groups, CategoryID: request.CategoryID, Reports: reports}
	s.render(w, r, status, page{Title: "Калькулятор теста", View: "calculator", Viewer: viewer, Error: message, Data: data})
}

func categoryIDOf(order contract.Order) int64 {
	if order.Category == nil {
		return 0
	}
	return order.Category.ID
}

func parseCategoryID(value string) int64 {
	id, _ := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	return id
}
