package router

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/config"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/health"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/middleware"
)

type Dependencies struct {
	Config    *config.Config
	DB, Redis health.Checker
	Logger    *slog.Logger
}
type DomainRegistrar func(chi.Router)

func New(deps Dependencies, registrars ...DomainRegistrar) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recovery(deps.Logger))
	r.Use(middleware.Logging(deps.Logger))
	if deps.Config != nil {
		r.Use(middleware.CORS(deps.Config.CORS.AllowedOrigins))
	}
	h := health.NewHandler(health.Dependencies{DB: deps.DB, Redis: deps.Redis})
	r.Get("/health", h.Health)
	r.Get("/ready", h.Ready)
	r.Route("/api/v1", func(api chi.Router) {
		for _, register := range registrars {
			register(api)
		}
	})
	return r
}
