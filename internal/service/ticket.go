package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/squall-chua/go-event-pubsub/pkg/event"
	apiv1 "github.com/squall-chua/go-support-ticket/api/v1"
	"github.com/squall-chua/go-support-ticket/internal/eventbus"
	"github.com/squall-chua/go-support-ticket/internal/eventconsts"
	"github.com/squall-chua/go-support-ticket/internal/middleware"
	"github.com/squall-chua/go-support-ticket/internal/model"
	"github.com/squall-chua/go-support-ticket/internal/repository"
	"go.mongodb.org/mongo-driver/v2/bson"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const SystemAuthor = "system"

type TicketServiceServer struct {
	apiv1.UnimplementedTicketServiceServer
	repo      repository.TicketRepository
	typeRepo  repository.TicketTypeRepository
	publisher event.Publisher
}

func (s *TicketServiceServer) getUserRoles(ctx context.Context) []string {
	tokenInfo, ok := middleware.TokenInfoFromContext(ctx)
	if !ok || tokenInfo == nil {
		return []string{}
	}
	return tokenInfo.Roles
}

func (s *TicketServiceServer) getUserID(ctx context.Context) string {
	tokenInfo, ok := middleware.TokenInfoFromContext(ctx)
	if !ok || tokenInfo == nil {
		return ""
	}
	return tokenInfo.UserID
}

func (s *TicketServiceServer) isInternal(roles []string) bool {
	for _, r := range roles {
		if r == "admin" || r == "agent" {
			return true
		}
	}
	return false
}

func NewTicketServiceServer(repo repository.TicketRepository, typeRepo repository.TicketTypeRepository, publisher event.Publisher) apiv1.TicketServiceServer {
	return &TicketServiceServer{
		repo:      repo,
		typeRepo:  typeRepo,
		publisher: publisher,
	}
}

func (s *TicketServiceServer) CreateTicketType(ctx context.Context, req *apiv1.CreateTicketTypeRequest) (*apiv1.CreateTicketTypeResponse, error) {
	tType := &model.TicketType{
		ID:              bson.NewObjectID(),
		Name:            req.Name,
		DisplayName:     req.DisplayName,
		Description:     req.Description,
		RequireApproval: req.RequireApproval,
		VisibleRoles:    req.VisibleRoles,
		Activated:       req.Activated,
	}

	if err := s.typeRepo.CreateType(ctx, tType); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	resp := &apiv1.CreateTicketTypeResponse{
		TicketType: tType.ToProto(),
	}

	evt := event.Event{
		EventId:    uuid.NewString(),
		EventType:  eventconsts.TicketTypeCreated,
		EventTime:  time.Now().UTC(),
		Source:     eventconsts.SourceSupportTicket,
		Schema:     eventconsts.SchemaTicketType,
		ResourceID: tType.ID.Hex(),
		Data:       eventbus.ProtoMarshaler{Message: resp.TicketType},
	}
	_ = s.publisher.Publish(ctx, &evt)

	return resp, nil
}

func (s *TicketServiceServer) ListTicketTypes(ctx context.Context, req *apiv1.ListTicketTypesRequest) (*apiv1.ListTicketTypesResponse, error) {
	activated := req.Activated
	if activated == nil {
		active := true
		activated = &active
	}

	filter := model.TicketTypeFilter{
		Name:            req.Name,
		DisplayName:     req.DisplayName,
		Description:     req.Description,
		RequireApproval: req.RequireApproval,
		VisibleRoles:    req.VisibleRoles,
		Activated:       activated,
		IncludeDeleted:  req.IncludeDeleted,
	}

	var sorts []model.TicketTypeSort
	for _, sort := range req.Sorts {
		var field string
		switch sort.Field {
		case apiv1.TicketTypeSort_FIELD_NAME:
			field = "name"
		case apiv1.TicketTypeSort_FIELD_CREATED_AT:
			field = "created_at"
		case apiv1.TicketTypeSort_FIELD_UPDATED_AT:
			field = "updated_at"
		}
		if field != "" {
			order := 1
			if sort.Order == apiv1.SortOrder_SORT_ORDER_DESC {
				order = -1
			}
			sorts = append(sorts, model.TicketTypeSort{Field: field, Order: order})
		}
	}

	limit, offset, pageNumber := getPaginationParams(req.Pagination)

	types, total, err := s.typeRepo.ListTypes(ctx, filter, sorts, limit, offset)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	pbTypes := make([]*apiv1.TicketType, 0, len(types))
	for _, t := range types {
		pbTypes = append(pbTypes, t.ToProto())
	}

	return &apiv1.ListTicketTypesResponse{
		TicketTypes: pbTypes,
		Pagination: &apiv1.PageInfo{
			TotalSize:  total,
			PageNumber: pageNumber,
		},
	}, nil
}

