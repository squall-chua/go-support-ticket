package service

import (
	"context"

	"github.com/squall-chua/go-event-pubsub/pkg/event"
	apiv1 "github.com/squall-chua/go-support-ticket/api/v1"
	"github.com/squall-chua/go-support-ticket/internal/eventbus"
	"github.com/squall-chua/go-support-ticket/internal/eventconsts"
	"github.com/squall-chua/go-support-ticket/internal/middleware"
	"github.com/squall-chua/go-support-ticket/internal/model"
	"github.com/squall-chua/go-support-ticket/internal/repository"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

type AuditServiceServer struct {
	apiv1.UnimplementedAuditServiceServer
	repo       repository.AuditRepository
	ticketRepo repository.TicketRepository
}

func NewAuditServiceServer(repo repository.AuditRepository, ticketRepo repository.TicketRepository) *AuditServiceServer {
	return &AuditServiceServer{
		repo:       repo,
		ticketRepo: ticketRepo,
	}
}

func (s *AuditServiceServer) ListAuditTrail(ctx context.Context, req *apiv1.ListAuditTrailRequest) (*apiv1.ListAuditTrailResponse, error) {
	limit, offset, pageNumber := getPaginationParams(req.Pagination)

	var metadataFilters []model.MetadataFilter
	for _, m := range req.Metadata {
		metadataFilters = append(metadataFilters, model.MetadataFilter{
			Key:      m.Key,
			Operator: model.MetadataOperator(m.Operator),
			Value:    m.Value.AsInterface(),
		})
	}

	filter := model.AuditLogFilter{
		EventIDs:    req.EventIds,
		EventTypes:  req.EventTypes,
		Users:       req.Users,
		Sources:     req.Sources,
		Schemas:     req.Schemas,
		ResourceIDs: req.ResourceIds,
		Metadata:    metadataFilters,
	}

	start, end := getTimeRange(req.TimeRange)
	if start != nil {
		filter.StartTime = *start
	}
	if end != nil {
		filter.EndTime = *end
	}

	logs, total, err := s.repo.ListLogs(ctx, filter, limit, offset)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	pbEntries := make([]*apiv1.AuditEntry, 0, len(logs))
	for _, l := range logs {
		pbEntries = append(pbEntries, l.ToProto())
	}

	return &apiv1.ListAuditTrailResponse{
		Entries: pbEntries,
		Pagination: &apiv1.PageInfo{
			TotalSize:  total,
			PageNumber: pageNumber,
		},
	}, nil
}

func (s *AuditServiceServer) GetTicketAuditTrail(ctx context.Context, req *apiv1.GetTicketAuditTrailRequest) (*apiv1.GetTicketAuditTrailResponse, error) {
	user, ok := middleware.TokenInfoFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user info not found in context")
	}

	// 1. Check if the user is an admin.
	isAdmin := false
	for _, role := range user.Roles {
		if role == "admin" {
			isAdmin = true
			break
		}
	}

	// 2. If not admin, check if the ticket exists and if the user has visibility.
	if !isAdmin {
		_, err := s.ticketRepo.GetTicket(ctx, req.TicketId, user.UserID, user.Roles)
		if err != nil {
			return nil, status.Error(codes.NotFound, "ticket not found or access denied")
		}
	}

	// 3. Query the audit trail for this ticket.
	limit, offset, pageNumber := getPaginationParams(req.Pagination)

	filter := model.AuditLogFilter{
		ResourceIDs: []string{req.TicketId},
	}

	logs, total, err := s.repo.ListLogs(ctx, filter, limit, offset)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	pbEntries := make([]*apiv1.AuditEntry, 0, len(logs))
	for _, l := range logs {
		pbEntries = append(pbEntries, l.ToProto())
	}

	return &apiv1.GetTicketAuditTrailResponse{
		Entries: pbEntries,
		Pagination: &apiv1.PageInfo{
			TotalSize:  total,
			PageNumber: pageNumber,
		},
	}, nil
}

func (s *AuditServiceServer) RegisterHandlers(subscriber event.Subscriber) {
	// Register for all events in schemas that require auditing using wildcard subscription.
	schemas := []string{
		eventconsts.SchemaApprovalConfig,
		eventconsts.SchemaApproval,
		eventconsts.SchemaAction,
		eventconsts.SchemaTicketType,
		eventconsts.SchemaSupportTicket,
	}

	for _, schema := range schemas {
		_ = subscriber.Subscribe(schema, "*", s.HandleEvent)
	}
}

func (s *AuditServiceServer) HandleEvent(ctx context.Context, evt *event.Event) error {
	auditLog := &model.AuditLog{
		EventID:    evt.EventId,
		EventType:  evt.EventType,
		EventTime:  evt.EventTime,
		User:       evt.User,
		Source:     evt.Source,
		Schema:     evt.Schema,
		ResourceID: evt.ResourceID,
		Metadata:   make(map[string]interface{}),
	}

	if data, ok := evt.Data.(proto.Message); ok {
		if anyData, ok := data.(*anypb.Any); ok {
			auditLog.Data = eventbus.ProtoMarshaler{Message: anyData}
		} else if anyData, err := anypb.New(data); err == nil {
			auditLog.Data = eventbus.ProtoMarshaler{Message: anyData}
		}
	} else if marshaler, ok := evt.Data.(eventbus.ProtoMarshaler); ok {
		if anyData, err := anypb.New(marshaler.Message); err == nil {
			auditLog.Data = eventbus.ProtoMarshaler{Message: anyData}
		}
	}

	// Populate metadata
	for k, v := range evt.Metadata {
		auditLog.Metadata[k] = v
	}

	return s.repo.CreateLog(ctx, auditLog)
}
