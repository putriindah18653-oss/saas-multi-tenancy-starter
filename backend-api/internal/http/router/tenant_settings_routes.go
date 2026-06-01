package router

import (
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/audit"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/auth"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/http/handler"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/middleware"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/rbac"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/tenant"
	"github.com/redis/go-redis/v9"
)

func TenantSettingsRoutes(a *auth.Service, rsvc *rbac.Service, ts *tenant.Service, as *audit.Service, redisClient *redis.Client, trustProxy bool) DomainRegistrar {
	return func(r chi.Router) {
		h := handler.NewTenantSettingsHandler(ts, as, trustProxy)
		r.Route("/tenant/settings", func(tr chi.Router) {
			tr.Use(middleware.RequireAuth(a))
			tr.Use(middleware.RequirePasswordChanged())
			tr.Use(middleware.RequireTenantAccess(rsvc))
			tr.With(middleware.RequirePermission(rsvc, "tenant.settings.read")).Get("/", h.Get)
			tr.With(middleware.RateLimitRules(redisClient, trustProxy,
				middleware.RateLimitRule{Name: "tenant:settings:update", Scope: middleware.RateLimitByTenant, Limit: 20, Window: time.Minute},
				middleware.RateLimitRule{Name: "tenant:settings:update", Scope: middleware.RateLimitByUser, Limit: 20, Window: time.Minute},
			), middleware.RequirePermission(rsvc, "tenant.settings.update")).Patch("/", h.Update)
		})
	}
}
