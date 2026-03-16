package eventconsts

const (
	// Schemas
	SchemaSupportTicket = "support_ticket"
	SchemaApproval      = "approval"
	SchemaAction        = "action"
	SchemaAudit         = "audit"
	SchemaTicketType    = "ticket_type"

	// Sources
	SourceSupportTicket = "support-ticket"
)

const (
	// Ticket events
	TicketCreated       = "ticket.created"
	TicketUpdated       = "ticket.updated"
	TicketAssigned      = "ticket.assigned"
	TicketMerged        = "ticket.merged"
	TicketCommentAdded  = "ticket.comment.added"
	TicketDeleted       = "ticket.deleted"
	TicketTypeCreated   = "ticket_type.created"
	TicketTypeUpdated   = "ticket_type.updated"
	TicketTypeDeleted   = "ticket_type.deleted"

	// Approval events
	ApprovalRequested = "approval.requested"
	ApprovalDecided   = "approval.decided"

	// Action events
	ActionSchemaCreated          = "action.schema.created"
	ActionExecutionStatusUpdated = "action.execution.status.updated"
	ActionExecutionPending       = "action.execution.pending_approval"

	// Audit events
	AuditLogCreated = "audit.log.created"
)
