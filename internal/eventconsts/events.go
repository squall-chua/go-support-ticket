package eventconsts

const (
	// Schemas
	SchemaApprovalConfig = "approval_config"
	SchemaApproval       = "approval"
	SchemaAction         = "action"
	SchemaAudit          = "audit"
	SchemaTicketType     = "ticket_type"
	SchemaSupportTicket  = "ticket"

	// Sources
	SourceApproval = "support-ticket.approval"
	SourceAction   = "support-ticket.action"
	SourceTicket   = "support-ticket.ticket"
)

const (
	// Approval events
	ApprovalConfigCreated = "approval_config.created"
	ApprovalConfigUpdated = "approval_config.updated"
	ApprovalConfigDeleted = "approval_config.deleted"
	ApprovalRequested     = "approval.requested"
	ApprovalPending       = "approval.pending"
	ApprovalDecided       = "approval.decided"

	// Action events
	ActionSchemaCreated          = "action.schema.created"
	ActionSchemaUpdated          = "action.schema.updated"
	ActionSchemaDeleted          = "action.schema.deleted"
	ActionExecutionStatusUpdated = "action.execution.status.updated"
	ActionExecutionTriggered     = "action.execution.triggered"
	ActionExecutionExecuted      = "action.execution.executed"
	ActionExecutionCompleted     = "action.execution.completed"
	ActionExecutionPending       = "action.execution.pending_approval"
	ActionExecutionCancelled     = "action.execution.cancelled"

	// Ticket events
	TicketTypeCreated           = "ticket_type.created"
	TicketTypeUpdated           = "ticket_type.updated"
	TicketTypeDeleted           = "ticket_type.deleted"
	TicketCreated               = "ticket.created"
	TicketUpdated               = "ticket.updated"
	TicketUpdatePendingApproval = "ticket.update.pending_approval"
	TicketAssigned              = "ticket.assigned"
	TicketMerged                = "ticket.merged"
	TicketMergePendingApproval  = "ticket.merge.pending_approval"
	TicketCommentAdded          = "ticket.comment.added"
	TicketDeleted               = "ticket.deleted"
)
const (
	// Approval Action types
	ActionTicketUpdate = "ticket.update"
	ActionTicketMerge  = "ticket.merge"
)
