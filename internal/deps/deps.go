package deps

import (
	"fmt"

	"bakery/internal/app"
	"bakery/internal/bot"
)

type AppDeps struct {
	AuthService  *app.AuthService
	OrderService *app.OrderService
	SyncService  *app.SyncService
	OrderBot     *bot.OrderBot
	AdminBot     *bot.AdminBot
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
		if infra == nil || infra.IikoQueries == nil {
			return fmt.Errorf("missing dependencies for AuthService")
		}
		deps.AuthService = app.NewAuthService(infra.IikoQueries)
		return nil
	}
}

func WithOrderService(infra *InfraDeps) appOption {
	return func(deps *AppDeps) error {
		if infra == nil || infra.ProductRepo == nil {
			return fmt.Errorf("missing dependencies for OrderService")
		}
		deps.OrderService = app.NewOrderService(infra.ProductRepo)
		return nil
	}
}

func WithSyncService(infra *InfraDeps) appOption {
	return func(deps *AppDeps) error {
		if infra == nil || infra.Config == nil || infra.DB == nil || infra.IikoClient == nil || infra.IikoQueries == nil {
			return fmt.Errorf("missing dependencies for SyncService")
		}
		deps.SyncService = app.NewSyncService(
			infra.IikoClient,
			infra.DB,
			infra.IikoQueries,
			infra.Config.Sync.Interval,
			infra.Config.Sync.DateFrom,
			infra.Config.Sync.DateTo,
		)
		return nil
	}
}

func WithOrderBot(infra *InfraDeps) appOption {
	return func(deps *AppDeps) error {
		if infra == nil || infra.Config == nil || infra.ProductRepo == nil || infra.OrderRepo == nil || infra.DoughRepo == nil || deps.OrderService == nil {
			return fmt.Errorf("missing dependencies for OrderBot")
		}
		if infra.Config.Telegram.OrderBotToken == "" {
			return fmt.Errorf("ORDER_BOT_TOKEN не задан")
		}
		orderBot, err := bot.NewOrderBot(infra.Config.Telegram.OrderBotToken, deps.OrderService, infra.OrderRepo, infra.ProductRepo, infra.DoughRepo)
		if err != nil {
			return err
		}
		deps.OrderBot = orderBot
		return nil
	}
}

func WithAdminBot(infra *InfraDeps) appOption {
	return func(deps *AppDeps) error {
		if infra == nil || infra.Config == nil || infra.OrderRepo == nil || infra.DoughRepo == nil || deps.OrderService == nil {
			return fmt.Errorf("missing dependencies for AdminBot")
		}
		if infra.Config.Telegram.AdminBotToken == "" {
			return fmt.Errorf("ADMIN_BOT_TOKEN не задан")
		}
		adminBot, err := bot.NewAdminBot(infra.Config.Telegram.AdminBotToken, deps.OrderService, infra.OrderRepo, infra.DoughRepo)
		if err != nil {
			return err
		}
		deps.AdminBot = adminBot
		return nil
	}
}
