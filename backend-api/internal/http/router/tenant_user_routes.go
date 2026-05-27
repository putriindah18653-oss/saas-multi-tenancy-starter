package router

import (
	"github.com/go-chi/chi/v5"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/auth"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/http/handler"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/middleware"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/rbac"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/user"
)

func TenantUserRoutes(a *auth.Service, rsvc *rbac.Service, us *user.Service) DomainRegistrar {
	return func(r chi.Router) {
		h := handler.NewTenantUserHandler(us)
		r.Route("/tenant", func(tr chi.Router) {
			tr.Use(middleware.RequireAuth(a))
			tr.Use(middleware.RequireTenantAccess(rsvc))
			tr.With(middleware.RequirePermission(rsvc, "tenant.dashboard.read")).Get("/dashboard", h.Dashboard)
			tr.Get("/me", h.Me)
			tr.With(middleware.RequirePermission(rsvc, "tenant.users.read")).Get("/users", h.List)
			tr.With(middleware.RequireTenantRole("owner-tenant"), middleware.RequirePermission(rsvc, "tenant.users.invite")).Post("/users/invite", h.Invite)
			tr.With(middleware.RequirePermission(rsvc, "tenant.users.update")).Patch("/users/{id}/role", h.ChangeRole)
			tr.With(middleware.RequirePermission(rsvc, "tenant.users.remove")).Delete("/users/{id}", h.Remove)
		})
	}
}
