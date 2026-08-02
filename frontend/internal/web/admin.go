package web

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"bakery/frontend/internal/application"
	"bakery/internal/inbound/api/contract"
)

type adminUserView struct {
	User           contract.User
	DepartmentCode string
	DepartmentName string
}

type adminUsersData struct {
	Users       []adminUserView
	Departments []contract.Department
}

type themeGroup struct {
	Name   string
	Dishes []contract.Dish
}

type dishGroup struct {
	Category contract.Category
	Themes   []themeGroup
	Count    int
}

type adminDishesData struct {
	Dishes        []contract.Dish
	Categories    []contract.Category
	Groups        []dishGroup
	Uncategorized []themeGroup
	Available     []contract.AvailableDish
	Search        string
	ActiveTab     int64 // category id of the open tab; 0 = «Без типа»
}

// groupDishesByTheme buckets dishes by their Theme («группа»), preserving the
// catalog order. Same grouping the order editor uses (buildEditorGroups).
func groupDishesByTheme(dishes []contract.Dish) []themeGroup {
	var themes []themeGroup
	index := map[string]int{}
	for _, dish := range dishes {
		name := dish.Theme
		if name == "" {
			name = "Без группы"
		}
		i, ok := index[name]
		if !ok {
			i = len(themes)
			index[name] = i
			themes = append(themes, themeGroup{Name: name})
		}
		themes[i].Dishes = append(themes[i].Dishes, dish)
	}
	return themes
}

// dishesRedirect builds the post-mutation URL, keeping the active category tab
// open and carrying an optional success flash.
func dishesRedirect(tab int64, success string) string {
	v := url.Values{}
	if tab > 0 {
		v.Set("tab", strconv.FormatInt(tab, 10))
	}
	if success != "" {
		v.Set("success", success)
	}
	return "/admin/dishes?" + v.Encode()
}

type adminTechCardsData struct {
	Categories []contract.TechCardCategory
}

func (s *server) adminUsersPage(w http.ResponseWriter, r *http.Request) {
	viewer, cred, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	s.renderAdminUsers(w, r, viewer, cred, http.StatusOK, "")
}

func (s *server) adminUserCreate(w http.ResponseWriter, r *http.Request) {
	viewer, cred, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		s.renderAdminUsers(w, r, viewer, cred, http.StatusBadRequest, "Не удалось прочитать форму.")
		return
	}
	body := contract.UserCreate{
		Username:         strings.TrimSpace(r.FormValue("username")),
		Password:         r.FormValue("password"),
		TelegramUsername: normalizeTelegramUsername(r.FormValue("telegram_username")),
		Role:             strings.TrimSpace(r.FormValue("role")),
		DepartmentCode:   strings.TrimSpace(r.FormValue("department_code")),
	}
	if body.Username == "" || body.Password == "" || body.Role == "" {
		s.renderAdminUsers(w, r, viewer, cred, http.StatusUnprocessableEntity, "Логин, пароль и роль обязательны.")
		return
	}
	if _, err := s.commands.CreateUser(r.Context(), cred, body); err != nil {
		s.renderAdminUsers(w, r, viewer, cred, statusOr(err, http.StatusBadGateway), application.MessageOf(err, "Не удалось создать пользователя."))
		return
	}
	s.redirect(w, r, "/admin/users?success="+url.QueryEscape("Пользователь создан."))
}

func (s *server) adminUserUpdate(w http.ResponseWriter, r *http.Request) {
	viewer, cred, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.renderAdminUsers(w, r, viewer, cred, http.StatusBadRequest, "Некорректный пользователь.")
		return
	}
	username := strings.TrimSpace(r.FormValue("username"))
	role := strings.TrimSpace(r.FormValue("role"))
	departmentCode := strings.TrimSpace(r.FormValue("department_code"))
	body := contract.UserUpdate{Username: &username, Role: &role, DepartmentCode: &departmentCode}
	if password := r.FormValue("password"); password != "" {
		body.Password = &password
	}
	if _, err := s.commands.UpdateUser(r.Context(), cred, id, body); err != nil {
		s.renderAdminUsers(w, r, viewer, cred, statusOr(err, http.StatusBadGateway), application.MessageOf(err, "Не удалось обновить пользователя."))
		return
	}
	s.redirect(w, r, "/admin/users?success="+url.QueryEscape("Пользователь обновлён."))
}

func (s *server) adminUserDelete(w http.ResponseWriter, r *http.Request) {
	_, cred, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.renderError(w, r, http.StatusBadRequest, "Некорректный пользователь.")
		return
	}
	if err := s.commands.DeleteUser(r.Context(), cred, id); err != nil {
		s.renderError(w, r, statusOr(err, http.StatusBadGateway), application.MessageOf(err, "Не удалось удалить пользователя."))
		return
	}
	s.redirect(w, r, "/admin/users?success="+url.QueryEscape("Пользователь удалён."))
}

