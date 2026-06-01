package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/database"
)

type Service struct{ db *pgxpool.Pool }
type Entry struct {
	ID           string          `json:"id"`
	ActorUserID  *string         `json:"actor_user_id,omitempty"`
	TenantID     *string         `json:"tenant_id,omitempty"`
	Action       string          `json:"action"`
	ResourceType string          `json:"resource_type"`
	ResourceID   *string         `json:"resource_id,omitempty"`
	Metadata     json.RawMessage `json:"metadata"`
	IPAddress    string          `json:"ip_address"`
	UserAgent    string          `json:"user_agent"`
	CreatedAt    time.Time       `json:"created_at"`
}

func NewService(db *pgxpool.Pool) *Service { return &Service{db: db} }

// Log writes an audit entry. tenantID == "" signals a platform-scoped event
// (PlatformAdmin=true). Pass the actual tenant ID for tenant-scoped events.
func (s *Service) Log(ctx context.Context, actorUserID, tenantID, action, resourceType, resourceID string, meta any, ip, ua string) error {
	platformAdmin := tenantID == ""
	rls := database.RLSContext{TenantID: tenantID, UserID: actorUserID, PlatformAdmin: platformAdmin}
	return database.WithRLS(ctx, s.db, rls, func(q database.Querier) error {
		return s.log(ctx, q, actorUserID, tenantID, action, resourceType, resourceID, meta, ip, ua)
	})
}

func (s *Service) log(ctx context.Context, q database.Querier, actorUserID, tenantID, action, resourceType, resourceID string, meta any, ip, ua string) error {
	b, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("audit: marshal metadata: %w", err)
	}
	var au, ti, ri any
	if actorUserID != "" {
		au = actorUserID
	}
	if tenantID != "" {
		ti = tenantID
	}
	if resourceID != "" {
		ri = resourceID
	}
	_, err = q.Exec(ctx, "INSERT INTO audit_logs(actor_user_id,tenant_id,action,resource_type,resource_id,metadata,ip_address,user_agent) VALUES($1,$2,$3,$4,$5,$6,$7,$8)", au, ti, action, resourceType, ri, string(b), ip, ua)
	return err
}

func (s *Service) ListApp(ctx context.Context, limit int) ([]Entry, error) {
	var out []Entry
	err := database.WithRLS(ctx, s.db, database.RLSContext{PlatformAdmin: true}, func(q database.Querier) error {
		var err error
		out, err = s.listApp(ctx, q, limit)
		return err
	})
	return out, err
}
func (s *Service) listApp(ctx context.Context, q database.Querier, limit int) ([]Entry, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	rows, err := q.Query(ctx, "SELECT id,actor_user_id,tenant_id,action,resource_type,resource_id,metadata,ip_address,user_agent,created_at FROM audit_logs ORDER BY created_at DESC LIMIT $1", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEntries(rows)
}
func (s *Service) ListTenant(ctx context.Context, tenantID string, limit int) ([]Entry, error) {
	var out []Entry
	err := database.WithRLS(ctx, s.db, database.RLSContext{TenantID: tenantID}, func(q database.Querier) error {
		var err error
		out, err = s.listTenant(ctx, q, tenantID, limit)
		return err
	})
	return out, err
}
func (s *Service) listTenant(ctx context.Context, q database.Querier, tenantID string, limit int) ([]Entry, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	rows, err := q.Query(ctx, "SELECT id,actor_user_id,tenant_id,action,resource_type,resource_id,metadata,ip_address,user_agent,created_at FROM audit_logs WHERE tenant_id=$1 ORDER BY created_at DESC LIMIT $2", tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEntries(rows)
}

type rowScanner interface {
	Next() bool
	Scan(...any) error
	Err() error
}

func scanEntries(rows rowScanner) ([]Entry, error) {
	out := []Entry{}
	for rows.Next() {
		var e Entry
		if err := rows.Scan(&e.ID, &e.ActorUserID, &e.TenantID, &e.Action, &e.ResourceType, &e.ResourceID, &e.Metadata, &e.IPAddress, &e.UserAgent, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
