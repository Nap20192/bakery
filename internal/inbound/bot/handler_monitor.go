package bot

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	monitoringdomain "bakery/internal/domain/monitoring"
	orderdomain "bakery/internal/domain/order"
	applog "bakery/pkg/logger"

	tele "gopkg.in/telebot.v3"
)

func (b *OrderBot) handleMonitor(c tele.Context) error {
	ctx := requestContext(c)
	args := strings.Fields(c.Message().Payload)
	if len(args) != 1 && len(args) != 2 {
		return sendText(c, "Формат: /monitor order_number [code]")
	}

	ctx = applog.WithOrderNumber(ctx, args[0])
	order, err := b.orderSvc.GetOrderByNumber(ctx, args[0])
	if err != nil {
		slog.WarnContext(ctx, "order lookup failed", "error", err)
		return sendText(c, fmt.Sprintf("Заказ %s не найден.", args[0]))
	}

	if len(args) == 1 {
		return b.sendMonitorReports(ctx, c, order, defaultMonitorCodes)
	}
	return b.sendMonitorReports(ctx, c, order, []string{args[1]})
}

func (b *OrderBot) handleSync(c tele.Context) error {
	if b.syncSvc == nil {
		return sendText(c, "Sync service недоступен.")
	}

	if err := sendText(c, "Синхронизация с iiko запущена..."); err != nil {
		return err
	}

	start := time.Now()
	ctx, cancel := context.WithTimeout(requestContext(c), 15*time.Minute)
	defer cancel()

	if err := b.syncSvc.SyncOnce(ctx); err != nil {
		slog.ErrorContext(ctx, "manual iiko sync failed", "error", err)
		return sendText(c, "Синхронизация с iiko не выполнена. Подробности записаны в лог.")
	}

	return sendText(c, fmt.Sprintf("Синхронизация с iiko завершена за %s.", time.Since(start).Round(time.Second)))
}

func (b *OrderBot) sendMonitorReports(ctx context.Context, c tele.Context, order orderdomain.Order, codes []string) error {
	message, err := b.buildMonitorReports(ctx, order, codes)
	if err != nil {
		return err
	}
	return sendHTML(c, message)
}

func (b *OrderBot) buildMonitorReports(ctx context.Context, order orderdomain.Order, codes []string) (string, error) {
	var reports []monitoringdomain.IngredientReport
	for _, code := range codes {
		code = strings.TrimSpace(code)
		reportCtx := applog.WithProductCode(ctx, code)
		report, err := b.monitorSvc.GetIngredientsByCode(reportCtx, code, order)
		if err != nil {
			slog.WarnContext(reportCtx, "monitor report failed", "error", err)
			return "", fmt.Errorf("monitor code %s: %w", code, err)
		}
		reports = append(reports, report)
	}
	return responses.MonitorReports(order, reports), nil
}

func (b *OrderBot) sendBatchMonitorReports(ctx context.Context, c tele.Context, orderNumbers []string) error {
	orders := make([]orderdomain.Order, 0, len(orderNumbers))
	for _, number := range orderNumbers {
		number = strings.TrimSpace(number)
		if number == "" {
			continue
		}
		orderCtx := applog.WithOrderNumber(ctx, number)
		order, err := b.orderSvc.GetOrderByNumber(orderCtx, number)
		if err != nil {
			slog.WarnContext(orderCtx, "order lookup failed", "error", err)
			return sendText(c, fmt.Sprintf("Заказ %s не найден.", number))
		}
		orders = append(orders, order)
	}
	if len(orders) == 0 {
		return sendText(c, "Выберите заказы в списке /orders.")
	}

	report, err := b.monitorSvc.GetBatchIngredientsByCodes(ctx, defaultMonitorCodes, orders)
	if err != nil {
		slog.WarnContext(ctx, "batch monitor report failed", "error", err)
		return sendText(c, "Не удалось посчитать калькуляцию по выбранным заказам.")
	}

	return sendHTML(c, responses.BatchMonitorReports(report))
}
