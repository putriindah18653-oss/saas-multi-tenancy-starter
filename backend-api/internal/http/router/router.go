package router

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/config"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/health"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/metrics"
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

	// Application middleware stack.
	r.Use(middleware.RequestID)
	r.Use(middleware.Recovery(deps.Logger))
	r.Use(middleware.SecurityHeaders)
	r.Use(metrics.Middleware)
	r.Use(middleware.Logging(deps.Logger))
	if deps.Config != nil {
		r.Use(middleware.CORS(deps.Config.CORS.AllowedOrigins))
	}

	// Infrastructure endpoints are wrapped in explicit Route groups so
	// the metrics middleware labels them with correct route patterns.
	h := health.NewHandler(health.Dependencies{DB: deps.DB, Redis: deps.Redis})
	r.Route("/health", func(mr chi.Router) { mr.Get("/", h.Health) })
	r.Route("/ready", func(mr chi.Router) { mr.Get("/", h.Ready) })
	// /metrics is intentionally unauthenticated so Prometheus can scrape it.
	// Protect it at the reverse-proxy / firewall layer in production.
	r.Route("/metrics", func(mr chi.Router) { mr.Handle("/", promhttp.Handler()) })

	r.Route("/api/v1", func(api chi.Router) {
		for _, register := range registrars {
			register(api)
		}
	})
	return r
}
