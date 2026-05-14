package bot

import (
	"strings"

	"bakery/internal/app"
	accessdomain "bakery/internal/domain/access"

	tele "gopkg.in/telebot.v3"
)

const (
	actionChooseShop     = "Выбрать магазин"
	actionChooseWorkshop = "Выбрать цех"
	actionTemplates      = "Шаблоны"
	actionOrders         = "Последние заказы"
	actionSubmitOrder    = "Отправить заказ"
	actionUpdateOrder    = "Обновить заказ"
	actionCancelOrder    = "Отменить заказ"
	actionCurrentOrder   = "Текущий заказ"
	actionAddTemplate    = "Добавить шаблон"
	actionSync           = "Sync iiko"
)

func (b *OrderBot) actionMarkup(c tele.Context) *tele.ReplyMarkup {
	user, ok := b.currentUser(c)
	if !ok {
		return replyKeyboard(
			[]string{actionChooseShop, actionChooseWorkshop},
		)
	}
	if strings.EqualFold(user.Role, accessdomain.RoleAdmin) {
		rows := [][]string{
			{actionOrders, actionTemplates},
			{actionAddTemplate, actionSync},
		}
		if action, ok := b.currentOrderAction(c); ok {
			rows = append(rows, []string{actionCurrentOrder})
			rows = append(rows, []string{action})
			rows = append(rows, []string{actionCancelOrder})
		}
		return replyKeyboard(rows...)
	}
	if b.userDepartmentType(c, user) == string(app.DepartmentTypeWorkshop) {
		return replyKeyboard(
			[]string{actionOrders, actionTemplates},
		)
	}
	rows := [][]string{
		{actionTemplates, actionOrders},
	}
	if action, ok := b.currentOrderAction(c); ok {
		rows = append(rows, []string{actionCurrentOrder})
		rows = append(rows, []string{action})
		rows = append(rows, []string{actionCancelOrder})
	}
	return replyKeyboard(rows...)
}

func (b *OrderBot) currentOrderAction(c tele.Context) (string, bool) {
	if c.Sender() == nil {
		return "", false
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	s := b.sessions[c.Sender().ID]
	if s == nil || len(s.items) == 0 {
		return "", false
	}
	if strings.TrimSpace(s.editOrderNumber) != "" {
		return actionUpdateOrder, true
	}
	return actionSubmitOrder, true
}

func replyKeyboard(rows ...[]string) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{ResizeKeyboard: true}
	replyRows := make([]tele.Row, 0, len(rows))
	for _, labels := range rows {
		buttons := make([]tele.Btn, 0, len(labels))
		for _, label := range labels {
			buttons = append(buttons, markup.Text(label))
		}
		replyRows = append(replyRows, markup.Row(buttons...))
	}
	markup.Reply(replyRows...)
	return markup
}

func (b *OrderBot) currentUser(c tele.Context) (accessdomain.AuthUser, bool) {
	if raw := c.Get(authUserContextKey); raw != nil {
		if user, ok := raw.(accessdomain.AuthUser); ok {
			return user, true
		}
	}
	if b.authSvc == nil || c.Sender() == nil {
		return accessdomain.AuthUser{}, false
	}
	user, err := b.authSvc.GetUserByTelegramID(requestContext(c), c.Sender().ID)
	if err != nil {
		return accessdomain.AuthUser{}, false
	}
	c.Set(authUserContextKey, user)
	return user, true
}

func (b *OrderBot) userDepartmentType(c tele.Context, user accessdomain.AuthUser) string {
	if b.departmentSvc == nil || user.DepartmentID == nil {
		return ""
	}
	department, err := b.departmentSvc.GetByID(requestContext(c), *user.DepartmentID)
	if err != nil {
		return ""
	}
	return department.Type
}
