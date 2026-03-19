package service

import (
	"context"
	"slices"
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
	"google.golang.org/protobuf/types/known/structpb"
)

type ApprovalServiceServer struct {
	apiv1.UnimplementedApprovalServiceServer
	repo       repository.ApprovalRepository
	configRepo repository.ApprovalConfigRepository
	publisher  event.Publisher
}

func NewApprovalServiceServer(
	repo repository.ApprovalRepository,
	configRepo repository.ApprovalConfigRepository,
	publisher event.Publisher,
) *ApprovalServiceServer {
	return &ApprovalServiceServer{
		repo:       repo,
		configRepo: configRepo,
		publisher:  publisher,
	}
}

func (s *ApprovalServiceServer) CreateApproval(ctx context.Context, req *apiv1.CreateApprovalRequest) (*apiv1.CreateApprovalResponse, error) {
	userID, _ := middleware.UserFromContext(ctx)

	if req.TicketType == "" && req.ActionType == "" {
		return nil, status.Error(codes.InvalidArgument, "either ticket_type or action_type is required")
	}
	ticketType := req.TicketType
	actionType := req.ActionType

	requiredApprovals := int32(1)
	var eligibleRoles []string
	config, err := s.configRepo.GetConfig(ctx, ticketType, actionType)
	if err == nil && config != nil {
		requiredApprovals = config.RequiredApprovals
		eligibleRoles = config.EligibleRoles
	}

	approval := &model.Approval{
		ID:                bson.NewObjectID(),
		TicketID:          req.TicketId,
		TicketType:        ticketType,
		ActionType:        actionType,
		TargetID:          req.TargetId,
		Requester:         userID,
		Origin:            req.Origin,
		Status:            int32(apiv1.ApprovalStatus_APPROVAL_STATUS_PENDING),
		RequiredApprovals: requiredApprovals,
		EligibleRoles:     eligibleRoles,
		Metadata:          make(map[string]any),
	}

	if req.Metadata != nil {
		for k, v := range req.Metadata {
			approval.Metadata[k] = v.AsInterface()
		}
	}

	if err := s.repo.CreateApproval(ctx, approval); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	evt := event.Event{
		EventId:    uuid.NewString(),
		EventType:  eventconsts.ApprovalRequested,
		EventTime:  time.Now().UTC(),
		User:       userID,
		Source:     eventconsts.SourceApproval,
		Schema:     eventconsts.SchemaApproval,
		ResourceID: approval.ID.Hex(),
		Data:       eventbus.ProtoMarshaler{Message: approval.ToProto()},
	}
	_ = s.publisher.Publish(ctx, &evt)

	return &apiv1.CreateApprovalResponse{
		Request: approval.ToProto(),
	}, nil
}

