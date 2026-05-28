package router

import (
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/audit"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/auth"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/http/handler"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/middleware"
	"github.com/redis/go-redis/v9"
)

func AuthRoutes(s *auth.Service, a *audit.Service, redisClient *redis.Client, trustProxy bool) DomainRegistrar {
	return func(r chi.Router) {
		h := handler.NewAuthHandler(s, a, trustProxy)
		r.With(middleware.RateLimit(redisClient, "auth:login", 10, time.Minute, trustProxy)).Post("/auth/login", h.Login)
		r.With(middleware.RateLimit(redisClient, "auth:refresh", 30, time.Minute, trustProxy)).Post("/auth/refresh", h.Refresh)
		r.Group(func(pr chi.Router) {
			pr.Use(middleware.RequireAuth(s))
			pr.Post("/auth/logout", h.Logout)
			pr.Get("/me", h.Me)
			pr.Patch("/me/profile", h.UpdateProfile)
			pr.With(middleware.RateLimit(redisClient, "auth:password", 5, time.Minute, trustProxy)).Post("/me/password", h.ChangePassword)
			pr.With(middleware.RequirePasswordChanged(s)).Get("/me/sessions", h.Sessions)
			pr.With(middleware.RequirePasswordChanged(s)).Delete("/me/sessions/{id}", h.RevokeSession)
		})
	}
}
