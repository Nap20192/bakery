package bot

import (
	"context"
	"fmt"
	"html"
	"log/slog"
	"strings"

	"bakery/internal/domain"

	tele "gopkg.in/telebot.v3"
)

func (b *OrderBot) handleStart(c tele.Context) error {
	return c.Send(
		"<b>orderbot</b>\n\n"+
			"Отправьте заказ одним сообщением в формате:\n\n"+
			"<code>код название количество</code>\n\n"+
			"После проверки бот покажет найденные позиции, ошибки и кнопку отправки.\n\n"+
			"/login username password - войти\n"+
			"/adduser username password role - добавить пользователя\n"+
			"/addgroup product_code - добавить группу\n"+
			"/groups - посмотреть группы\n"+
			"/orders - посмотреть заказы\n"+
			"/monitor order_number group_code - мониторинг заказа\n"+
			"/template - стандартный шаблон\n"+
			"/cancel - отменить неподтвержденный заказ",
		tele.ModeHTML,
	)
}

func (b *OrderBot) handleLogin(c tele.Context) error {
	args := strings.Fields(c.Message().Payload)
	if len(args) != 2 {
		return c.Send("Формат: /login username password")
	}

	user, err := b.authSvc.LoginTelegramUser(context.Background(), c.Sender().ID, args[0], args[1])
	if err != nil {
		slog.Warn("login failed", "user_id", c.Sender().ID, "username", args[0], "error", err)
		return c.Send("Не удалось войти: проверьте username и password.")
	}

	c.Set(authUserContextKey, user)
	return c.Send(fmt.Sprintf("Вход выполнен. Роль: %s", user.Role))
}

func (b *OrderBot) handleAddUser(c tele.Context) error {
	args := strings.Fields(c.Message().Payload)
	if len(args) != 3 {
		return c.Send("Формат: /adduser username password role\nРоли: admin, baker, client")
	}

	user, err := b.authSvc.CreateUserWithPassword(context.Background(), domain.PasswordAuthUserInput{
		Username: args[0],
		Password: args[1],
		Role:     args[2],
	})
	if err != nil {
		return c.Send(fmt.Sprintf("Ошибка создания пользователя: %v", err))
	}

	return c.Send(fmt.Sprintf("Пользователь %s создан. Роль: %s", user.Username, user.Role))
}

func (b *OrderBot) handleAddGroup(c tele.Context) error {
	args := strings.Fields(c.Message().Payload)
	if len(args) != 1 {
		return c.Send("Формат: /addgroup product_code")
	}

	group, err := b.groupSvc.AddGroupByProductCode(context.Background(), domain.GroupInput{
		Code: args[0],
	})
	if err != nil {
		return c.Send(fmt.Sprintf("Ошибка создания группы: %v", err))
	}

	return c.Send(fmt.Sprintf("Группа %s создана", group.Name))
}

func (b *OrderBot) handleGroups(c tele.Context) error {
	groups, err := b.groupSvc.ListGroups(context.Background())
	if err != nil {
		return c.Send(fmt.Sprintf("Ошибка получения групп: %v", err))
	}
	if len(groups) == 0 {
		return c.Send("Групп пока нет.")
	}

	var sb strings.Builder
	sb.WriteString("<b>Группы</b>\n\n")
	for _, group := range groups {
		sb.WriteString(fmt.Sprintf("<code>%s</code> %s\n", html.EscapeString(group.Code), html.EscapeString(group.Name)))
	}
	return c.Send(sb.String(), tele.ModeHTML)
}

func (b *OrderBot) handleOrders(c tele.Context) error {
	orders, err := b.orderSvc.ListOrders(context.Background(), 10)
	if err != nil {
		return c.Send(fmt.Sprintf("Ошибка получения заказов: %v", err))
	}
	if len(orders) == 0 {
		return c.Send("Заказов пока нет.")
	}

	var sb strings.Builder
	sb.WriteString("<b>Последние заказы</b>\n\n")
	for _, order := range orders {
		sb.WriteString(fmt.Sprintf("<code>%s</code>\n", html.EscapeString(order.Number)))
		for _, item := range order.Items {
			sb.WriteString(fmt.Sprintf("• %s - %s\n", html.EscapeString(item.ProductName), formatQuantity(item.Quantity)))
		}
		sb.WriteString("\n")
	}
	return c.Send(sb.String(), tele.ModeHTML)
}

