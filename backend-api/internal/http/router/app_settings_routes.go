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

func AppSettingsRoutes(a *auth.Service, rsvc *rbac.Service, ts *tenant.Service, as *audit.Service, trustProxy bool) DomainRegistrar {
	return func(r chi.Router) {
		h := handler.NewAppSettingsHandler(ts, as, trustProxy)
		r.Route("/app/settings", func(tr chi.Router) {
			tr.Use(middleware.BodyLimit(middleware.DefaultBodyLimit))
			tr.Use(middleware.RequireAuth(a))
			tr.Use(middleware.RequirePasswordChanged())
			tr.With(middleware.RequirePermission(rsvc, "app.settings.read")).Get("/", h.Get)
			tr.With(middleware.RequirePermission(rsvc, "app.settings.update")).Patch("/", h.Update)
		})
	}
}
