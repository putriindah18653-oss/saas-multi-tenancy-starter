package router

import (
	"github.com/go-chi/chi/v5"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/audit"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/auth"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/http/handler"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/middleware"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/rbac"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/tenant"
)

func AppTenantRoutes(a *auth.Service, rsvc *rbac.Service, ts *tenant.Service, as *audit.Service, trustProxy bool) DomainRegistrar {
	return func(r chi.Router) {
		h := handler.NewTenantHandler(ts, as, trustProxy)
		r.Route("/app/tenants", func(tr chi.Router) {
			tr.Use(middleware.RequireAuth(a))
			tr.Use(middleware.RequirePasswordChanged(a))
			tr.With(middleware.RequirePermission(rsvc, "app.tenants.read")).Get("/", h.List)
			tr.With(middleware.RequirePermission(rsvc, "app.tenants.create")).Post("/", h.Create)
			tr.With(middleware.RequirePermission(rsvc, "app.tenants.read")).Get("/{id}", h.Get)
			tr.With(middleware.RequirePermission(rsvc, "app.tenants.update")).Patch("/{id}", h.Update)
			tr.With(middleware.RequirePermission(rsvc, "app.tenants.delete")).Delete("/{id}", h.Delete)
		})
	}
}
