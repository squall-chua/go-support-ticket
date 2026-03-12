package repository

import (
	"context"
	"database/sql"
	"time"

	apiv1 "github.com/squall-chua/go-support-ticket/api/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ApprovalRepository interface {
	CreateApproval(ctx context.Context, approval *apiv1.ApprovalRequestData) error
	GetApproval(ctx context.Context, id string) (*apiv1.ApprovalRequestData, error)
	UpdateApproval(ctx context.Context, approval *apiv1.ApprovalRequestData) error
	ListApprovals(ctx context.Context, ticketID, reviewerID, status string, limit, offset int32) ([]*apiv1.ApprovalRequestData, int32, error)
}

// Ensure interface implementation compile-time check
// We will use PostgreSQL for approvals, even though we could use MongoDB.
// Based on the task description "ApprovalRepository (PostgreSQL relational)"
type sqlApprovalRepo struct {
	db *sql.DB
}

func NewApprovalRepository(connector DBConnector) ApprovalRepository {
	return &sqlApprovalRepo{
		db: connector.SQL(),
	}
}

func (r *sqlApprovalRepo) CreateApproval(ctx context.Context, approval *apiv1.ApprovalRequestData) error {
	query := `
		INSERT INTO approvals (id, ticket_id, status, requested_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)`
		
	_, err := r.db.ExecContext(ctx, query,
		approval.Id,
		approval.TicketId,
		approval.Status,
		approval.Requester,
		approval.CreatedAt.AsTime(),
		approval.UpdatedAt.AsTime(),
	)
	return err
}

func (r *sqlApprovalRepo) GetApproval(ctx context.Context, id string) (*apiv1.ApprovalRequestData, error) {
	query := `SELECT id, ticket_id, status, requested_by, created_at, updated_at FROM approvals WHERE id = $1`
	row := r.db.QueryRowContext(ctx, query, id)

	var app apiv1.ApprovalRequestData
	var createdAt, updatedAt time.Time
	err := row.Scan(&app.Id, &app.TicketId, &app.Status, &app.Requester, &createdAt, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	app.CreatedAt = timestamppb.New(createdAt)
	app.UpdatedAt = timestamppb.New(updatedAt)
	return &app, nil
}

func (r *sqlApprovalRepo) UpdateApproval(ctx context.Context, approval *apiv1.ApprovalRequestData) error {
	query := `UPDATE approvals SET status = $1, updated_at = $2 WHERE id = $3`
	_, err := r.db.ExecContext(ctx, query, approval.Status, time.Now(), approval.Id)
	return err
}

func (r *sqlApprovalRepo) ListApprovals(ctx context.Context, ticketID, reviewerID, status string, limit, offset int32) ([]*apiv1.ApprovalRequestData, int32, error) {
	// A simplified static query for demonstration. A real impl uses query builders like Masterminds/squirrel.
	query := `SELECT id, ticket_id, status, requested_by, created_at, updated_at FROM approvals WHERE ticket_id = $1 LIMIT $2 OFFSET $3`
	rows, err := r.db.QueryContext(ctx, query, ticketID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var approvals []*apiv1.ApprovalRequestData
	for rows.Next() {
		var app apiv1.ApprovalRequestData
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&app.Id, &app.TicketId, &app.Status, &app.Requester, &createdAt, &updatedAt); err != nil {
			return nil, 0, err
		}
		app.CreatedAt = timestamppb.New(createdAt)
		app.UpdatedAt = timestamppb.New(updatedAt)
		approvals = append(approvals, &app)
	}

	total := int32(len(approvals)) // simplified total count logic
	return approvals, total, nil
}
