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

type ApprovalServiceServer struct {
	apiv1.UnimplementedApprovalServiceServer
	repo      repository.ApprovalRepository
	publisher event.Publisher
}

func NewApprovalServiceServer(
	repo repository.ApprovalRepository,
	publisher event.Publisher,
) *ApprovalServiceServer {
	return &ApprovalServiceServer{
		repo:      repo,
		publisher: publisher,
	}
}

func (s *ApprovalServiceServer) CreateApproval(ctx context.Context, req *apiv1.CreateApprovalRequest) (*apiv1.CreateApprovalResponse, error) {
	userID, _ := middleware.UserFromContext(ctx)
	approval := &model.Approval{
		ID:          bson.NewObjectID(),
		TicketID:    req.TicketId,
		Action:      req.Action,
		Requester:   userID,
		Status:      int32(apiv1.ApprovalStatus_APPROVAL_STATUS_PENDING),
		ExecutionID: req.ExecutionId,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if err := s.repo.CreateApproval(ctx, approval); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	evt := event.Event{
		EventId:    uuid.NewString(),
		EventType:  eventconsts.ApprovalRequested,
		EventTime:  time.Now().UTC(),
		Source:     eventconsts.SourceSupportTicket,
		Schema:     eventconsts.SchemaApproval,
		ResourceID: req.TicketId,
		Data:       eventbus.ProtoMarshaler{Message: approval.ToProto()},
		Metadata:   map[string]any{"target_id": req.ExecutionId},
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

	newDecisions := append(approval.Decisions, model.Decision{
		Approver:  approver,
		Approved:  req.Approve,
		Reason:    req.Reason,
		DecidedAt: time.Now().UTC(),
	})

	var newStatus int32
	if req.Approve {
		// Simplified logic: first approval wins if required is 0 (default handled)
		newStatus = int32(apiv1.ApprovalStatus_APPROVAL_STATUS_APPROVED)
	} else {
		newStatus = int32(apiv1.ApprovalStatus_APPROVAL_STATUS_REJECTED)
	}

	update := model.ApprovalUpdate{
		Status:    &newStatus,
		Decisions: &newDecisions,
		UpdatedAt: time.Now().UTC(),
	}

	if err := s.repo.UpdateApproval(ctx, req.ApprovalRequestId, update); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Update local object for subsequent logic and response
	approval.Status = newStatus
	approval.Decisions = newDecisions
	approval.UpdatedAt = update.UpdatedAt

	// If approved/rejected, publish final decision event
	decisionEvt := event.Event{
		EventId:    uuid.NewString(),
		EventType:  eventconsts.ApprovalDecided,
		EventTime:  time.Now().UTC(),
		Source:     eventconsts.SourceSupportTicket,
		Schema:     eventconsts.SchemaApproval,
		ResourceID: approval.TicketID,
		Data:       eventbus.ProtoMarshaler{Message: approval.ToProto()},
	}
	_ = s.publisher.Publish(ctx, &decisionEvt)

	return &apiv1.DecideApprovalResponse{
		Request: approval.ToProto(),
	}, nil
}

func (s *ApprovalServiceServer) ListApprovals(ctx context.Context, req *apiv1.ListApprovalsRequest) (*apiv1.ListApprovalsResponse, error) {
	var limit, offset, pageNumber int32 = 100, 0, 1
	if req.Pagination != nil {
		limit = req.Pagination.PageSize
		pageNumber = req.Pagination.PageNumber
		if pageNumber > 1 {
			offset = (pageNumber - 1) * limit
		} else {
			pageNumber = 1
		}
	}

	filter := model.ApprovalFilter{
		TicketIDs:         req.TicketIds,
		Actions:           req.Actions,
		Requesters:        req.Requesters,
		ExecutionIDs:      req.ExecutionIds,
		RequiredApprovals: req.RequiredApprovals,
		Approvers:         req.Approvers,
	}

	for _, st := range req.Statuses {
		filter.Statuses = append(filter.Statuses, int32(st))
	}

	if req.TimeRange != nil {
		if req.TimeRange.StartTime != nil {
			st := req.TimeRange.StartTime.AsTime()
			filter.StartTime = &st
		}
		if req.TimeRange.EndTime != nil {
			et := req.TimeRange.EndTime.AsTime()
			filter.EndTime = &et
		}
	}

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

func (s *ApprovalServiceServer) RegisterHandlers(subscriber event.Subscriber) {
	subscriber.Subscribe(eventconsts.SchemaAction, eventconsts.ActionExecutionPending, s.HandleActionPendingApproval)
}

func (s *ApprovalServiceServer) HandleActionPendingApproval(ctx context.Context, evt *event.Event) error {
	var execution apiv1.ActionExecution
	if err := eventbus.UnmarshalPayload(evt.Data, &execution); err != nil {
		return err
	}

	// Set user in context for CreateApproval
	ctx = middleware.WithUser(ctx, execution.ExecutingUser)

	_, err := s.CreateApproval(ctx, &apiv1.CreateApprovalRequest{
		TicketId:    execution.TicketId,
		Action:      execution.ActionType,
		ExecutionId: execution.Id,
	})

	return err
}
