package repository

import (
	"context"
	"database/sql"
	"time"

	apiv1 "github.com/squall-chua/go-support-ticket/api/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AuditRepository interface {
	CreateLog(ctx context.Context, log *apiv1.AuditEntry) error
	ListLogs(ctx context.Context, ticketID, action string, limit, offset int32) ([]*apiv1.AuditEntry, int32, error)
}

type sqlAuditRepo struct {
	db *sql.DB
}

func NewAuditRepository(connector DBConnector) AuditRepository {
	return &sqlAuditRepo{
		db: connector.SQL(),
	}
}

func (r *sqlAuditRepo) CreateLog(ctx context.Context, log *apiv1.AuditEntry) error {
	query := `
		INSERT INTO audit_logs (id, ticket_id, action, actor, timestamp)
		VALUES ($1, $2, $3, $4, $5)`
	_, err := r.db.ExecContext(ctx, query,
		log.Id,
		log.TicketId,
		log.Action,
		log.Actor,
		log.CreatedAt.AsTime(),
	)
	return err
}

func (r *sqlAuditRepo) ListLogs(ctx context.Context, ticketID, action string, limit, offset int32) ([]*apiv1.AuditEntry, int32, error) {
	// A simplified static query.
	query := `SELECT id, ticket_id, action, actor, timestamp FROM audit_logs WHERE ticket_id = $1 LIMIT $2 OFFSET $3`
	rows, err := r.db.QueryContext(ctx, query, ticketID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []*apiv1.AuditEntry
	for rows.Next() {
		var log apiv1.AuditEntry
		var ts time.Time
		if err := rows.Scan(&log.Id, &log.TicketId, &log.Action, &log.Actor, &ts); err != nil {
			return nil, 0, err
		}
		log.CreatedAt = timestamppb.New(ts)
		logs = append(logs, &log)
	}

	total := int32(len(logs))
	return logs, total, nil
}