func (s *ApprovalServiceServer) DecideApproval(ctx context.Context, req *apiv1.DecideApprovalRequest) (*apiv1.DecideApprovalResponse, error) {
	approval, err := s.repo.GetApproval(ctx, req.ApprovalRequestId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if approval == nil {
		return nil, status.Error(codes.NotFound, "approval not found")
	}

	if approval.Status != int32(apiv1.ApprovalStatus_APPROVAL_STATUS_PENDING) {
		return nil, status.Error(codes.FailedPrecondition, "approval is not pending")
	}

	approver, _ := middleware.UserFromContext(ctx)
	if approver == "" {
		return nil, status.Error(codes.Unauthenticated, "approver identity not found")
	}

	if len(approval.EligibleRoles) > 0 {
		tokenInfo, ok := middleware.TokenInfoFromContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "token info not found")
		}
		isEligible := slices.ContainsFunc(approval.EligibleRoles, func(role string) bool {
			return slices.Contains(tokenInfo.Roles, role)
		})
		if !isEligible {
			return nil, status.Error(codes.PermissionDenied, "user is not eligible to decide on this approval")
		}
	}

	decision := model.Decision{
		Approver:  approver,
		Approved:  req.Approve,
		Reason:    req.Reason,
		DecidedAt: time.Now().UTC(),
	}
	newDecisions := append(approval.Decisions, decision)

	var newStatus int32
	if req.Approve {
		approvedCount := int32(0)
		for _, d := range newDecisions {
			if d.Approved {
				approvedCount++
			}
		}
		if approvedCount >= approval.RequiredApprovals {
			newStatus = int32(apiv1.ApprovalStatus_APPROVAL_STATUS_APPROVED)
		} else {
			newStatus = int32(apiv1.ApprovalStatus_APPROVAL_STATUS_PENDING)
		}
	} else {
		newStatus = int32(apiv1.ApprovalStatus_APPROVAL_STATUS_REJECTED)
	}

	update := model.ApprovalUpdate{
		Status:   &newStatus,
		Decision: &decision, // Send only the new decision for atomic push
	}

	if err := s.repo.UpdateApproval(ctx, req.ApprovalRequestId, update); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Update local object for subsequent logic and response
	approval.Status = newStatus
	approval.Decisions = newDecisions

	// If approved/rejected (final state), publish final decision event
	// If still pending, publish pending event
	var eventType string
	if newStatus == int32(apiv1.ApprovalStatus_APPROVAL_STATUS_APPROVED) || newStatus == int32(apiv1.ApprovalStatus_APPROVAL_STATUS_REJECTED) {
		eventType = eventconsts.ApprovalDecided
	} else if newStatus == int32(apiv1.ApprovalStatus_APPROVAL_STATUS_PENDING) {
		eventType = eventconsts.ApprovalPending
	}

	if eventType != "" {
		decisionEvt := event.Event{
			EventId:    uuid.NewString(),
			EventType:  eventType,
			EventTime:  time.Now().UTC(),
			User:       approver,
			Source:     eventconsts.SourceApproval,
			Schema:     eventconsts.SchemaApproval,
			ResourceID: approval.ID.Hex(),
			Data:       eventbus.ProtoMarshaler{Message: approval.ToProto()},
			Metadata: map[string]any{
				"decision": eventbus.ProtoMarshaler{Message: decision.ToProto()},
			},
		}
		_ = s.publisher.Publish(ctx, &decisionEvt)
	}

	return &apiv1.DecideApprovalResponse{
		Request: approval.ToProto(),
	}, nil
}

func (s *ApprovalServiceServer) ListApprovals(ctx context.Context, req *apiv1.ListApprovalsRequest) (*apiv1.ListApprovalsResponse, error) {
	limit, offset, pageNumber := getPaginationParams(req.Pagination)

	filter := model.ApprovalFilter{
		TicketIDs:         req.TicketIds,
		TicketTypes:       req.TicketTypes,
		ActionTypes:       req.ActionTypes,
		Requesters:        req.Requesters,
		TargetIDs:         req.TargetIds,
		RequiredApprovals: req.RequiredApprovals,
		Approvers:         req.Approvers,
		Origins:           req.Origins,
	}

	for _, st := range req.Statuses {
		filter.Statuses = append(filter.Statuses, int32(st))
	}

	filter.StartTime, filter.EndTime = getTimeRange(req.TimeRange)

	approvals, total, err := s.repo.ListApprovals(ctx, filter, limit, offset)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	pbApprovals := make([]*apiv1.ApprovalRequestData, 0, len(approvals))
	for _, a := range approvals {
		pbApprovals = append(pbApprovals, a.ToProto())
	}

	return &apiv1.ListApprovalsResponse{
		Requests: pbApprovals,
		Pagination: &apiv1.PageInfo{
			TotalSize:  total,
			PageNumber: pageNumber,
		},
	}, nil
}

func (s *ApprovalServiceServer) CreateApprovalConfig(ctx context.Context, req *apiv1.CreateApprovalConfigRequest) (*apiv1.CreateApprovalConfigResponse, error) {
	if req.Config == nil {
		return nil, status.Error(codes.InvalidArgument, "config is required")
	}
	config := model.ApprovalConfigFromProto(req.Config)
	if config.ApprovalConfigID.IsZero() {
		config.ApprovalConfigID = bson.NewObjectID()
	}

	if err := s.configRepo.CreateConfig(ctx, config); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	userID, _ := middleware.UserFromContext(ctx)
	evt := event.Event{
		EventId:    uuid.NewString(),
		EventType:  eventconsts.ApprovalConfigCreated,
		EventTime:  time.Now().UTC(),
		User:       userID,
		Source:     eventconsts.SourceApproval,
		Schema:     eventconsts.SchemaApprovalConfig,
		ResourceID: config.ApprovalConfigID.Hex(),
		Data:       eventbus.ProtoMarshaler{Message: config.ToProto()},
	}
	_ = s.publisher.Publish(ctx, &evt)

	return &apiv1.CreateApprovalConfigResponse{Config: config.ToProto()}, nil
}

