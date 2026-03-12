package repository

// Repositories is a container for all domain repositories.
type Repositories struct {
	Tickets       TicketRepository
	Audit         AuditRepository
	Approvals     ApprovalRepository
	ActionSchemas ActionSchemaRepository
	Executions    ActionExecutionRepository
}

// NewRepositories creates and wires up all the necessary repositories.
func NewRepositories(connector DBConnector) *Repositories {
	return &Repositories{
		Tickets:       NewTicketRepository(connector),
		Audit:         NewAuditRepository(connector),
		Approvals:     NewApprovalRepository(connector),
		ActionSchemas: NewActionSchemaRepository(connector),
		Executions:    NewActionExecutionRepository(connector),
	}
}