func (s *TicketServiceServer) UpdateTicketType(ctx context.Context, req *apiv1.UpdateTicketTypeRequest) (*apiv1.UpdateTicketTypeResponse, error) {
	update := model.TicketTypeUpdate{
		DisplayName:     req.DisplayName,
		Description:     req.Description,
		RequireApproval: req.RequireApproval,
		VisibleRoles:    req.VisibleRoles,
		Activated:       req.Activated,
	}

	tType, err := s.typeRepo.UpdateType(ctx, req.Id, update)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if tType == nil {
		return nil, status.Error(codes.NotFound, "ticket type not found")
	}

	resp := &apiv1.UpdateTicketTypeResponse{
		TicketType: tType.ToProto(),
	}

	evt := event.Event{
		EventId:    uuid.NewString(),
		EventType:  eventconsts.TicketTypeUpdated,
		EventTime:  time.Now().UTC(),
		Source:     eventconsts.SourceSupportTicket,
		Schema:     eventconsts.SchemaTicketType,
		ResourceID: tType.ID.Hex(),
		Data:       eventbus.ProtoMarshaler{Message: resp.TicketType},
	}
	_ = s.publisher.Publish(ctx, &evt)

	return resp, nil
}

func (s *TicketServiceServer) DeleteTicketType(ctx context.Context, req *apiv1.DeleteTicketTypeRequest) (*apiv1.DeleteTicketTypeResponse, error) {
	if err := s.typeRepo.DeleteType(ctx, req.Id); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	evt := event.Event{
		EventId:    uuid.NewString(),
		EventType:  eventconsts.TicketTypeDeleted,
		EventTime:  time.Now().UTC(),
		Source:     eventconsts.SourceSupportTicket,
		Schema:     eventconsts.SchemaTicketType,
		ResourceID: req.Id,
	}
	_ = s.publisher.Publish(ctx, &evt)

	return &apiv1.DeleteTicketTypeResponse{}, nil
}

func (s *TicketServiceServer) CreateTicket(ctx context.Context, req *apiv1.CreateTicketRequest) (*apiv1.CreateTicketResponse, error) {
	ticket := &model.Ticket{
		ID:          bson.NewObjectID(),
		Title:       req.Title,
		Description: req.Description,
		Priority:    int32(req.Priority),
		Status:      int32(apiv1.TicketStatus_TICKET_STATUS_OPEN),
		TicketType:  req.TicketType,
		CustomerID:  req.CustomerId,
	}

	userID := s.getUserID(ctx)

	if !s.isInternal(s.getUserRoles(ctx)) || ticket.CustomerID == "" {
		ticket.CustomerID = userID
	}
	ticket.CreatedBy = userID

	// Fetch TicketType to cache visibility and approval settings
	// Not checking for empty ticket type because empty ticket type is considered as default ticket type
	if tt, err := s.typeRepo.GetType(ctx, req.TicketType); err == nil && tt != nil {
		if !tt.Activated {
			return nil, status.Error(codes.FailedPrecondition, "ticket type is not activated")
		}
		ticket.RequireApproval = tt.RequireApproval
		ticket.VisibleRoles = tt.VisibleRoles
	}

	if err := s.repo.CreateTicket(ctx, ticket); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	pb, err := ticket.ToProto()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	payload := map[string]interface{}{"ticket_id": ticket.ID.Hex(), "status": pb.Status.String()}
	evt := event.Event{
		EventId:    uuid.NewString(),
		EventType:  eventconsts.TicketCreated,
		EventTime:  time.Now().UTC(),
		Source:     eventconsts.SourceSupportTicket,
		Schema:     eventconsts.SchemaSupportTicket,
		ResourceID: ticket.ID.Hex(),
		Data:       payload,
	}
	_ = s.publisher.Publish(ctx, &evt)

	resp := &apiv1.CreateTicketResponse{
		Ticket: pb,
	}

	return resp, nil
}

