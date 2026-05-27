package router

import (
	"github.com/go-chi/chi/v5"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/auth"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/http/handler"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/middleware"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/rbac"
)

func RBACRoutes(a *auth.Service, s *rbac.Service) DomainRegistrar {
	return func(r chi.Router) {
		h := handler.NewMeHandler(a, s)
		r.Group(func(pr chi.Router) {
			pr.Use(middleware.RequireAuth(a))
			pr.Use(middleware.RequirePasswordChanged(a))
			pr.Get("/me/tenants", h.Tenants)
			pr.With(middleware.RequireTenantAccess(s)).Get("/me/permissions", h.Permissions)
		})
	}
}
