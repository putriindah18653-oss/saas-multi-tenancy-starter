package health

import (
	"context"
	"net/http"
	"time"

	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/http/response"
)

type Checker interface{ Ping(context.Context) error }
type Dependencies struct{ DB, Redis Checker }
type Handler struct{ db, redis Checker }

func NewHandler(deps Dependencies) *Handler { return &Handler{db: deps.DB, redis: deps.Redis} }

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	response.Success(w, r, http.StatusOK, map[string]any{"status": "ok", "service": "api"})
}

// Ready checks database and Redis connectivity. Both healthy → 200.
// Either unhealthy → 503 with success:false and per-dependency status.
func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	checks := map[string]string{"database": "ok", "redis": "ok"}
	ready := true
	if h.db == nil || h.db.Ping(ctx) != nil {
		checks["database"] = "unavailable"
		ready = false
	}
	if h.redis == nil || h.redis.Ping(ctx) != nil {
		checks["redis"] = "unavailable"
		ready = false
	}
	if !ready {
		response.JSON(w, http.StatusServiceUnavailable, response.Envelope{
			Success: false,
			Data:    map[string]any{"status": "not_ready", "checks": checks},
			Error:   &response.ErrorBody{Code: "not_ready", Message: "service dependencies are unavailable"},
			RequestID: response.RequestID(r),
		})
		return
	}
	response.Success(w, r, http.StatusOK, map[string]any{"status": "ready", "checks": checks})
}
