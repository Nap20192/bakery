package deps

import (
	"fmt"

	"bakery/internal/app"
	"bakery/internal/bot"
)

type AppDeps struct {
	AuthService     *app.AuthService
	MonitorService  *app.MonitorService
	OrderService    *app.OrderService
	SyncService     *app.SyncService
	TechCardService *app.TechCardService
	OrderBot        *bot.OrderBot
}

type appOption func(*AppDeps) error

func NewAppDeps(opts ...appOption) (*AppDeps, error) {
	deps := &AppDeps{}
	for _, opt := range opts {
		if err := opt(deps); err != nil {
			return nil, err
		}
	}
	return deps, nil
}

func WithAuthService(infra *InfraDeps) appOption {
	return func(deps *AppDeps) error {
		if infra == nil || infra.queries == nil {
			return fmt.Errorf("missing dependencies for AuthService")
		}
		deps.AuthService = app.NewAuthService(infra.queries)
		return nil
	}
}

func WithOrderService(infra *InfraDeps) appOption {
	return func(deps *AppDeps) error {
		if infra == nil || infra.queries == nil {
			return fmt.Errorf("missing dependencies for OrderService")
		}
		deps.OrderService = app.NewOrderService(infra.queries)
		return nil
	}
}

func WithMonitorService(infra *InfraDeps) appOption {
	return func(deps *AppDeps) error {
		if infra == nil || infra.queries == nil {
			return fmt.Errorf("missing dependencies for MonitorService")
		}
		deps.MonitorService = app.NewMonitorService(infra.queries)
		return nil
	}
}

func WithTechCardService(infra *InfraDeps) appOption {
	return func(deps *AppDeps) error {
		if infra == nil || infra.queries == nil {
			return fmt.Errorf("missing dependencies for TechCardService")
		}
		deps.TechCardService = app.NewTechCardService(infra.queries)
		return nil
	}
}

func WithSyncService(infra *InfraDeps) appOption {
	return func(deps *AppDeps) error {
		if infra == nil || infra.config == nil || infra.DB == nil || infra.iikoClient == nil || infra.queries == nil {
			return fmt.Errorf("missing dependencies for SyncService")
		}
		deps.SyncService = app.NewSyncService(
			infra.iikoClient,
			infra.DB,
			infra.queries,
			infra.config.Sync.Interval,
			infra.config.Sync.DateFrom,
			infra.config.Sync.DateTo,
		)
		return nil
	}
}

func WithOrderBot(infra *InfraDeps) appOption {
	return func(deps *AppDeps) error {
		if infra == nil || infra.config == nil || deps.OrderService == nil || deps.AuthService == nil || deps.MonitorService == nil || deps.TechCardService == nil {
			return fmt.Errorf("missing dependencies for OrderBot")
		}
		if infra.config.Telegram.OrderBotToken == "" {
			return fmt.Errorf("ORDER_BOT_TOKEN не задан")
		}
		orderBot, err := bot.NewOrderBot(infra.config.Telegram.OrderBotToken, deps.OrderService, deps.AuthService, deps.MonitorService, deps.TechCardService)
		if err != nil {
			return err
		}
		deps.OrderBot = orderBot
		return nil
	}
}
