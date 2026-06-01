package router

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/audit"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/auth"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/http/handler"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/middleware"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/rbac"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/user"
	"github.com/redis/go-redis/v9"
)

func TenantUserRoutes(a *auth.Service, rsvc *rbac.Service, us *user.Service, as *audit.Service, redisClient *redis.Client, trustProxy bool, proxySecret string) DomainRegistrar {
	return func(r chi.Router) {
		h := handler.NewTenantUserHandler(us, as, trustProxy)
		r.Route("/tenant", func(tr chi.Router) {
			tr.Use(middleware.BodyLimit(middleware.DefaultBodyLimit))
			tr.Use(middleware.RequireAuth(a))
			tr.Use(middleware.RequirePasswordChanged())
			tr.Use(middleware.RequireTenantAccess(rsvc, proxySecret))
			tr.With(middleware.RequirePermission(rsvc, "tenant.dashboard.read")).Get("/dashboard", h.Dashboard)
			tr.Get("/me", h.Me)
			tr.With(middleware.RequirePermission(rsvc, "tenant.users.read")).Get("/users", h.List)
			tr.With(tenantUserWriteLimit(redisClient, trustProxy, "tenant:users:invite", 10), middleware.RequirePermission(rsvc, "tenant.users.invite")).Post("/users/invite", h.Invite)
			tr.With(tenantUserWriteLimit(redisClient, trustProxy, "tenant:users:role", 20), middleware.RequirePermission(rsvc, "tenant.users.update")).Patch("/users/{id}/role", h.ChangeRole)
			tr.With(tenantUserWriteLimit(redisClient, trustProxy, "tenant:users:remove", 10), middleware.RequirePermission(rsvc, "tenant.users.remove")).Delete("/users/{id}", h.Remove)
		})
	}
}

func tenantUserWriteLimit(redisClient *redis.Client, trustProxy bool, name string, limit int64) func(http.Handler) http.Handler {
	return middleware.RateLimitRules(redisClient, trustProxy,
		middleware.RateLimitRule{Name: name, Scope: middleware.RateLimitByTenant, Limit: limit, Window: time.Minute},
		middleware.RateLimitRule{Name: name, Scope: middleware.RateLimitByUser, Limit: limit, Window: time.Minute},
	)
}
