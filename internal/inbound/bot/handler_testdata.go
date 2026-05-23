package bot

import (
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	orderdomain "bakery/internal/domain/order"
	"bakery/internal/pkg/enum"

	tele "gopkg.in/telebot.v3"
)

var (
	errInvalidTestOrderDay      = errors.New("invalid test order day")
	errTestOrderDateUnavailable = errors.New("test order date unavailable")
)

var testOrderItemsByShopCode = map[string][]orderdomain.OrderItem{
	"gagarina": {
		{Code: "15647", ProductName: "Сосиска в тесте", Quantity: 5},
		{Code: "15635", ProductName: "Пирожок с капустой", Quantity: 7},
		{Code: "19864", ProductName: "Булочка сахарная", Quantity: 4},
	},
	"saryarka": {
		{Code: "20495", ProductName: "Пирожок с картошкой", Quantity: 6},
		{Code: "15648", ProductName: "Сосиска с сыром в тесте", Quantity: 5},
		{Code: "15660", ProductName: "Пицца домашняя", Quantity: 3},
	},
	"sholokhova": {
		{Code: "15658", ProductName: "Учпучмак", Quantity: 6},
		{Code: "20078", ProductName: "Вишневый пирог", Quantity: 2},
		{Code: "15629", ProductName: "Ватрушка с творогом", Quantity: 4},
	},
}

func (b *OrderBot) handleTestOrders(c tele.Context) error {
	ctx := requestContext(c)
	args := strings.Fields(c.Message().Payload)
	if len(args) != 1 {
		return sendText(c, "Формат: /test 25\nУкажите только число месяца.")
	}

	fulfillmentDate, err := parseTestOrderDate(time.Now(), args[0])
	if err != nil {
		return sendText(c, testOrderDateErrorMessage(err))
	}

	shops, err := b.departmentSvc.ListByType(ctx, enum.DepartmentTypeShop)
	if err != nil {
		slog.ErrorContext(ctx, "list shops for test orders failed", "error", err)
		return sendText(c, "Не удалось получить магазины для тестовых заказов.")
	}
	workshop, err := b.departmentSvc.GetByCode(ctx, workshopDepartmentCode)
	if err != nil {
		slog.ErrorContext(ctx, "get workshop for test orders failed", "error", err)
		return sendText(c, "Не удалось найти цех для тестовых заказов.")
	}

	sender := c.Sender()
	var senderID int64
	if sender != nil {
		senderID = sender.ID
	}

	var created []string
	var failed []string
	for _, shop := range shops {
		items := testItemsForShop(shop.Code)
		fromID := shop.ID
		toID := workshop.ID
		order, err := b.orderSvc.CreateOrder(ctx, orderdomain.CreateOrderInput{
			Items:             items,
			FromDepartmentID:  &fromID,
			ToDepartmentID:    &toID,
			CreatedByUsername: "test",
			FulfillmentDate:   fulfillmentDate,
		})
		if err != nil {
			slog.ErrorContext(ctx, "create test order failed", "shop", shop.Code, "error", err)
			failed = append(failed, fmt.Sprintf("%s: ошибка создания", shop.Name))
			continue
		}

		created = append(created, fmt.Sprintf("%s: %s", shop.Name, order.Number))
		if err := b.notifyWorkshop(ctx, senderID, responses.OrderSummary(order, shop.Name, workshop.Name)); err != nil {
			slog.WarnContext(ctx, "notify workshop about test order failed", "order_number", order.Number, "error", err)
		}
	}

	return sendText(c, testOrdersResultMessage(fulfillmentDate, created, failed), b.actionMarkup(c))
}

func parseTestOrderDate(now time.Time, raw string) (time.Time, error) {
	day, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || day < 1 || day > 31 {
		return time.Time{}, errInvalidTestOrderDay
	}

	location := now.Location()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location)
	for offset := 0; offset < 24; offset++ {
		monthStart := time.Date(today.Year(), today.Month()+time.Month(offset), 1, 0, 0, 0, 0, location)
		candidate := time.Date(monthStart.Year(), monthStart.Month(), day, 0, 0, 0, 0, location)
		if candidate.Month() != monthStart.Month() {
			continue
		}
		if candidate.Before(today) {
			continue
		}
		return candidate, nil
	}
	return time.Time{}, fmt.Errorf("%w: day %d", errTestOrderDateUnavailable, day)
}

func testOrderDateErrorMessage(err error) string {
	switch {
	case errors.Is(err, errInvalidTestOrderDay):
		return "Укажите число месяца от 1 до 31."
	case errors.Is(err, errTestOrderDateUnavailable):
		return "Не удалось подобрать дату с таким числом."
	default:
		return "Не удалось разобрать дату для тестовых заказов."
	}
}

func testItemsForShop(code string) []orderdomain.OrderItem {
	items := testOrderItemsByShopCode[strings.ToLower(strings.TrimSpace(code))]
	if len(items) == 0 {
		items = testOrderItemsByShopCode["gagarina"]
	}
	result := make([]orderdomain.OrderItem, len(items))
	copy(result, items)
	return result
}

func testOrdersResultMessage(date time.Time, created []string, failed []string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Тестовые заказы на %s\n\n", date.Format("02.01.2006")))
	if len(created) > 0 {
		sb.WriteString("Созданы:\n")
		for _, row := range created {
			sb.WriteString(row)
			sb.WriteString("\n")
		}
	}
	if len(failed) > 0 {
		if len(created) > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString("Не созданы:\n")
		for _, row := range failed {
			sb.WriteString(row)
			sb.WriteString("\n")
		}
	}
	if len(created) == 0 && len(failed) == 0 {
		sb.WriteString("Магазины не найдены.")
	}
	return strings.TrimSpace(sb.String())
}
