package deps

import (
	"fmt"

	"bakery/internal/app"
	"bakery/internal/inbound/api"
	"bakery/internal/inbound/bot"
)

type AppDeps struct {
	AuthService       *app.AuthService
	RbacService       *app.RbacService
	DepartmentService *app.DepartmentService
	MonitorService    *app.MonitorService
	OrderService      *app.OrderService
	SyncService       *app.SyncService
	TechCardService   *app.TechCardService
	APIServer         *api.Server
	OrderBot          *bot.OrderBot
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

func WithRbacService() appOption {
	return func(deps *AppDeps) error {
		deps.RbacService = app.NewRbacService()
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

func WithDepartmentService(infra *InfraDeps) appOption {
	return func(deps *AppDeps) error {
		if infra == nil || infra.queries == nil {
			return fmt.Errorf("missing dependencies for DepartmentService")
		}
		deps.DepartmentService = app.NewDepartmentService(infra.queries)
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
		)
		return nil
	}
}

func WithOrderBot(infra *InfraDeps) appOption {
	return func(deps *AppDeps) error {
		if infra == nil || infra.config == nil || deps.OrderService == nil || deps.AuthService == nil || deps.RbacService == nil || deps.DepartmentService == nil || deps.MonitorService == nil || deps.SyncService == nil || deps.TechCardService == nil {
			return fmt.Errorf("missing dependencies for OrderBot")
		}
		if infra.config.Telegram.BotToken == "" {
			switch infra.config.Telegram.BotEnv {
			case "prod", "production":
				return fmt.Errorf("PROD_BOT_TOKEN не задан")
			default:
				return fmt.Errorf("TEST_BOT_TOKEN не задан")
			}
		}
		orderBot, err := bot.NewOrderBot(infra.config.Telegram.BotToken, deps.OrderService, deps.AuthService, deps.RbacService, deps.DepartmentService, deps.MonitorService, deps.SyncService, deps.TechCardService)
		if err != nil {
			return err
		}
		deps.OrderBot = orderBot
		return nil
	}
}

func WithAPIServer() appOption {
	return func(deps *AppDeps) error {
		if deps.OrderService == nil || deps.MonitorService == nil || deps.DepartmentService == nil {
			return fmt.Errorf("missing dependencies for APIServer")
		}
		deps.APIServer = api.NewServer(deps.OrderService, deps.MonitorService, deps.DepartmentService)
		return nil
	}
}
