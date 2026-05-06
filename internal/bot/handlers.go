package bot

import (
	"context"
	"fmt"
	"html"
	"log"
	"strings"

	"bakery/internal/domain"

	tele "gopkg.in/telebot.v3"
)

func (b *OrderBot) handleStart(c tele.Context) error {
	return c.Send(
		"🍞 *bakery bot*\n\n"+
			"*Клиент:*\n"+
			"Отправьте заявку одним сообщением в формате:\n\n"+
			"`Локация`\n"+
			"`Категория`\n"+
			"`Продукт количество`\n\n"+
			"/template — отправить стандартный шаблон\n"+
			"/cancel — отменить неподтверждённую заявку\n\n"+
			"*Пекарь/Админ:*\n"+
			"/orders — последние заказы\n"+
			"Можно отправить один или несколько номеров заказа, чтобы посмотреть детали.",
		tele.ModeMarkdown,
	)
}

func (b *OrderBot) handleTemplate(c tele.Context) error {
	return c.Send("<pre>"+html.EscapeString(defaultOrderTemplate)+"</pre>", tele.ModeHTML)
}

func (b *OrderBot) handleCancel(c tele.Context) error {
	b.clearSession(c.Sender().ID)
	return c.Send("Заявка отменена.")
}

func (b *OrderBot) handleText(c tele.Context) error {
	text := strings.TrimSpace(c.Text())
	if len(splitNumbers(text)) > 0 {
		if err := b.ensurePermission(c, permViewOrders); err != nil {
			return err
		}
		return b.handleOrderLookup(c, text)
	}
	if isBulkOrder(text) {
		if err := b.ensurePermission(c, permCreateOrder); err != nil {
			return err
		}
		return b.handleBulkOrder(c, text)
	}
	if strings.HasPrefix(text, "/") {
		return c.Send("Неизвестная команда.\n\n/start — показать список команд")
	}
	return c.Send("Отправьте batch-заявку или номер заказа. /start — список форматов.", tele.ModeMarkdown)
}

func (b *OrderBot) handleOrderLookup(c tele.Context, text string) error {
	numbers := splitNumbers(text)

	if len(numbers) == 1 {
		return b.showSingleOrder(c, numbers[0])
	}
	return b.showMultipleOrders(c, numbers)
}

func (b *OrderBot) showSingleOrder(c tele.Context, number string) error {
	order, err := b.orderSvc.GetOrderByNumber(context.Background(), number)
	if err != nil {
		return c.Send(fmt.Sprintf("Заказ %q не найден.", number))
	}
	return c.Send(formatOrderSummary(order), tele.ModeMarkdown)
}

func (b *OrderBot) showMultipleOrders(c tele.Context, numbers []string) error {
	var (
		found    []domain.Order
		notFound []string
	)

	for _, num := range numbers {
		order, err := b.orderSvc.GetOrderByNumber(context.Background(), num)
		if err != nil {
			notFound = append(notFound, num)
			continue
		}
		found = append(found, order)
	}

	if len(found) == 0 {
		return c.Send("Ни один из указанных заказов не найден.")
	}

	return c.Send(formatMultiOrderSummary(found, notFound), tele.ModeMarkdown)
}

func splitNumbers(text string) []string {
	var result []string
	for _, part := range strings.FieldsFunc(text, func(r rune) bool {
		return r == ' ' || r == '\n' || r == '\t' || r == ','
	}) {
		if strings.Contains(part, "_ORDER_") {
			result = append(result, part)
		}
	}
	return result
}

func (b *OrderBot) handleConfirm(c tele.Context) error {
	var location string
	var items []domain.OrderItem
	b.updateSession(c.Sender().ID, func(s *session) {
		location = s.location
		items = make([]domain.OrderItem, len(s.items))
		copy(items, s.items)
		s.location = ""
		s.items = nil
	})

	_ = c.Respond()
	if len(items) == 0 {
		return c.Send("Заявка пустая.")
	}

	order, err := b.orderSvc.CreateOrder(context.Background(), domain.CreateOrderInput{
		Items:    items,
		Location: location,
	})
	if err != nil {
		log.Printf("ERR create order uid=%d: %v", c.Sender().ID, err)
		return c.Send(fmt.Sprintf("Ошибка создания заказа: %v", err))
	}
	log.Printf("ORDER created %s uid=%d items=%d", order.Number, c.Sender().ID, len(items))

	return c.Send(formatOrderSummary(order), tele.ModeMarkdown)
}

func (b *OrderBot) handleCancelCallback(c tele.Context) error {
	b.clearSession(c.Sender().ID)
	_ = c.Respond()
	return c.Send("Заявка отменена.")
}

func (b *OrderBot) handleOrders(c tele.Context) error {
	orders, err := b.orderSvc.ListOrders(context.Background(), 10)
	if err != nil {
		return c.Send(fmt.Sprintf("Ошибка: %v", err))
	}
	if len(orders) == 0 {
		return c.Send("Заказов пока нет.")
	}

	var sb strings.Builder
	sb.WriteString("*Последние заказы:*\n\n")
	for _, o := range orders {
		sb.WriteString(fmt.Sprintf("📦 `%s` — %s\n",
			o.Number, o.CreatedAt.Local().Format("02.01.2006 15:04"),
		))
		for _, it := range o.Items {
			sb.WriteString(fmt.Sprintf("  • %s × %s\n", it.ProductName, formatQuantity(it.Quantity)))
		}
		sb.WriteString("\n")
	}
	sb.WriteString("_Отправьте номер заказа чтобы увидеть расчёт теста._")
	return c.Send(sb.String(), tele.ModeMarkdown)
}

