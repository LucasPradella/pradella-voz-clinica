package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

// AuditService persists access events for LGPD compliance.
// It MUST NOT store patient PII in the metadata field.
type AuditService struct {
	db *pgxpool.Pool
}

func NewAuditService(db *pgxpool.Pool) *AuditService {
	return &AuditService{db: db}
}

// Log records an audit event. userID and resourceID may be empty strings (stored as NULL).
// metadata must not contain patient PII.
func (s *AuditService) Log(ctx context.Context, userID, action, resourceID string, metadata map[string]interface{}) {
	meta, err := json.Marshal(metadata)
	if err != nil {
		slog.Error("audit: marshal metadata", "err", err)
		meta = []byte("{}")
	}

	var uid, rid interface{}
	if userID != "" {
		uid = userID
	}
	if resourceID != "" {
		rid = resourceID
	}

	_, err = s.db.Exec(ctx,
		`INSERT INTO audit_logs (user_id, action, resource_id, metadata)
		 VALUES ($1, $2, $3, $4)`,
		uid, action, rid, meta,
	)
	if err != nil {
		// Audit failures are logged but must not disrupt the main request.
		slog.Error("audit: insert failed", "action", action, "err", fmt.Sprintf("%v", err))
	}
}
