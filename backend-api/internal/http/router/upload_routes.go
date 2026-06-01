package router

import (
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/auth"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/http/handler"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/middleware"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/rbac"
	"github.com/redis/go-redis/v9"
)

func UploadRoutes(a *auth.Service, rsvc *rbac.Service, redisClient *redis.Client, trustProxy bool) DomainRegistrar {
	return func(r chi.Router) {
		h := handler.NewUploadHandler("storage/uploads")
		r.With(middleware.RequireAuth(a)).Get("/uploads/*", h.Serve)
		r.Route("/app/uploads", func(ur chi.Router) {
			ur.Use(middleware.RequireAuth(a))
			ur.Use(middleware.RequirePasswordChanged())
			ur.Use(middleware.RateLimitRules(redisClient, trustProxy,
				middleware.RateLimitRule{Name: "upload:app_logo", Scope: middleware.RateLimitByIP, Limit: 10, Window: time.Minute},
				middleware.RateLimitRule{Name: "upload:app_logo", Scope: middleware.RateLimitByUser, Limit: 10, Window: time.Minute},
			))
			ur.With(middleware.RequirePermission(rsvc, "app.settings.update")).Post("/logo", h.AppLogo)
		})
		r.Route("/me/uploads", func(ur chi.Router) {
			ur.Use(middleware.RequireAuth(a))
			ur.Use(middleware.RequirePasswordChanged())
			ur.Use(middleware.RateLimitRules(redisClient, trustProxy,
				middleware.RateLimitRule{Name: "upload:avatar", Scope: middleware.RateLimitByIP, Limit: 10, Window: time.Minute},
				middleware.RateLimitRule{Name: "upload:avatar", Scope: middleware.RateLimitByUser, Limit: 10, Window: time.Minute},
			))
			ur.Post("/avatar", h.Avatar)
		})
	}
}