func (s *ApprovalServiceServer) GetApprovalConfig(ctx context.Context, req *apiv1.GetApprovalConfigRequest) (*apiv1.ApprovalConfig, error) {
	ticketType := req.TicketType
	actionType := req.ActionType
	if ticketType == "" && actionType == "" {
		return nil, status.Error(codes.InvalidArgument, "either ticket_type or action_type is required")
	}
	config, err := s.configRepo.GetConfig(ctx, ticketType, actionType)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if config == nil {
		return nil, status.Error(codes.NotFound, "approval config not found")
	}
	return config.ToProto(), nil
}

func (s *ApprovalServiceServer) UpdateApprovalConfig(ctx context.Context, req *apiv1.UpdateApprovalConfigRequest) (*apiv1.UpdateApprovalConfigResponse, error) {
	ticketType := req.TicketType
	actionType := req.ActionType
	if ticketType == "" && actionType == "" {
		return nil, status.Error(codes.InvalidArgument, "either ticket_type or action_type is required")
	}

	update := model.ApprovalConfigUpdate{
		RequiredApprovals: &req.RequiredApprovals,
		EligibleRoles:     req.EligibleRoles,
	}

	before, err := s.configRepo.UpdateConfig(ctx, ticketType, actionType, update, false)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if before == nil {
		return nil, status.Error(codes.NotFound, "approval config not found")
	}

	updated, err := s.configRepo.GetConfig(ctx, ticketType, actionType)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	userID, _ := middleware.UserFromContext(ctx)
	evt := event.Event{
		EventId:    uuid.NewString(),
		EventType:  eventconsts.ApprovalConfigUpdated,
		EventTime:  time.Now().UTC(),
		User:       userID,
		Source:     eventconsts.SourceApproval,
		Schema:     eventconsts.SchemaApprovalConfig,
		ResourceID: updated.ApprovalConfigID.Hex(),
		Data:       eventbus.ProtoMarshaler{Message: updated.ToProto()},
		Metadata: map[string]any{
			"before": eventbus.ProtoMarshaler{Message: before.ToProto()},
			"update": eventbus.ProtoMarshaler{Message: req},
		},
	}
	_ = s.publisher.Publish(ctx, &evt)

	return &apiv1.UpdateApprovalConfigResponse{Config: updated.ToProto()}, nil
}

func (s *ApprovalServiceServer) DeleteApprovalConfig(ctx context.Context, req *apiv1.DeleteApprovalConfigRequest) (*apiv1.DeleteApprovalConfigResponse, error) {
	ticketType := req.TicketType
	actionType := req.ActionType
	if ticketType == "" && actionType == "" {
		return nil, status.Error(codes.InvalidArgument, "either ticket_type or action_type is required")
	}
	config, err := s.configRepo.DeleteConfig(ctx, ticketType, actionType)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if config != nil {
		userID, _ := middleware.UserFromContext(ctx)
		evt := event.Event{
			EventId:    uuid.NewString(),
			EventType:  eventconsts.ApprovalConfigDeleted,
			EventTime:  time.Now().UTC(),
			User:       userID,
			Source:     eventconsts.SourceApproval,
			Schema:     eventconsts.SchemaApprovalConfig,
			ResourceID: config.ApprovalConfigID.Hex(),
			Data:       eventbus.ProtoMarshaler{Message: config.ToProto()},
		}
		_ = s.publisher.Publish(ctx, &evt)
	}

	return &apiv1.DeleteApprovalConfigResponse{}, nil
}