func (s *TicketServiceServer) GetTicket(ctx context.Context, req *apiv1.GetTicketRequest) (*apiv1.GetTicketResponse, error) {
	ticket, err := s.repo.GetTicket(ctx, req.TicketId, s.getUserID(ctx), s.getUserRoles(ctx))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if ticket == nil {
		return nil, status.Error(codes.NotFound, "ticket not found")
	}

	pb, err := ticket.ToProto()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &apiv1.GetTicketResponse{
		Ticket: pb,
	}, nil
}

func (s *TicketServiceServer) UpdateTicket(ctx context.Context, req *apiv1.UpdateTicketRequest) (*apiv1.UpdateTicketResponse, error) {
	userID := s.getUserID(ctx)
	roles := s.getUserRoles(ctx)

	update := model.TicketUpdate{
		Title:       req.Title,
		Description: req.Description,
		TicketType:  req.TicketType,
	}

	if req.Priority != nil {
		p := int32(req.GetPriority())
		update.Priority = &p
	}
	if req.Status != nil {
		st := int32(req.GetStatus())
		update.Status = &st
	}

	if len(req.Metadata) > 0 {
		update.Metadata = make(model.Metadata)
		for k, v := range req.Metadata {
			update.Metadata[k] = v.AsInterface()
		}
	}

	if req.TicketType != nil {
		if tt, err := s.typeRepo.GetType(ctx, *req.TicketType); err == nil && tt != nil {
			if !tt.Activated {
				return nil, status.Error(codes.FailedPrecondition, "ticket type is not activated")
			}
			update.RequireApproval = &tt.RequireApproval
			update.VisibleRoles = tt.VisibleRoles
		}
	}

	ticket, err := s.repo.UpdateTicket(ctx, req.TicketId, update, userID, roles)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if ticket == nil {
		return nil, status.Error(codes.NotFound, "ticket not found")
	}

	pb, err := ticket.ToProto()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	evt := event.Event{
		EventId:    uuid.NewString(),
		EventType:  eventconsts.TicketUpdated,
		EventTime:  time.Now().UTC(),
		Source:     eventconsts.SourceSupportTicket,
		Schema:     eventconsts.SchemaSupportTicket,
		ResourceID: ticket.ID.Hex(),
		Data:       eventbus.ProtoMarshaler{Message: pb},
	}
	_ = s.publisher.Publish(ctx, &evt)

	return &apiv1.UpdateTicketResponse{
		Ticket: pb,
	}, nil
}

func (s *TicketServiceServer) publishTicketAssigned(ctx context.Context, ticket *model.Ticket) {
	payload := map[string]interface{}{"ticket_id": ticket.ID.Hex(), "assignee_id": ticket.AssignedTo}
	evt := event.Event{
		EventId:    uuid.NewString(),
		EventType:  eventconsts.TicketAssigned,
		EventTime:  time.Now().UTC(),
		Source:     eventconsts.SourceSupportTicket,
		Schema:     eventconsts.SchemaSupportTicket,
		ResourceID: ticket.ID.Hex(),
		Data:       payload,
		Metadata: map[string]any{
			"assigned_to": ticket.AssignedTo,
		},
	}
	_ = s.publisher.Publish(ctx, &evt)
}

func (s *TicketServiceServer) publishTicketMerged(ctx context.Context, source *model.Ticket, target *model.Ticket) {
	targetPb, _ := target.ToProto()
	evt := event.Event{
		EventId:    uuid.NewString(),
		EventType:  eventconsts.TicketMerged,
		EventTime:  time.Now().UTC(),
		Source:     eventconsts.SourceSupportTicket,
		Schema:     eventconsts.SchemaSupportTicket,
		ResourceID: target.ID.Hex(),
		Data:       eventbus.ProtoMarshaler{Message: targetPb},
		Metadata: map[string]any{
			"source_ticket_id": source.ID.Hex(),
			"target_ticket_id": target.ID.Hex(),
		},
	}
	_ = s.publisher.Publish(ctx, &evt)
}