func (s *server) renderAdminUsers(w http.ResponseWriter, r *http.Request, viewer *contract.Me, cred application.Credentials, status int, message string) {
	users, err := s.queries.Users(r.Context(), cred)
	if err != nil {
		s.renderError(w, r, statusOr(err, http.StatusBadGateway), application.MessageOf(err, "Не удалось загрузить пользователей."))
		return
	}
	departments, err := s.queries.AdminDepartments(r.Context(), cred)
	if err != nil {
		s.renderError(w, r, statusOr(err, http.StatusBadGateway), application.MessageOf(err, "Не удалось загрузить подразделения."))
		return
	}
	byID := make(map[int64]contract.Department, len(departments))
	for _, department := range departments {
		byID[department.ID] = department
	}
	data := adminUsersData{Departments: departments}
	for _, user := range users {
		row := adminUserView{User: user}
		if user.DepartmentID != nil {
			row.DepartmentCode = byID[*user.DepartmentID].Code
			row.DepartmentName = byID[*user.DepartmentID].Name
		}
		data.Users = append(data.Users, row)
	}
	s.render(w, r, status, page{Title: "Пользователи", View: "admin-users", Viewer: viewer, Error: message, Success: queryMessage(r, "success"), Data: data})
}

func (s *server) adminDishesPage(w http.ResponseWriter, r *http.Request) {
	viewer, cred, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	s.renderAdminDishes(w, r, viewer, cred, http.StatusOK, "")
}

func (s *server) adminDishCreate(w http.ResponseWriter, r *http.Request) {
	viewer, cred, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	body := parseDishWrite(r)
	if body.Code == "" || body.Name == "" || body.Theme == "" {
		s.renderAdminDishes(w, r, viewer, cred, http.StatusUnprocessableEntity, "Код, название и группа блюда обязательны.")
		return
	}
	if body.CategoryID == nil {
		s.renderAdminDishes(w, r, viewer, cred, http.StatusUnprocessableEntity, "Выберите тип заявки для блюда.")
		return
	}
	if _, err := s.commands.CreateDish(r.Context(), cred, body); err != nil {
		s.renderAdminDishes(w, r, viewer, cred, statusOr(err, http.StatusBadGateway), application.MessageOf(err, "Не удалось добавить блюдо."))
		return
	}
	s.redirect(w, r, dishesRedirect(*body.CategoryID, "Блюдо добавлено."))
}

func (s *server) adminDishUpdate(w http.ResponseWriter, r *http.Request) {
	viewer, cred, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	body := parseDishWrite(r)
	body.Code = r.PathValue("code")
	if body.CategoryID == nil {
		s.renderAdminDishes(w, r, viewer, cred, http.StatusUnprocessableEntity, "Выберите тип заявки для блюда.")
		return
	}
	if _, err := s.commands.UpdateDish(r.Context(), cred, body.Code, body); err != nil {
		s.renderAdminDishes(w, r, viewer, cred, statusOr(err, http.StatusBadGateway), application.MessageOf(err, "Не удалось обновить блюдо."))
		return
	}
	s.redirect(w, r, dishesRedirect(*body.CategoryID, "Блюдо обновлено."))
}

func (s *server) adminDishDelete(w http.ResponseWriter, r *http.Request) {
	_, cred, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	if err := s.commands.DeleteDish(r.Context(), cred, r.PathValue("code")); err != nil {
		s.renderError(w, r, statusOr(err, http.StatusBadGateway), application.MessageOf(err, "Не удалось удалить блюдо."))
		return
	}
	s.redirect(w, r, dishesRedirect(parseInt64(r.FormValue("tab")), "Блюдо удалено."))
}

func (s *server) adminDishReorder(w http.ResponseWriter, r *http.Request) {
	_, cred, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		s.renderError(w, r, http.StatusBadRequest, "Не удалось прочитать порядок блюд.")
		return
	}
	// Drag-and-drop posts the full catalogue order (every dish code, all tabs, in
	// display order) via fetch; nothing to do without it.
	codes := r.Form["codes"]
	if len(codes) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if err := s.commands.ReorderDishes(r.Context(), cred, codes); err != nil {
		s.renderError(w, r, statusOr(err, http.StatusBadGateway), application.MessageOf(err, "Не удалось изменить порядок блюд."))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) adminCategoryCreate(w http.ResponseWriter, r *http.Request) {
	viewer, cred, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	body := parseCategoryWrite(r)
	category, err := s.commands.CreateCategory(r.Context(), cred, body)
	if err != nil {
		s.renderAdminDishes(w, r, viewer, cred, statusOr(err, http.StatusBadGateway), application.MessageOf(err, "Не удалось создать тип заявки."))
		return
	}
	s.redirect(w, r, dishesRedirect(category.ID, "Тип заявки создан."))
}

