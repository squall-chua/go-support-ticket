package repository

import (
	"context"
	"time"

	"github.com/squall-chua/go-support-ticket/internal/model"
)

type TicketRepository interface {
	CreateTicket(ctx context.Context, ticket *model.Ticket) error
	GetTicket(ctx context.Context, id string) (*model.Ticket, error)
	UpdateTicket(ctx context.Context, id string, update model.TicketUpdate) (*model.Ticket, error)
	UpdateTickets(ctx context.Context, updates map[string]model.TicketUpdate) ([]*model.Ticket, error)
	ListTickets(ctx context.Context, filter model.TicketFilter, sorts []model.TicketSort, limit, offset int32) ([]*model.Ticket, int32, error)
	AddComment(ctx context.Context, ticketID string, comment *model.Comment) error
	DeleteTicket(ctx context.Context, id string, deletedAt time.Time) error
}

type TicketTypeRepository interface {
	CreateType(ctx context.Context, tType *model.TicketType) error
	GetType(ctx context.Context, id string) (*model.TicketType, error)
	ListTypes(ctx context.Context, filter model.TicketTypeFilter, sorts []model.TicketTypeSort, limit, offset int32) ([]*model.TicketType, int32, error)
	UpdateType(ctx context.Context, id string, update model.TicketTypeUpdate) (*model.TicketType, error)
	DeleteType(ctx context.Context, id string, deletedAt time.Time) error
}

type ActionSchemaRepository interface {
	CreateSchema(ctx context.Context, schema *model.ActionSchema) error
	GetSchema(ctx context.Context, actionType string) (*model.ActionSchema, error)
	ListSchemas(ctx context.Context, filter model.ActionSchemaFilter, limit, offset int32) ([]*model.ActionSchema, int32, error)
	UpdateSchema(ctx context.Context, actionType string, update model.ActionSchemaUpdate) error
}

type ActionExecutionRepository interface {
	CreateExecution(ctx context.Context, execution *model.ActionExecution) error
	GetExecution(ctx context.Context, id string) (*model.ActionExecution, error)
	UpdateExecution(ctx context.Context, id string, update model.ActionExecutionUpdate) error
	ListExecutions(ctx context.Context, ticketID, actionType string, limit, offset int32) ([]*model.ActionExecution, int32, error)
}

type ApprovalRepository interface {
	CreateApproval(ctx context.Context, approval *model.Approval) error
	GetApproval(ctx context.Context, id string) (*model.Approval, error)
	UpdateApproval(ctx context.Context, id string, update model.ApprovalUpdate) error
	ListApprovals(ctx context.Context, filter model.ApprovalFilter, limit, offset int32) ([]*model.Approval, int32, error)
}

type AuditRepository interface {
	CreateLog(ctx context.Context, log *model.AuditLog) error
	ListLogs(ctx context.Context, ticketID, action string, limit, offset int32) ([]*model.AuditLog, int32, error)
}

// Repositories is a container for all domain repositories.
type Repositories struct {
	Tickets       TicketRepository
	TicketTypes   TicketTypeRepository
	Audit         AuditRepository
	Approvals     ApprovalRepository
	ActionSchemas ActionSchemaRepository
	Executions    ActionExecutionRepository
}