func (s *TicketServiceServer) AssignTicket(ctx context.Context, req *apiv1.AssignTicketRequest) (*apiv1.AssignTicketResponse, error) {
	userID := s.getUserID(ctx)
	roles := s.getUserRoles(ctx)

	update := model.TicketUpdate{
		AssignedTo: &req.AssignTo,
	}

	ticket, err := s.repo.UpdateTicket(ctx, req.TicketId, update, userID, roles)
	if err != nil || ticket == nil {
		return nil, status.Error(codes.Internal, "failed to update ticket")
	}

	pb, _ := ticket.ToProto()
	s.publishTicketAssigned(ctx, ticket)

	return &apiv1.AssignTicketResponse{
		Ticket: pb,
	}, nil
}

func (s *TicketServiceServer) DistributeTickets(ctx context.Context, req *apiv1.DistributeTicketsRequest) (*apiv1.DistributeTicketsResponse, error) {
	updates := make(map[string]model.TicketUpdate, len(req.Assignments))
	for _, assignment := range req.Assignments {
		updates[assignment.TicketId] = model.TicketUpdate{
			AssignedTo: &assignment.AssignTo,
		}
	}

	tickets, err := s.repo.UpdateTickets(ctx, updates, s.getUserID(ctx), s.getUserRoles(ctx))
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to distribute tickets")
	}

	updatedMap := make(map[string]*model.Ticket, len(tickets))
	for _, t := range tickets {
		updatedMap[t.ID.Hex()] = t
		s.publishTicketAssigned(ctx, t)
	}

	return &apiv1.DistributeTicketsResponse{
		ModifiedCount: int32(len(tickets)),
	}, nil
}

func (s *TicketServiceServer) MergeTickets(ctx context.Context, req *apiv1.MergeTicketsRequest) (*apiv1.MergeTicketsResponse, error) {
	userID := s.getUserID(ctx)
	roles := s.getUserRoles(ctx)
	source, err := s.repo.GetTicket(ctx, req.SourceTicketId, userID, roles)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if source == nil {
		return nil, status.Error(codes.NotFound, "source ticket not found")
	}

	target, err := s.repo.GetTicket(ctx, req.TargetTicketId, userID, roles)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if target == nil {
		return nil, status.Error(codes.NotFound, "target ticket not found")
	}

	mergedStatus := int32(apiv1.TicketStatus_TICKET_STATUS_MERGED)
	updateSource := model.TicketUpdate{
		Status:     &mergedStatus,
		MergedInto: &target.ID,
	}

	source, err = s.repo.UpdateTicket(ctx, req.SourceTicketId, updateSource, userID, roles)
	if err != nil || source == nil {
		return nil, status.Error(codes.Internal, "failed to update source ticket status")
	}

	// Optionally add a comment to target ticket about the merge
	comment := &model.Comment{
		ID:      bson.NewObjectID(),
		Author:  SystemAuthor,
		Content: "Ticket " + source.ID.Hex() + " merged into this ticket.",
	}
	_ = s.repo.AddComment(ctx, req.TargetTicketId, comment, userID, roles)

	// Fetch updated target ticket to return
	target, _ = s.repo.GetTicket(ctx, target.ID.Hex(), userID, roles)

	s.publishTicketMerged(ctx, source, target)

	sourcePb, _ := source.ToProto()
	targetPb, _ := target.ToProto()

	return &apiv1.MergeTicketsResponse{
		SourceTicket: sourcePb,
		TargetTicket: targetPb,
	}, nil
}

