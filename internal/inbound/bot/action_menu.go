package bot

import (
	"strings"

	"bakery/internal/app"
	accessdomain "bakery/internal/domain/access"

	tele "gopkg.in/telebot.v3"
)

type actionMenu struct {
	text   string
	markup *tele.ReplyMarkup
}

func (b *OrderBot) sendActionMenu(c tele.Context) error {
	menu := b.actionMenu(c)
	if menu.text == "" {
		return nil
	}
	return sendText(c, menu.text, menu.markup)
}

func (b *OrderBot) actionMenu(c tele.Context) actionMenu {
	user, ok := b.currentUser(c)
	if !ok {
		return guestActionMenu()
	}
	departmentType := b.userDepartmentType(c, user)
	if strings.EqualFold(user.Role, accessdomain.RoleAdmin) {
		return adminActionMenu()
	}
	if departmentType == string(app.DepartmentTypeWorkshop) {
		return workshopActionMenu()
	}
	return shopActionMenu()
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

func guestActionMenu() actionMenu {
	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(
		markup.Data("Выбрать магазин", "dept_shop"),
		markup.Data("Выбрать цех", "dept_workshop"),
	))
	return actionMenu{
		text:   "Доступные действия:\n/start - выбрать локацию\n/login username password - войти",
		markup: markup,
	}
}

func shopActionMenu() actionMenu {
	markup := &tele.ReplyMarkup{}
	markup.Inline(
		markup.Row(
			markup.Data("Шаблоны", "open_templates"),
			markup.Data("Последние заказы", "open_orders"),
		),
		markup.Row(markup.Data("Отменить текущий заказ", "cancel_cb")),
	)
	return actionMenu{
		text: "Доступные действия:\n" +
			"Отправьте позиции сообщением, чтобы добавить их в текущий заказ.\n" +
			"/templates - выбрать шаблон\n" +
			"/orders - последние заказы\n" +
			"/cancel - отменить текущий заказ",
		markup: markup,
	}
}

func workshopActionMenu() actionMenu {
	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(
		markup.Data("Последние заказы", "open_orders"),
		markup.Data("Шаблоны", "open_templates"),
	))
	return actionMenu{
		text: "Доступные действия:\n" +
			"/orders - последние заказы\n" +
			"/order order_number - открыть заказ\n" +
			"/monitor order_number - посмотреть расход",
		markup: markup,
	}
}

func adminActionMenu() actionMenu {
	markup := &tele.ReplyMarkup{}
	markup.Inline(
		markup.Row(
			markup.Data("Последние заказы", "open_orders"),
			markup.Data("Шаблоны", "open_templates"),
		),
		markup.Row(
			markup.Data("Добавить шаблон", "start_addtemplate"),
			markup.Data("Sync iiko", "run_sync"),
		),
	)
	return actionMenu{
		text: "Доступные действия администратора:\n" +
			"/orders - последние заказы\n" +
			"/templates - шаблоны\n" +
			"/addtemplate - добавить шаблон\n" +
			"/sync - синхронизировать iiko\n" +
			"/techcard code - техкарта",
		markup: markup,
	}
}
