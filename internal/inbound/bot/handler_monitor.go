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
		return c.Send("Формат: /monitor order_number [code]")
	}

	ctx = applog.WithOrderNumber(ctx, args[0])
	order, err := b.orderSvc.GetOrderByNumber(ctx, args[0])
	if err != nil {
		slog.WarnContext(ctx, "order lookup failed", "error", err)
		return c.Send(fmt.Sprintf("Заказ %s не найден.", args[0]))
	}

	if len(args) == 1 {
		return b.sendMonitorReports(ctx, c, order, defaultMonitorCodes)
	}
	return b.sendMonitorReports(ctx, c, order, []string{args[1]})
}

func (b *OrderBot) handleSync(c tele.Context) error {
	if b.syncSvc == nil {
		return c.Send("Sync service недоступен.")
	}

	if err := c.Send("Синхронизация с iiko запущена..."); err != nil {
		return err
	}

	start := time.Now()
	ctx, cancel := context.WithTimeout(requestContext(c), 15*time.Minute)
	defer cancel()

	if err := b.syncSvc.SyncOnce(ctx); err != nil {
		slog.ErrorContext(ctx, "manual iiko sync failed", "error", err)
		return c.Send("Синхронизация с iiko не выполнена. Подробности записаны в лог.")
	}

	return c.Send(fmt.Sprintf("Синхронизация с iiko завершена за %s.", time.Since(start).Round(time.Second)))
}

func (b *OrderBot) sendMonitorReports(ctx context.Context, c tele.Context, order orderdomain.Order, codes []string) error {
	message, err := b.buildMonitorReports(ctx, order, codes)
	if err != nil {
		return err
	}
	return c.Send(message, tele.ModeHTML)
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
