// Package database provides the database connection pool and the WithRLS
// transaction wrapper that enforces PostgreSQL Row-Level Security.
//
// All tenant-scoped queries MUST use WithRLS (or WithRLSRead for read-only
// paths) instead of calling pool.Query/Exec directly. WithRLS opens a
// transaction, sets session-level GUC variables (app.current_tenant_id,
// app.current_user_id, app.is_platform_admin), runs the callback, and
// commits. WithRLSRead does the same but uses a read-only transaction.
//
// RLSContext controls which GUCs are set:
//   - TenantID + UserID: standard tenant-scoped operation
//   - PlatformAdmin: true: bypasses tenant RLS (use only for admin operations)
//   - UserID only: user-scoped query (e.g., listing own memberships)
//
// Migration 000006 installs the RLS policies and helper functions. New
// tables that hold tenant data must receive RLS policies before they are
// queried through WithRLS.
package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Querier abstracts the subset of pgx.Tx used by service callbacks so they
// cannot accidentally call Commit or Rollback.
type Querier interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

// RLSContext carries the tenant/user identity to inject into PostgreSQL
// session-level GUCs before executing tenant-scoped queries.
type RLSContext struct {
	TenantID      string
	UserID        string
	PlatformAdmin bool
}

// SetLocalRLS configures session-level GUC variables on the given
// transaction so that PostgreSQL RLS policies can read them via
// current_setting(). All three settings are set in a single round-trip.
func SetLocalRLS(ctx context.Context, tx pgx.Tx, rls RLSContext) error {
	pa := "false"
	if rls.PlatformAdmin {
		pa = "true"
	}
	_, err := tx.Exec(ctx,
		"SELECT set_config('app.current_tenant_id', $1, true),"+
			"     set_config('app.current_user_id',   $2, true),"+
			"     set_config('app.is_platform_admin',  $3, true)",
		rls.TenantID, rls.UserID, pa)
	return err
}

// WithRLS wraps fn in a read-write transaction with RLS GUCs set.
// Errors from Begin, SetLocalRLS, fn, or Commit are returned.
func WithRLS(ctx context.Context, pool *pgxpool.Pool, rls RLSContext, fn func(Querier) error) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("rls: context cancelled before begin: %w", err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("rls: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("rls: context cancelled after begin: %w", err)
	}
	if err := SetLocalRLS(ctx, tx, rls); err != nil {
		return fmt.Errorf("rls: set guc: %w", err)
	}
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
