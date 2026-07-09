package deps

import (
	"fmt"

	"bakery/internal/inbound/api"
	adminapp "bakery/internal/services/admin/app"
	adminuc "bakery/internal/services/admin/usecase/admin"
	authapp "bakery/internal/services/auth/app"
	authuc "bakery/internal/services/auth/usecase/auth"
	departmentapp "bakery/internal/services/department/app"
	departmentuc "bakery/internal/services/department/usecase/department"
	monitorapp "bakery/internal/services/monitor/app"
	monitoruc "bakery/internal/services/monitor/usecase/monitor"
	orderapp "bakery/internal/services/order/app"
	orderoutbox "bakery/internal/services/order/infra/outbox"
	orderuc "bakery/internal/services/order/usecase/order"
	syncapp "bakery/internal/services/sync/app"
	syncuc "bakery/internal/services/sync/usecase/sync"
	techcardapp "bakery/internal/services/techcard/app"
	techcarduc "bakery/internal/services/techcard/usecase/techcard"
)

type AppDeps struct {
	AuthService       authuc.UseCase
	AdminService      adminuc.UseCase
	RbacService       *authuc.RBAC
	DepartmentService departmentuc.UseCase
	MonitorService    monitoruc.UseCase
	OrderService      orderuc.UseCase
	OrderOutboxRelay  *orderoutbox.Relay
	SyncService       syncuc.UseCase
	TechCardService   techcarduc.UseCase
	APIServer         *api.Server
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
		deps.AuthService = authapp.New(infra.queries)
		return nil
	}
}

func WithAdminService() appOption {
	return func(deps *AppDeps) error {
		if deps.AuthService == nil || deps.DepartmentService == nil {
			return fmt.Errorf("missing dependencies for AdminService")
		}
		deps.AdminService = adminapp.New(deps.AuthService, deps.DepartmentService)
		return nil
	}
}

func WithRbacService() appOption {
	return func(deps *AppDeps) error {
		deps.RbacService = authapp.NewRBAC()
		return nil
	}
}

func WithOrderService(infra *InfraDeps) appOption {
	return func(deps *AppDeps) error {
		if infra == nil || infra.queries == nil || infra.DB == nil {
			return fmt.Errorf("missing dependencies for OrderService")
		}
		deps.OrderService = orderapp.New(infra.queries, infra.DB)
		return nil
	}
}

func WithOrderOutboxRelay(infra *InfraDeps) appOption {
	return func(deps *AppDeps) error {
		if infra == nil || infra.queries == nil || infra.DB == nil || infra.eventPublisher == nil {
			return fmt.Errorf("missing dependencies for OrderOutboxRelay")
		}
		deps.OrderOutboxRelay = orderapp.NewOutboxRelay(infra.queries, infra.DB, infra.eventPublisher, infra.config.Outbox.Interval)
		return nil
	}
}

func WithDepartmentService(infra *InfraDeps) appOption {
	return func(deps *AppDeps) error {
		if infra == nil || infra.queries == nil {
			return fmt.Errorf("missing dependencies for DepartmentService")
		}
		deps.DepartmentService = departmentapp.New(infra.queries)
		return nil
	}
}

func WithMonitorService(infra *InfraDeps) appOption {
	return func(deps *AppDeps) error {
		if infra == nil || infra.queries == nil {
			return fmt.Errorf("missing dependencies for MonitorService")
		}
		deps.MonitorService = monitorapp.New(infra.queries)
		return nil
	}
}

func WithTechCardService(infra *InfraDeps) appOption {
	return func(deps *AppDeps) error {
		if infra == nil || infra.queries == nil {
			return fmt.Errorf("missing dependencies for TechCardService")
		}
		deps.TechCardService = techcardapp.New(infra.queries)
		return nil
	}
}

func WithSyncService(infra *InfraDeps) appOption {
	return func(deps *AppDeps) error {
		if infra == nil || infra.config == nil || infra.DB == nil || infra.iikoClient == nil || infra.queries == nil {
			return fmt.Errorf("missing dependencies for SyncService")
		}
		deps.SyncService = syncapp.New(infra.iikoClient, infra.DB, infra.queries, infra.config.Sync.Interval)
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
