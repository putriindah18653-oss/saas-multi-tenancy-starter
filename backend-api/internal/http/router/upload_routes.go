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

		// Public read-only access for processed upload assets (logos, avatars).
		//
		// Security note:
		//   - These are NON-SECRET, already-processed AVIF assets meant to be rendered
		//     by plain <img> tags, which cannot attach an Authorization header. Gating
		//     reads behind auth broke normal image display, so reads are intentionally
		//     public.
		//   - Confidentiality relies on unguessable paths: filenames embed a 12-byte
		//     crypto/rand id (see media.AVIFProcessor), so URLs are not enumerable.
		//   - http.FileServer over http.Dir scopes access to storage/uploads and rejects
		//     path traversal (e.g. ../), so only files under that directory are served.
		//   - This is READ-ONLY. Writes stay locked down below: every upload POST still
		//     requires auth + password-changed gate + permission + rate limiting.
		fileServer := http.StripPrefix("/api/v1/uploads/", http.FileServer(http.Dir("storage/uploads")))
		r.Get("/uploads/*", func(w http.ResponseWriter, req *http.Request) {
			fileServer.ServeHTTP(w, req)
		})


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
