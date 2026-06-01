package router

import (
	"github.com/go-chi/chi/v5"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/audit"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/auth"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/http/handler"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/middleware"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/rbac"
)

func AuditRoutes(a *auth.Service, rsvc *rbac.Service, as *audit.Service) DomainRegistrar {
	return func(r chi.Router) {
		h := handler.NewAuditHandler(as)
		r.With(middleware.RequireAuth(a), middleware.RequirePasswordChanged(), middleware.RequirePermission(rsvc, "app.audit.read")).Get("/app/audit", h.AppList)
		r.With(middleware.RequireAuth(a), middleware.RequirePasswordChanged(), middleware.RequireTenantAccess(rsvc), middleware.RequirePermission(rsvc, "tenant.audit.read")).Get("/tenant/audit", h.TenantList)
	}
}