func (b *OrderBot) handleMonitor(c tele.Context) error {
	args := strings.Fields(c.Message().Payload)
	if len(args) != 2 {
		return c.Send("Формат: /monitor order_number group_code")
	}

	order, err := b.orderSvc.GetOrderByNumber(context.Background(), args[0])
	if err != nil {
		return c.Send(fmt.Sprintf("Заказ %s не найден.", args[0]))
	}
	group, err := b.groupSvc.GetGroupByCode(context.Background(), args[1])
	if err != nil {
		return c.Send(fmt.Sprintf("Группа %s не найдена.", args[1]))
	}

	report, err := b.monitorSvc.GetIngredientsByGroup(context.Background(), group, order)
	if err != nil {
		return c.Send(fmt.Sprintf("Ошибка мониторинга: %v", err))
	}
	return c.Send(formatMonitorReport(report), tele.ModeHTML)
}

func (b *OrderBot) handleTemplate(c tele.Context) error {
	template, err := b.orderSvc.GetTemplate(context.Background())
	if err != nil {
		return c.Send(fmt.Sprintf("Ошибка получения шаблона: %v", err))
	}
	return c.Send("<pre>"+html.EscapeString(template)+"</pre>", tele.ModeHTML)
}

func (b *OrderBot) handleCancel(c tele.Context) error {
	b.clearSession(c.Sender().ID)
	return c.Send("Заказ отменен.")
}

func (b *OrderBot) handleText(c tele.Context) error {
	text := strings.TrimSpace(c.Text())
	if strings.HasPrefix(text, "/") {
		return c.Send("Неизвестная команда.\n\n/start - список команд")
	}
	if text == "" {
		return c.Send("Отправьте batch-заказ одним сообщением.")
	}
	if err := b.ensurePermission(c, permCreateOrder); err != nil {
		return err
	}
	return b.handleBulkOrder(c, text)
}

func (b *OrderBot) handleConfirm(c tele.Context) error {
	var items []domain.OrderItem
	b.updateSession(c.Sender().ID, func(s *session) {
		items = make([]domain.OrderItem, len(s.items))
		copy(items, s.items)
		s.items = nil
	})

	_ = c.Respond()
	if len(items) == 0 {
		return c.Send("Заказ пустой или уже отправлен.")
	}

	order, err := b.orderSvc.CreateOrder(context.Background(), domain.CreateOrderInput{
		Items: items,
	})
	if err != nil {
		slog.Error("create order failed", "user_id", c.Sender().ID, "error", err)
		return c.Send(fmt.Sprintf("Ошибка создания заказа: %v", err))
	}

	slog.Info("order created", "number", order.Number, "user_id", c.Sender().ID, "items", len(items))
	return c.Send(formatOrderSummary(order), tele.ModeMarkdown)
}

func (b *OrderBot) handleCancelCallback(c tele.Context) error {
	b.clearSession(c.Sender().ID)
	_ = c.Respond()
	return c.Send("Заказ отменен.")
}

func (b *OrderBot) ensurePermission(c tele.Context, permission string) error {
	user, err := b.authUserFromContext(c)
	if err != nil {
		return err
	}
	if !userHasPermission(user.Role, permission) {
		return c.Send(fmt.Sprintf("Доступ запрещен: недостаточно прав (%s).", permission))
	}
	return nil
}