func (b *OrderBot) handleAcceptOrder(c tele.Context) error {
	return c.Send("Команда принятия заказа принята. Детальная реализация операции будет добавлена отдельно.")
}

func (b *OrderBot) handleDeleteOrder(c tele.Context) error {
	return c.Send("Команда удаления заказа принята. Детальная реализация операции будет добавлена отдельно.")
}

func (b *OrderBot) handleCloseOrder(c tele.Context) error {
	return c.Send("Команда закрытия заказа принята. Детальная реализация операции будет добавлена отдельно.")
}

func (b *OrderBot) handleReports(c tele.Context) error {
	return c.Send("Раздел отчётов доступен для пекаря и администратора. Детальный отчёт будет добавлен отдельной командой.")
}

func (b *OrderBot) handleAddGroup(c tele.Context) error {
	return c.Send("Команда добавления группы принята. Управление группами доступно только администратору.")
}

func (b *OrderBot) handleAddUser(c tele.Context) error {
	return c.Send("Команда добавления пользователя принята. Управление пользователями доступно только администратору.")
}

func (b *OrderBot) ensurePermission(c tele.Context, permission string) error {
	user, err := b.authUserFromContext(c)
	if err != nil {
		return err
	}
	if !userHasPermission(user.Role, permission) {
		return c.Send(fmt.Sprintf("Доступ запрещён: недостаточно прав (%s).", permission))
	}
	return nil
}

func (b *OrderBot) handleBulkOrder(c tele.Context, text string) error {
	location, _, parsed := parseBulkOrder(text)

	if len(parsed) == 0 {
		return c.Send("Не удалось распознать позиции в сообщении.")
	}

	var items []domain.OrderItem
	var unknown []string

	for _, line := range parsed {
		dish, err := b.orderSvc.ResolveDishByName(context.Background(), line.Name)
		if err != nil {
			unknown = append(unknown, line.Name)
			continue
		}
		dish.Quantity = line.Quantity
		items = append(items, dish)
	}

	if len(items) == 0 {
		return c.Send("Ни один продукт не найден в каталоге.\n\nПроверьте правильность названий.")
	}

	b.updateSession(c.Sender().ID, func(s *session) {
		s.location = location
		s.items = items
	})

	var sb strings.Builder
	sb.WriteString("<b>Предпросмотр batch-заявки</b>")
	if location != "" {
		sb.WriteString(fmt.Sprintf("\n%s", html.EscapeString(location)))
	}
	sb.WriteString(fmt.Sprintf("\n\n<b>Распознано позиций: %d</b>\n", len(items)))
	for _, it := range items {
		sb.WriteString(fmt.Sprintf("• %s — %s шт.\n", html.EscapeString(it.ProductName), formatQuantity(it.Quantity)))
	}
	if len(unknown) > 0 {
		sb.WriteString(fmt.Sprintf("\n<b>Ошибки: не найдены в каталоге (%d)</b>\n", len(unknown)))
		for _, name := range unknown {
			sb.WriteString(fmt.Sprintf("— %s\n", html.EscapeString(name)))
		}
		sb.WriteString("\nБудут отправлены только распознанные позиции. Если нужно исправить, отправьте новый заказ целиком.")
	} else {
		sb.WriteString("\nОшибок нет.")
	}
	sb.WriteString("\n\n<blockquote expandable>")
	sb.WriteString("<b>Текст для копирования</b>\n")
	sb.WriteString(html.EscapeString(formatCopyableOrder(location, items, unknown)))
	sb.WriteString("</blockquote>")

	markup := &tele.ReplyMarkup{}
	markup.Inline(
		markup.Row(
			markup.Data("📤 Отправить", "confirm"),
			markup.Data("❌ Отмена", "cancel_cb"),
		),
	)
	return c.Send(sb.String(), tele.ModeHTML, markup)
}

func formatCopyableOrder(location string, items []domain.OrderItem, unknown []string) string {
	var sb strings.Builder
	if location != "" {
		sb.WriteString(location)
		sb.WriteString("\n")
	}
	if location != "" {
		sb.WriteString("\n")
	}
	for _, it := range items {
		sb.WriteString(fmt.Sprintf("%s %s\n", it.ProductName, formatQuantity(it.Quantity)))
	}
	if len(unknown) > 0 {
		sb.WriteString("\nНЕ НАЙДЕНЫ В КАТАЛОГЕ:\n")
		for _, name := range unknown {
			sb.WriteString(name)
			sb.WriteString("\n")
		}
	}
	return strings.TrimSpace(sb.String())
}

func formatMultiOrderSummary(orders []domain.Order, notFound []string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("📦 *Сводка по %d заказам*\n\n", len(orders)))

	for _, o := range orders {
		sb.WriteString(fmt.Sprintf("`%s` (%s)\n", o.Number, o.CreatedAt.Local().Format("02.01 15:04")))
		for _, it := range o.Items {
			sb.WriteString(fmt.Sprintf("  • %s × %s\n", it.ProductName, formatQuantity(it.Quantity)))
		}
	}

	if len(notFound) > 0 {
		sb.WriteString("\n⚠️ *Не найдены:*\n")
		for _, n := range notFound {
			sb.WriteString(fmt.Sprintf("  — `%s`\n", n))
		}
	}

	return sb.String()
}

func formatOrderSummary(order domain.Order) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("✅ *Заказ %s*\n", order.Number))
	sb.WriteString(fmt.Sprintf("🕐 %s\n\n", order.CreatedAt.Local().Format("02.01.2006 15:04")))
	sb.WriteString("*Состав заявки:*\n")
	for _, it := range order.Items {
		sb.WriteString(fmt.Sprintf("• %s — %s шт.\n", it.ProductName, formatQuantity(it.Quantity)))
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
