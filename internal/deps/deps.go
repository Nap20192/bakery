package deps

import (
	"context"
	"fmt"

	"bakery/internal/inbound/api"
	"bakery/internal/inbound/bot"
	adminuc "bakery/internal/services/admin/usecase/admin"
	authrepo "bakery/internal/services/auth/infra/repo"
	authuc "bakery/internal/services/auth/usecase/auth"
	departmentrepo "bakery/internal/services/department/infra/repo"
	departmentuc "bakery/internal/services/department/usecase/department"
	monitorrepo "bakery/internal/services/monitor/infra/repo"
	monitoruc "bakery/internal/services/monitor/usecase/monitor"
	orderrepo "bakery/internal/services/order/infra/repo"
	orderuc "bakery/internal/services/order/usecase/order"
	syncrepo "bakery/internal/services/sync/infra/repo"
	syncuc "bakery/internal/services/sync/usecase/sync"
	techcardrepo "bakery/internal/services/techcard/infra/repo"
	techcarduc "bakery/internal/services/techcard/usecase/techcard"
)

type AppDeps struct {
	AuthService       authuc.UseCase
	AdminService      adminuc.UseCase
	RbacService       *authuc.RBAC
	DepartmentService departmentuc.UseCase
	MonitorService    monitoruc.UseCase
	OrderService      orderuc.UseCase
	SyncService       syncuc.UseCase
	TechCardService   techcarduc.UseCase
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
		deps.AuthService = authuc.NewService(authrepo.New(infra.queries))
		return nil
	}
}

// adminDepartments adapts the department service to the admin service's
// Departments port.
type adminDepartments struct {
	svc departmentuc.UseCase
}

func (a adminDepartments) GetByCode(ctx context.Context, code string) (adminuc.Department, error) {
	d, err := a.svc.GetByCode(ctx, code)
	if err != nil {
		return adminuc.Department{}, err
	}
	return adminuc.Department{ID: d.ID, Code: d.Code, Name: d.Name, Type: d.Type}, nil
}

func (a adminDepartments) ListAll(ctx context.Context) ([]adminuc.Department, error) {
	rows, err := a.svc.ListByType(ctx, "")
	if err != nil {
		return nil, err
	}
	out := make([]adminuc.Department, 0, len(rows))
	for _, d := range rows {
		out = append(out, adminuc.Department{ID: d.ID, Code: d.Code, Name: d.Name, Type: d.Type})
	}
	return out, nil
}

func WithAdminService() appOption {
	return func(deps *AppDeps) error {
		if deps.AuthService == nil || deps.DepartmentService == nil {
			return fmt.Errorf("missing dependencies for AdminService")
		}
		deps.AdminService = adminuc.NewService(deps.AuthService, adminDepartments{svc: deps.DepartmentService})
		return nil
	}
}

func WithRbacService() appOption {
	return func(deps *AppDeps) error {
		deps.RbacService = authuc.NewRBAC()
		return nil
	}
}

func WithOrderService(infra *InfraDeps) appOption {
	return func(deps *AppDeps) error {
		if infra == nil || infra.queries == nil || infra.DB == nil || infra.eventPublisher == nil {
			return fmt.Errorf("missing dependencies for OrderService")
		}
		deps.OrderService = orderuc.NewService(orderrepo.New(infra.queries, infra.DB), infra.eventPublisher)
		return nil
	}
}

func WithDepartmentService(infra *InfraDeps) appOption {
	return func(deps *AppDeps) error {
		if infra == nil || infra.queries == nil {
			return fmt.Errorf("missing dependencies for DepartmentService")
		}
		deps.DepartmentService = departmentuc.NewService(departmentrepo.New(infra.queries))
		return nil
	}
}

func WithMonitorService(infra *InfraDeps) appOption {
	return func(deps *AppDeps) error {
		if infra == nil || infra.queries == nil {
			return fmt.Errorf("missing dependencies for MonitorService")
		}
		deps.MonitorService = monitoruc.NewService(monitorrepo.New(infra.queries))
		return nil
	}
}

func WithTechCardService(infra *InfraDeps) appOption {
	return func(deps *AppDeps) error {
		if infra == nil || infra.queries == nil {
			return fmt.Errorf("missing dependencies for TechCardService")
		}
		deps.TechCardService = techcarduc.NewService(techcardrepo.New(infra.queries))
		return nil
	}
}

func WithSyncService(infra *InfraDeps) appOption {
	return func(deps *AppDeps) error {
		if infra == nil || infra.config == nil || infra.DB == nil || infra.iikoClient == nil || infra.queries == nil {
			return fmt.Errorf("missing dependencies for SyncService")
		}
		deps.SyncService = syncuc.NewService(
			infra.iikoClient,
			syncrepo.New(infra.DB, infra.queries),
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
		orderBot, err := bot.NewOrderBot(
			infra.config.Telegram.BotToken,
			deps.OrderService,
			deps.AuthService,
			deps.RbacService,
			deps.DepartmentService,
			deps.MonitorService,
			deps.SyncService,
			deps.TechCardService,
			infra.eventConsumer,
			infra.config.Telegram.MiniAppURL,
			infra.config.Telegram.WorkshopChatID,
		)
		if err != nil {
			return err
		}
		deps.OrderBot = orderBot
		return nil
	}
}

func WithAPIServerConfig(infra *InfraDeps) appOption {
	return func(deps *AppDeps) error {
		if infra == nil || infra.config == nil {
			return fmt.Errorf("missing dependencies for APIServer config")
		}
		if deps.OrderService == nil || deps.MonitorService == nil || deps.DepartmentService == nil || deps.AuthService == nil || deps.AdminService == nil {
			return fmt.Errorf("missing dependencies for APIServer")
		}
		deps.APIServer = api.NewServer(deps.OrderService, deps.MonitorService, deps.DepartmentService, deps.AuthService, deps.AdminService, api.ServerConfig{
			Addr:           infra.config.Server.Addr(),
			AllowedOrigins: infra.config.Server.AllowedOrigins,
			BotToken:       infra.config.Telegram.BotToken,
		})
		return nil
	}
}