func (b *OrderBot) handleBulkOrder(c tele.Context, text string) error {
	result := b.orderSvc.ValidateBulkOrder(context.Background(), text)
	if len(result.ValidItems) == 0 {
		return c.Send(formatValidationErrors(result.Errors), tele.ModeHTML)
	}

	b.updateSession(c.Sender().ID, func(s *session) {
		s.items = result.ValidItems
	})

	var sb strings.Builder
	sb.WriteString("<b>Предпросмотр batch-заказа</b>")
	sb.WriteString(fmt.Sprintf("\n\n<b>Распознано позиций: %d</b>\n", len(result.ValidItems)))
	for _, it := range result.ValidItems {
		sb.WriteString(fmt.Sprintf(
			"• %s %s - %s\n",
			html.EscapeString(it.Code),
			html.EscapeString(it.ProductName),
			formatQuantity(it.Quantity),
		))
	}

	if len(result.Errors) > 0 {
		sb.WriteString(fmt.Sprintf("\n<b>Ошибки: %d</b>\n", len(result.Errors)))
		for _, errText := range result.Errors {
			sb.WriteString("— ")
			sb.WriteString(html.EscapeString(errText))
			sb.WriteString("\n")
		}
		sb.WriteString("\nБудут отправлены только корректные позиции. Чтобы исправить заказ, отправьте новый текст целиком.")
	} else {
		sb.WriteString("\nОшибок нет.")
	}

	sb.WriteString("\n\n<blockquote expandable>")
	sb.WriteString(html.EscapeString(formatCopyableOrder(result.ValidItems)))
	sb.WriteString("</blockquote>")

	markup := &tele.ReplyMarkup{}
	markup.Inline(
		markup.Row(
			markup.Data("Отправить", "confirm"),
			markup.Data("Отмена", "cancel_cb"),
		),
	)
	return c.Send(sb.String(), tele.ModeHTML, markup)
}

func formatValidationErrors(errors []string) string {
	if len(errors) == 0 {
		return "Не удалось распознать позиции в сообщении."
	}

	var sb strings.Builder
	sb.WriteString("<b>Заказ не распознан</b>\n\n")
	for _, errText := range errors {
		sb.WriteString("— ")
		sb.WriteString(html.EscapeString(errText))
		sb.WriteString("\n")
	}
	sb.WriteString("\nОтправьте новый заказ целиком.")
	return sb.String()
}

func formatCopyableOrder(items []domain.OrderItem) string {
	var sb strings.Builder
	for _, it := range items {
		sb.WriteString(fmt.Sprintf("%s %s %s\n", it.Code, it.ProductName, formatQuantity(it.Quantity)))
	}
	return strings.TrimSpace(sb.String())
}

func formatOrderSummary(order domain.Order) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("✅ *Заказ %s отправлен*\n\n", order.Number))
	sb.WriteString("*Состав заказа:*\n")
	for _, it := range order.Items {
		sb.WriteString(fmt.Sprintf("• `%s` %s - %s\n", it.Code, it.ProductName, formatQuantity(it.Quantity)))
	}
	return sb.String()
}

func formatMonitorReport(report domain.GroupIngredientsReport) string {
	var sb strings.Builder
	sb.WriteString("<b>Мониторинг</b>\n\n")
	sb.WriteString(fmt.Sprintf("Группа: <code>%s</code> %s\n", html.EscapeString(report.Group.Code), html.EscapeString(report.Group.Name)))
	sb.WriteString(fmt.Sprintf(
		"Продукт: <code>%s</code> %s\n",
		html.EscapeString(report.Ingredient.ProductCode),
		html.EscapeString(report.Ingredient.ProductName),
	))
	sb.WriteString(fmt.Sprintf("Итого: %s %s\n\n", formatQuantity(report.Ingredient.Quantity), html.EscapeString(report.Ingredient.Unit)))

	if len(report.Breakdown) > 0 {
		sb.WriteString("<b>По блюдам</b>\n")
		for _, item := range report.Breakdown {
			sb.WriteString(fmt.Sprintf(
				"• <code>%s</code> %s: %s %s\n",
				html.EscapeString(item.OrderItemCode),
				html.EscapeString(item.OrderItemName),
				formatQuantity(item.IngredientQuantity),
				html.EscapeString(report.Ingredient.Unit),
			))
		}
	}

	if len(report.Warnings) > 0 {
		sb.WriteString("\n<b>Предупреждения</b>\n")
		for _, warning := range report.Warnings {
			sb.WriteString("— ")
			sb.WriteString(html.EscapeString(warning))
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

func formatQuantity(quantity float64) string {
	result := fmt.Sprintf("%.3f", quantity)
	result = strings.TrimRight(result, "0")
	result = strings.TrimRight(result, ".")
	if result == "" {
		return "0"
	}
	return result
}
