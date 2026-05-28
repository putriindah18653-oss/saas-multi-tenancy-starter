package router

import (
	"net/http"
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
		r.With(middleware.RequireAuth(a)).Get("/uploads/*", func(w http.ResponseWriter, req *http.Request) {
			http.StripPrefix("/api/v1/uploads/", http.FileServer(http.Dir("storage/uploads"))).ServeHTTP(w, req)
		})
		r.Route("/app/uploads", func(ur chi.Router) {
			ur.Use(middleware.RequireAuth(a))
			ur.Use(middleware.RequirePasswordChanged(a))
			ur.Use(middleware.RateLimit(redisClient, "upload", 10, time.Minute, trustProxy))
			ur.With(middleware.RequirePermission(rsvc, "app.settings.update")).Post("/logo", h.AppLogo)
		})
		r.Route("/me/uploads", func(ur chi.Router) {
			ur.Use(middleware.RequireAuth(a))
			ur.Use(middleware.RequirePasswordChanged(a))
			ur.Use(middleware.RateLimit(redisClient, "upload", 10, time.Minute, trustProxy))
			ur.Post("/avatar", h.Avatar)
		})
	}
}
