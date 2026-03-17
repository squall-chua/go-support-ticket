package repository

import (
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// NewRepositories initializes and returns all domain repositories using MongoDB.
func NewRepositories(db *mongo.Database) *Repositories {
	return &Repositories{
		Tickets:         NewTicketRepo(db.Collection("tickets")),
		TicketTypes:     NewTicketTypeRepo(db.Collection("ticket_types")),
		Audit:           NewAuditRepo(db.Collection("audit_logs")),
		Approvals:       NewApprovalRepo(db.Collection("approvals")),
		ApprovalConfigs: NewApprovalConfigRepo(db.Collection("approval_configs")),
		ActionSchemas:   NewActionSchemaRepo(db.Collection("action_schemas")),
		Executions:      NewActionExecutionRepo(db.Collection("action_executions")),
	}
}
