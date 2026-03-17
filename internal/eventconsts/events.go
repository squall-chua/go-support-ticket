package eventconsts

const (
	// Schemas
	SchemaSupportTicket  = "support_ticket"
	SchemaApproval       = "approval"
	SchemaAction         = "action"
	SchemaAudit          = "audit"
	SchemaTicketType     = "ticket_type"
	SchemaApprovalConfig = "approval_config"

	// Sources
	SourceSupportTicket = "support-ticket"
)

const (
	// Ticket events
	TicketCreated      = "ticket.created"
	TicketUpdated      = "ticket.updated"
	TicketAssigned     = "ticket.assigned"
	TicketMerged       = "ticket.merged"
	TicketCommentAdded = "ticket.comment.added"
	TicketDeleted      = "ticket.deleted"
	TicketTypeCreated  = "ticket_type.created"
	TicketTypeUpdated  = "ticket_type.updated"
	TicketTypeDeleted  = "ticket_type.deleted"

	// Approval events
	ApprovalRequested     = "approval.requested"
	ApprovalDecided       = "approval.decided"
	ApprovalConfigCreated = "approval_config.created"
	ApprovalConfigUpdated = "approval_config.updated"
	ApprovalConfigDeleted = "approval_config.deleted"

	// Action events
	ActionSchemaCreated          = "action.schema.created"
	ActionSchemaUpdated          = "action.schema.updated"
	ActionSchemaDeleted          = "action.schema.deleted"
	ActionExecutionStatusUpdated = "action.execution.status.updated"
	ActionExecutionTriggered     = "action.execution.triggered"
	ActionExecutionExecuted      = "action.execution.executed"
	ActionExecutionCompleted     = "action.execution.completed"
	ActionExecutionPending       = "action.execution.pending_approval"
)