func (s *TicketServiceServer) ListTickets(ctx context.Context, req *apiv1.ListTicketsRequest) (*apiv1.ListTicketsResponse, error) {
	limit, offset, pageNumber := getPaginationParams(req.Pagination)

	statuses := make([]int32, len(req.Statuses))
	for i, st := range req.Statuses {
		statuses[i] = int32(st)
	}

	priorities := make([]int32, len(req.Priority))
	for i, p := range req.Priority {
		priorities[i] = int32(p)
	}

	filter := model.TicketFilter{
		Statuses:            statuses,
		AssignedTo:          req.AssignedTo,
		TitleContains:       req.TitleContains,
		DescriptionContains: req.DescriptionContains,
		TicketTypes:         req.TicketType,
		Priorities:          priorities,
		CustomerIDs:         req.CustomerId,
		CreatedBy:           req.CreatedBy,
		MergedInto:          req.MergedInto,
		IncludeDeleted:      req.IncludeDeleted,
	}

	if tokenInfo, ok := middleware.TokenInfoFromContext(ctx); ok && tokenInfo != nil {
		filter.UserRoles = tokenInfo.Roles
	}
	if len(req.Metadata) > 0 {
		filter.Metadata = make(model.Metadata)
		for k, v := range req.Metadata {
			filter.Metadata[k] = v.AsInterface()
		}
	}

	var sorts []model.TicketSort
	for _, s := range req.Sort {
		order := 1 // asc
		if s.Order == apiv1.SortOrder_SORT_ORDER_DESC {
			order = -1
		}
		var field string
		switch s.Field {
		case apiv1.TicketSort_FIELD_PRIORITY:
			field = "priority"
		case apiv1.TicketSort_FIELD_CREATED_AT:
			field = "created_at"
		case apiv1.TicketSort_FIELD_UPDATED_AT:
			field = "updated_at"
		}
		if field != "" {
			sorts = append(sorts, model.TicketSort{Field: field, Order: order})
		}
	}

	tickets, total, err := s.repo.ListTickets(ctx, filter, sorts, s.getUserID(ctx), s.getUserRoles(ctx), limit, offset)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	pbTickets := make([]*apiv1.Ticket, 0, len(tickets))
	for _, t := range tickets {
		p, _ := t.ToProto()
		pbTickets = append(pbTickets, p)
	}

	return &apiv1.ListTicketsResponse{
		Tickets: pbTickets,
		Pagination: &apiv1.PageInfo{
			TotalSize:  total,
			PageNumber: pageNumber,
		},
	}, nil
}

func (s *TicketServiceServer) AddComment(ctx context.Context, req *apiv1.AddCommentRequest) (*apiv1.AddCommentResponse, error) {
	author, _ := middleware.UserFromContext(ctx)

	comment := &model.Comment{
		ID:      bson.NewObjectID(),
		Author:  author,
		Content: req.Content,
	}

	if err := s.repo.AddComment(ctx, req.TicketId, comment, s.getUserID(ctx), s.getUserRoles(ctx)); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	evt := event.Event{
		EventId:    uuid.NewString(),
		EventType:  eventconsts.TicketCommentAdded,
		EventTime:  time.Now().UTC(),
		Source:     eventconsts.SourceSupportTicket,
		Schema:     eventconsts.SchemaSupportTicket,
		ResourceID: req.TicketId,
		Data:       eventbus.ProtoMarshaler{Message: comment.ToProto()},
		Metadata: map[string]any{
			"ticket_id":  req.TicketId,
			"comment_id": comment.ID.Hex(),
			"author":     author,
		},
	}
	_ = s.publisher.Publish(ctx, &evt)

	return &apiv1.AddCommentResponse{
		Comment: comment.ToProto(),
	}, nil
}

func (s *TicketServiceServer) DeleteTicket(ctx context.Context, req *apiv1.DeleteTicketRequest) (*apiv1.DeleteTicketResponse, error) {
	userID := s.getUserID(ctx)
	roles := s.getUserRoles(ctx)
	if err := s.repo.DeleteTicket(ctx, req.TicketId, userID, roles); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	evt := event.Event{
		EventId:    uuid.NewString(),
		EventType:  eventconsts.TicketDeleted,
		EventTime:  time.Now().UTC(),
		Source:     eventconsts.SourceSupportTicket,
		Schema:     eventconsts.SchemaSupportTicket,
		ResourceID: req.TicketId,
		Metadata: map[string]any{
			"ticket_id": req.TicketId,
		},
	}
	_ = s.publisher.Publish(ctx, &evt)

	return &apiv1.DeleteTicketResponse{}, nil
}