func (s *server) adminCategoryUpdate(w http.ResponseWriter, r *http.Request) {
	viewer, cred, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if _, err := s.commands.UpdateCategory(r.Context(), cred, id, parseCategoryWrite(r)); err != nil {
		s.renderAdminDishes(w, r, viewer, cred, statusOr(err, http.StatusBadGateway), application.MessageOf(err, "Не удалось обновить тип заявки."))
		return
	}
	s.redirect(w, r, dishesRedirect(id, "Тип заявки обновлён."))
}

func (s *server) adminCategoryDelete(w http.ResponseWriter, r *http.Request) {
	_, cred, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err := s.commands.DeleteCategory(r.Context(), cred, id); err != nil {
		s.renderError(w, r, statusOr(err, http.StatusBadGateway), application.MessageOf(err, "Не удалось удалить тип заявки."))
		return
	}
	s.redirect(w, r, dishesRedirect(0, "Тип заявки удалён."))
}

func (s *server) renderAdminDishes(w http.ResponseWriter, r *http.Request, viewer *contract.Me, cred application.Credentials, status int, message string) {
	dishes, err := s.queries.Dishes(r.Context(), cred)
	if err != nil {
		s.renderError(w, r, statusOr(err, http.StatusBadGateway), application.MessageOf(err, "Не удалось загрузить каталог."))
		return
	}
	categories, err := s.queries.Categories(r.Context(), cred)
	if err != nil {
		s.renderError(w, r, statusOr(err, http.StatusBadGateway), application.MessageOf(err, "Не удалось загрузить типы заявок."))
		return
	}
	search := strings.TrimSpace(r.URL.Query().Get("q"))
	var available []contract.AvailableDish
	if search != "" {
		available, err = s.queries.AvailableDishes(r.Context(), cred, search)
		if err != nil {
			message = application.MessageOf(err, "Не удалось выполнить поиск в iiko.")
			status = statusOr(err, http.StatusBadGateway)
		}
	}
	data := adminDishesData{Dishes: dishes, Categories: categories, Available: available, Search: search}
	// Group dishes into per-category tabs; anything without a type falls into a
	// separate bucket so the admin can still find and fix legacy rows.
	for _, category := range categories {
		var picked []contract.Dish
		for _, dish := range dishes {
			if dish.CategoryID != nil && *dish.CategoryID == category.ID {
				picked = append(picked, dish)
			}
		}
		data.Groups = append(data.Groups, dishGroup{Category: category, Themes: groupDishesByTheme(picked), Count: len(picked)})
	}
	var uncategorized []contract.Dish
	for _, dish := range dishes {
		if dish.CategoryID == nil {
			uncategorized = append(uncategorized, dish)
		}
	}
	data.Uncategorized = groupDishesByTheme(uncategorized)
	// Keep the requested tab open across reloads; default to the first type.
	data.ActiveTab = parseInt64(r.URL.Query().Get("tab"))
	if data.ActiveTab == 0 && len(data.Groups) > 0 {
		data.ActiveTab = data.Groups[0].Category.ID
	}
	s.render(w, r, status, page{Title: "Меню", View: "admin-dishes", Viewer: viewer, Error: message, Success: queryMessage(r, "success"), Data: data})
}

func (s *server) adminTechCardsPage(w http.ResponseWriter, r *http.Request) {
	viewer, cred, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	categories, err := s.queries.DishTechCards(r.Context(), cred)
	if err != nil {
		s.renderError(w, r, statusOr(err, http.StatusBadGateway), application.MessageOf(err, "Не удалось загрузить тех.карты."))
		return
	}
	data := adminTechCardsData{Categories: categories}
	s.render(w, r, http.StatusOK, page{Title: "Тех.карты", View: "admin-techcards", Viewer: viewer, Data: data})
}

func parseDishWrite(r *http.Request) contract.DishWrite {
	categoryID := parseInt64(r.FormValue("category_id"))
	var category *int64
	if categoryID > 0 {
		category = &categoryID
	}
	return contract.DishWrite{
		Code: strings.TrimSpace(r.FormValue("code")), Name: strings.TrimSpace(r.FormValue("name")),
		Theme: strings.TrimSpace(r.FormValue("theme")), CategoryID: category, SortOrder: parseInt64(r.FormValue("sort_order")),
	}
}

func parseCategoryWrite(r *http.Request) contract.CategoryWrite {
	codes := strings.FieldsFunc(r.FormValue("monitor_codes"), func(r rune) bool { return r == ',' || r == '\n' || r == ';' })
	for index := range codes {
		codes[index] = strings.TrimSpace(codes[index])
	}
	return contract.CategoryWrite{
		Code: strings.TrimSpace(r.FormValue("code")), Letter: strings.TrimSpace(r.FormValue("letter")),
		Name: strings.TrimSpace(r.FormValue("name")), Color: strings.TrimSpace(r.FormValue("color")),
		SortOrder: parseInt64(r.FormValue("sort_order")), MonitorCodes: codes,
	}
}

func normalizeTelegramUsername(value string) string {
	return strings.TrimPrefix(strings.TrimSpace(value), "@")
}