func (s *ApprovalServiceServer) ListApprovalConfigs(ctx context.Context, req *apiv1.ListApprovalConfigsRequest) (*apiv1.ListApprovalConfigsResponse, error) {
	limit, offset, pageNumber := getPaginationParams(req.Pagination)

	filter := model.ApprovalConfigFilter{
		ActionTypes:    req.ActionTypes,
		TicketTypes:    req.TicketTypes,
		EligibleRoles:  req.EligibleRoles,
		IncludeDeleted: req.IncludeDeleted,
	}

	if req.RequiredApprovals != 0 {
		ra := req.RequiredApprovals
		filter.RequiredApprovals = &ra
	}

	filter.StartTime, filter.EndTime = getTimeRange(req.TimeRange)

	configs, total, err := s.configRepo.ListConfigs(ctx, filter, limit, offset)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	pbConfigs := make([]*apiv1.ApprovalConfig, 0, len(configs))
	for _, c := range configs {
		pbConfigs = append(pbConfigs, c.ToProto())
	}

	return &apiv1.ListApprovalConfigsResponse{
		Configs: pbConfigs,
		Pagination: &apiv1.PageInfo{
			TotalSize:  total,
			PageNumber: pageNumber,
		},
	}, nil
}

func (s *ApprovalServiceServer) RegisterHandlers(subscriber event.Subscriber) {
	subscriber.Subscribe(eventconsts.SchemaAction, eventconsts.ActionExecutionPending, s.HandleActionPendingApproval)
	subscriber.Subscribe(eventconsts.SchemaSupportTicket, eventconsts.TicketUpdatePendingApproval, s.HandleTicketUpdatePendingApproval)
	subscriber.Subscribe(eventconsts.SchemaSupportTicket, eventconsts.TicketMergePendingApproval, s.HandleTicketMergePendingApproval)
}

func (s *ApprovalServiceServer) HandleActionPendingApproval(ctx context.Context, evt *event.Event) error {
	var execution apiv1.ActionExecution
	if err := eventbus.UnmarshalPayload(evt.Data, &execution); err != nil {
		return err
	}

	// Set user in context for CreateApproval
	ctx = middleware.WithUser(ctx, execution.ExecutingUser)

	metadata := make(map[string]*structpb.Value)
	if evt.Metadata != nil {
		for k, v := range evt.Metadata {
			if val, err := structpb.NewValue(v); err == nil {
				metadata[k] = val
			}
		}
	}

	_, err := s.CreateApproval(ctx, &apiv1.CreateApprovalRequest{
		TicketId:   execution.TicketId,
		TargetId:   execution.Id,
		TicketType: execution.TicketType,
		ActionType: execution.ActionType,
		Origin:     evt.Source,
		Metadata:   metadata,
	})
	return err
}

func (s *ApprovalServiceServer) HandleTicketUpdatePendingApproval(ctx context.Context, evt *event.Event) error {
	var ticket apiv1.Ticket
	if err := eventbus.UnmarshalPayload(evt.Data, &ticket); err != nil {
		return err
	}

	// Set user in context for CreateApproval
	ctx = middleware.WithUser(ctx, ticket.AssignedTo)

	metadata := make(map[string]*structpb.Value)
	if evt.Metadata != nil {
		for k, v := range evt.Metadata {
			if val, err := structpb.NewValue(v); err == nil {
				metadata[k] = val
			}
		}
	}

	_, err := s.CreateApproval(ctx, &apiv1.CreateApprovalRequest{
		TicketId:   ticket.Id,
		TicketType: ticket.TicketType,
		ActionType: eventconsts.ActionTicketUpdate,
		Origin:     evt.Source,
		Metadata:   metadata,
	})
	return err
}

func (s *ApprovalServiceServer) HandleTicketMergePendingApproval(ctx context.Context, evt *event.Event) error {
	var ticket apiv1.Ticket
	if err := eventbus.UnmarshalPayload(evt.Data, &ticket); err != nil {
		return err
	}

	// Set user in context for CreateApproval
	ctx = middleware.WithUser(ctx, ticket.AssignedTo)

	targetID, _ := evt.Metadata["target_ticket_id"].(string)

	metadata := make(map[string]*structpb.Value)
	if evt.Metadata != nil {
		for k, v := range evt.Metadata {
			if val, err := structpb.NewValue(v); err == nil {
				metadata[k] = val
			}
		}
	}

	_, err := s.CreateApproval(ctx, &apiv1.CreateApprovalRequest{
		TicketId:   ticket.Id,
		TargetId:   targetID, // Use target_id as a proxy for target ticket id during merge
		TicketType: ticket.TicketType,
		ActionType: eventconsts.ActionTicketMerge,
		Origin:     evt.Source,
		Metadata:   metadata,
	})
	return err
}
