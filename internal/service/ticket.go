package service

import (
	"context"
	"encoding/json"
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

func NewTicketServiceServer(repo repository.TicketRepository, typeRepo repository.TicketTypeRepository, publisher event.Publisher) *TicketServiceServer {
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
		Source:     eventconsts.SourceTicket,
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
		Source:     eventconsts.SourceTicket,
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
		Source:     eventconsts.SourceTicket,
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
		Source:     eventconsts.SourceTicket,
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

	// Fetch current to check if it already requires approval
	current, err := s.repo.GetTicket(ctx, req.TicketId, userID, roles)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Override update logic if the ticket requires approval
	if current != nil && current.RequireApproval {
		// Map changes to metadata for approval
		metadata := make(map[string]any)
		if data, err := json.Marshal(update); err == nil {
			_ = json.Unmarshal(data, &metadata)
		}

		currentPb, _ := current.ToProto()
		s.publishEvent(ctx, eventconsts.TicketUpdatePendingApproval, current.ID.Hex(), eventbus.ProtoMarshaler{Message: currentPb}, metadata)

		// Execute update as status-only transition
		status := int32(apiv1.TicketStatus_TICKET_STATUS_PENDING_APPROVAL)
		update = model.TicketUpdate{Status: &status}
	}

	ticket, err := s.repo.UpdateTicket(ctx, req.TicketId, update, userID, roles)
	if err != nil || ticket == nil {
		if err == nil {
			return nil, status.Error(codes.NotFound, "ticket not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	pb, _ := ticket.ToProto()

	// Only publish TicketUpdated if it's not pending approval
	if update.Status != nil && *update.Status != int32(apiv1.TicketStatus_TICKET_STATUS_PENDING_APPROVAL) {
		s.publishEvent(ctx, eventconsts.TicketUpdated, ticket.ID.Hex(), eventbus.ProtoMarshaler{Message: pb}, nil)
	}

	return &apiv1.UpdateTicketResponse{
		Ticket: pb,
	}, nil
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
	if source.Status != int32(apiv1.TicketStatus_TICKET_STATUS_OPEN) && source.Status != int32(apiv1.TicketStatus_TICKET_STATUS_IN_PROGRESS) {
		return nil, status.Error(codes.FailedPrecondition, "source ticket must be in OPEN or IN_PROGRESS status to be merged")
	}

	target, err := s.repo.GetTicket(ctx, req.TargetTicketId, userID, roles)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if target == nil {
		return nil, status.Error(codes.NotFound, "target ticket not found")
	}
	if target.Status != int32(apiv1.TicketStatus_TICKET_STATUS_OPEN) && target.Status != int32(apiv1.TicketStatus_TICKET_STATUS_IN_PROGRESS) {
		return nil, status.Error(codes.FailedPrecondition, "target ticket must be in OPEN or IN_PROGRESS status to be merged")
	}

	statusToApply := int32(apiv1.TicketStatus_TICKET_STATUS_MERGED)
	if source.RequireApproval || target.RequireApproval {
		statusToApply = int32(apiv1.TicketStatus_TICKET_STATUS_PENDING_APPROVAL)

		// Both tickets go to PENDING_APPROVAL
		updateTarget := model.TicketUpdate{Status: &statusToApply}
		_, err = s.repo.UpdateTicket(ctx, req.TargetTicketId, updateTarget, userID, roles)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to update target ticket status")
		}

		// Map source ticket to metadata for approval
		sourcePb, _ := source.ToProto()
		metadata := map[string]any{
			"source_ticket_id": source.ID.Hex(),
			"target_ticket_id": target.ID.Hex(),
		}
		s.publishEvent(ctx, eventconsts.TicketMergePendingApproval, source.ID.Hex(), eventbus.ProtoMarshaler{Message: sourcePb}, metadata)
	}

	updateSource := model.TicketUpdate{
		Status: &statusToApply,
	}
	if statusToApply == int32(apiv1.TicketStatus_TICKET_STATUS_MERGED) {
		updateSource.MergedInto = &target.ID
	}

	source, err = s.repo.UpdateTicket(ctx, req.SourceTicketId, updateSource, userID, roles)
	if err != nil || source == nil {
		return nil, status.Error(codes.Internal, "failed to update source ticket status")
	}

	if statusToApply == int32(apiv1.TicketStatus_TICKET_STATUS_MERGED) {
		// Optionally add a comment to target ticket about the merge
		comment := &model.Comment{
			ID:      bson.NewObjectID(),
			Author:  SystemAuthor,
			Content: "Ticket " + source.ID.Hex() + " merged into this ticket.",
		}
		_ = s.repo.AddComment(ctx, req.TargetTicketId, comment, userID, roles)

		// Fetch updated target ticket
		target, _ = s.repo.GetTicket(ctx, target.ID.Hex(), userID, roles)

		s.publishTicketMerged(ctx, source, target, eventconsts.TicketMerged)
	}

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
		Source:     eventconsts.SourceTicket,
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
		Source:     eventconsts.SourceTicket,
		Schema:     eventconsts.SchemaSupportTicket,
		ResourceID: req.TicketId,
		Metadata: map[string]any{
			"ticket_id": req.TicketId,
		},
	}
	_ = s.publisher.Publish(ctx, &evt)

	return &apiv1.DeleteTicketResponse{}, nil
}

func (s *TicketServiceServer) publishEvent(ctx context.Context, eventType, resourceID string, data any, metadata map[string]any) {
	evt := event.Event{
		EventId:    uuid.NewString(),
		EventType:  eventType,
		EventTime:  time.Now().UTC(),
		Source:     eventconsts.SourceTicket,
		Schema:     eventconsts.SchemaSupportTicket,
		ResourceID: resourceID,
		Data:       data,
		Metadata:   metadata,
	}
	_ = s.publisher.Publish(ctx, &evt)
}

func (s *TicketServiceServer) publishTicketUpdated(ctx context.Context, ticket *model.Ticket) {
	pb, _ := ticket.ToProto()
	s.publishEvent(ctx, eventconsts.TicketUpdated, ticket.ID.Hex(), eventbus.ProtoMarshaler{Message: pb}, nil)
}

func (s *TicketServiceServer) publishTicketAssigned(ctx context.Context, ticket *model.Ticket) {
	ticketPb, _ := ticket.ToProto()
	metadata := map[string]any{"assigned_to": ticket.AssignedTo}
	s.publishEvent(ctx, eventconsts.TicketAssigned, ticket.ID.Hex(), eventbus.ProtoMarshaler{Message: ticketPb}, metadata)
}

func (s *TicketServiceServer) publishTicketMerged(ctx context.Context, source *model.Ticket, target *model.Ticket, eventType string) {
	targetPb, _ := target.ToProto()
	metadata := map[string]any{
		"source_ticket_id": source.ID.Hex(),
		"target_ticket_id": target.ID.Hex(),
	}
	s.publishEvent(ctx, eventType, target.ID.Hex(), eventbus.ProtoMarshaler{Message: targetPb}, metadata)
}

func (s *TicketServiceServer) RegisterHandlers(subscriber event.Subscriber) {
	subscriber.Subscribe(eventconsts.SchemaApproval, eventconsts.ApprovalDecided, s.HandleApprovalDecided)
}

func (s *TicketServiceServer) HandleApprovalDecided(ctx context.Context, evt *event.Event) error {
	var approval apiv1.ApprovalRequestData
	if err := eventbus.UnmarshalPayload(evt.Data, &approval); err != nil {
		return err
	}

	if approval.Origin != eventconsts.SourceTicket {
		return nil
	}

	userID := SystemAuthor
	roles := []string{"admin", "agent", "system"}

	switch approval.ActionType {
	case eventconsts.ActionTicketUpdate:
		if approval.Status == apiv1.ApprovalStatus_APPROVAL_STATUS_APPROVED {
			var update model.TicketUpdate
			metadata := make(map[string]any)
			for k, v := range approval.Metadata {
				metadata[k] = v.AsInterface()
			}
			data, _ := json.Marshal(metadata)
			_ = json.Unmarshal(data, &update)

			// Force status to IN_PROGRESS if no status is provided
			if update.Status == nil || *update.Status == int32(apiv1.TicketStatus_TICKET_STATUS_PENDING_APPROVAL) {
				statusInProgress := int32(apiv1.TicketStatus_TICKET_STATUS_IN_PROGRESS)
				update.Status = &statusInProgress
			}

			ticket, err := s.repo.UpdateTicket(ctx, approval.TicketId, update, userID, roles)
			if err == nil && ticket != nil {
				s.publishTicketUpdated(ctx, ticket)
			}
			return err
		} else if approval.Status == apiv1.ApprovalStatus_APPROVAL_STATUS_REJECTED {
			statusInProgress := int32(apiv1.TicketStatus_TICKET_STATUS_IN_PROGRESS)
			update := model.TicketUpdate{Status: &statusInProgress}
			_, err := s.repo.UpdateTicket(ctx, approval.TicketId, update, userID, roles)
			return err
		}

	case eventconsts.ActionTicketMerge:
		if approval.Status == apiv1.ApprovalStatus_APPROVAL_STATUS_APPROVED {
			sourceID := approval.Metadata["source_ticket_id"].GetStringValue()
			targetIDStr := approval.TargetId
			targetOID, _ := bson.ObjectIDFromHex(targetIDStr)

			statusMerged := int32(apiv1.TicketStatus_TICKET_STATUS_MERGED)
			statusInProgress := int32(apiv1.TicketStatus_TICKET_STATUS_IN_PROGRESS)

			// Update source status to MERGED
			updateSource := model.TicketUpdate{
				Status:     &statusMerged,
				MergedInto: &targetOID,
			}
			source, err := s.repo.UpdateTicket(ctx, sourceID, updateSource, userID, roles)
			if err == nil && source != nil {
				// Revert target status from PENDING_APPROVAL to IN_PROGRESS
				updateTarget := model.TicketUpdate{Status: &statusInProgress}
				_, _ = s.repo.UpdateTicket(ctx, targetIDStr, updateTarget, userID, roles)

				comment := &model.Comment{
					ID:        bson.NewObjectID(),
					Author:    SystemAuthor,
					Content:   "Ticket " + source.ID.Hex() + " merged into this ticket following approval.",
					CreatedAt: time.Now().UTC(),
				}
				_ = s.repo.AddComment(ctx, targetIDStr, comment, userID, roles)

				pb, _ := source.ToProto()
				metadata := map[string]any{"target_ticket_id": targetIDStr}
				s.publishEvent(ctx, eventconsts.TicketMerged, source.ID.Hex(), eventbus.ProtoMarshaler{Message: pb}, metadata)
			}
			return err
		} else if approval.Status == apiv1.ApprovalStatus_APPROVAL_STATUS_REJECTED {
			sourceID := approval.Metadata["source_ticket_id"].GetStringValue()
			targetID := approval.Metadata["target_ticket_id"].GetStringValue()

			statusInProgress := int32(apiv1.TicketStatus_TICKET_STATUS_IN_PROGRESS)
			update := model.TicketUpdate{
				Status: &statusInProgress,
			}
			// Revert both tickets' statuses
			_, _ = s.repo.UpdateTicket(ctx, sourceID, update, userID, roles)
			_, err := s.repo.UpdateTicket(ctx, targetID, update, userID, roles)
			return err
		}
	}

	return nil
}
