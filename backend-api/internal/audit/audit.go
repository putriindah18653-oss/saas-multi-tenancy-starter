package audit

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct{ db *pgxpool.Pool }

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
