package router

import (
	"github.com/go-chi/chi/v5"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/auth"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/http/handler"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/middleware"
)

func AuthRoutes(s *auth.Service) DomainRegistrar {
	return func(r chi.Router) {
		h := handler.NewAuthHandler(s)
		r.Post("/auth/login", h.Login)
		r.Post("/auth/refresh", h.Refresh)
		r.Group(func(pr chi.Router) {
			pr.Use(middleware.RequireAuth(s))
			pr.Post("/auth/logout", h.Logout)
			pr.Get("/me", h.Me)
		})
	}
}
