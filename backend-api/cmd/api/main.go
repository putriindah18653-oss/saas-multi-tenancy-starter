package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/audit"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/auth"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/config"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/database"
	apirouter "github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/http/router"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/rbac"
	apiredis "github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/redis"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/tenant"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/user"
)

// secureCookie is derived from !IsDevelopment() so that refresh-token
// cookies are automatically Secure in staging/production. This requires
// HTTPS; non-HTTPS production deployments will silently fail to set cookies.
func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := config.Load()
	if err != nil {
		logger.Error("load config", "error", err)
		os.Exit(1)
	}
	ctx := context.Background()
	db, err := database.Connect(ctx, cfg.Database)
	if err != nil {
		logger.Error("connect database", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	redisClient, err := apiredis.Connect(ctx, cfg.Redis)
	if err != nil {
		logger.Error("connect redis", "error", err)
		os.Exit(1)
	}
	defer func() { _ = redisClient.Close() }()
	authSvc := auth.NewService(db.Pool, cfg.JWT)
	rbacSvc := rbac.NewService(db.Pool)
	tenantSvc := tenant.NewService(db.Pool)
	auditSvc := audit.NewService(db.Pool)
	userSvc := user.NewService(db.Pool)
	srv := &http.Server{
		Addr: cfg.HTTP.Addr,
		Handler: apirouter.New(
			apirouter.Dependencies{Config: cfg, DB: db, Redis: redisClient, Logger: logger},
			apirouter.AuthRoutes(authSvc, auditSvc, redisClient.Client, cfg.App.TrustProxy, cfg.JWT),
			apirouter.RBACRoutes(authSvc, rbacSvc),
			apirouter.AppTenantRoutes(authSvc, rbacSvc, tenantSvc, auditSvc, redisClient.Client, cfg.App.TrustProxy),
			apirouter.AppSettingsRoutes(authSvc, rbacSvc, tenantSvc, auditSvc, cfg.App.TrustProxy),
			apirouter.UploadRoutes(authSvc, rbacSvc, redisClient.Client, cfg.App.TrustProxy),
			apirouter.TenantUserRoutes(authSvc, rbacSvc, userSvc, auditSvc, redisClient.Client, cfg.App.TrustProxy, cfg.App.InternalProxySecret),
			apirouter.TenantSettingsRoutes(authSvc, rbacSvc, tenantSvc, auditSvc, redisClient.Client, cfg.App.TrustProxy, cfg.App.InternalProxySecret),
			apirouter.AuditRoutes(authSvc, rbacSvc, auditSvc, cfg.App.InternalProxySecret),
		),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		logger.Info("api server listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()
	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, os.Interrupt, syscall.SIGTERM)
	select {
	case sig := <-stopCh:
		logger.Info("shutdown signal received", "signal", sig.String())
	case err := <-errCh:
		logger.Error("server failed", "error", err)
		os.Exit(1)
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}
	logger.Info("api server stopped")
}
