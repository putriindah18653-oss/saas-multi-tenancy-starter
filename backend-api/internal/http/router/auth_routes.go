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

func AuthRoutes(s *auth.Service, a *audit.Service, redisClient *redis.Client, trustProxy, secureCookie bool) DomainRegistrar {
	return func(r chi.Router) {
		h := handler.NewAuthHandler(s, a, trustProxy, secureCookie)
		r.With(middleware.RateLimit(redisClient, "auth:login", 10, time.Minute, trustProxy)).Post("/auth/login", h.Login)
		r.With(middleware.RateLimit(redisClient, "auth:refresh", 30, time.Minute, trustProxy)).Post("/auth/refresh", h.Refresh)
		r.Group(func(pr chi.Router) {
			pr.Use(middleware.RequireAuth(s))
			pr.Post("/auth/logout", h.Logout)
			pr.Get("/me", h.Me)
			pr.Patch("/me/profile", h.UpdateProfile)
			pr.With(middleware.RateLimitRules(redisClient, trustProxy,
				middleware.RateLimitRule{Name: "auth:password", Scope: middleware.RateLimitByIP, Limit: 5, Window: time.Minute},
				middleware.RateLimitRule{Name: "auth:password", Scope: middleware.RateLimitByUser, Limit: 5, Window: time.Minute},
			)).Post("/me/password", h.ChangePassword)
			pr.With(middleware.RequirePasswordChanged()).Get("/me/sessions", h.Sessions)
			pr.With(middleware.RequirePasswordChanged()).Delete("/me/sessions/{id}", h.RevokeSession)
		})
	}
}
