package audit

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
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
func (s *Service) Log(ctx context.Context, actorUserID, tenantID, action, resourceType, resourceID string, meta any, ip, ua string) error {
	b, _ := json.Marshal(meta)
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
	_, err := s.db.Exec(ctx, "INSERT INTO audit_logs(actor_user_id,tenant_id,action,resource_type,resource_id,metadata,ip_address,user_agent) VALUES($1,$2,$3,$4,$5,$6,$7,$8)", au, ti, action, resourceType, ri, string(b), ip, ua)
	return err
}
func (s *Service) ListApp(ctx context.Context, limit int) ([]Entry, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	rows, err := s.db.Query(ctx, "SELECT id,actor_user_id,tenant_id,action,resource_type,resource_id,metadata,ip_address,user_agent,created_at FROM audit_logs ORDER BY created_at DESC LIMIT $1", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEntries(rows)
}
func (s *Service) ListTenant(ctx context.Context, tenantID string, limit int) ([]Entry, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	rows, err := s.db.Query(ctx, "SELECT id,actor_user_id,tenant_id,action,resource_type,resource_id,metadata,ip_address,user_agent,created_at FROM audit_logs WHERE tenant_id=$1 ORDER BY created_at DESC LIMIT $2", tenantID, limit)
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
