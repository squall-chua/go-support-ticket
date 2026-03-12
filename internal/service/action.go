package service

import (
	"context"

	apiv1 "github.com/squall-chua/go-support-ticket/api/v1"
	"github.com/squall-chua/go-support-ticket/internal/repository"
	"github.com/squall-chua/go-support-ticket/pkg/event"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ActionServiceServer struct {
	apiv1.UnimplementedActionServiceServer
	repo      repository.ActionExecutionRepository
	schema    repository.ActionSchemaRepository
	approval  repository.ApprovalRepository
	publisher event.Publisher
}

func NewActionServiceServer(
	repo repository.ActionExecutionRepository,
	schema repository.ActionSchemaRepository,
	approval repository.ApprovalRepository,
	publisher event.Publisher,
) apiv1.ActionServiceServer {
	return &ActionServiceServer{
		repo:      repo,
		schema:    schema,
		approval:  approval,
		publisher: publisher,
	}
}

func (s *ActionServiceServer) ListActionSchemas(ctx context.Context, req *apiv1.ListActionSchemasRequest) (*apiv1.ListActionSchemasResponse, error) {
	schemas, err := s.schema.ListSchemas(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &apiv1.ListActionSchemasResponse{
		Schemas: schemas,
	}, nil
}

func (s *ActionServiceServer) SubmitAction(ctx context.Context, req *apiv1.SubmitActionRequest) (*apiv1.SubmitActionResponse, error) {
	schema, err := s.schema.GetSchema(ctx, req.ActionType)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if schema == nil {
		return nil, status.Error(codes.NotFound, "action schema not found")
	}

	execution := &apiv1.ActionExecution{
		TicketId:   req.TicketId,
		ActionType: req.ActionType,
		Parameters: req.Parameters,
		ExecutingUser: "sys", // placeholder
		Status:     "PENDING",
		CreatedAt:  timestamppb.Now(),
		UpdatedAt:  timestamppb.Now(),
	}

	if schema.RequireApproval {
		// Needs approval before execution. Wait for approval.
		execution.Status = "PENDING_APPROVAL"
	} else {
		// Execute immediately
		execution.Status = "IN_PROGRESS"
	}

	if err := s.repo.CreateExecution(ctx, execution); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if schema.RequireApproval {
		approval := &apiv1.ApprovalRequestData{
			TicketId:   execution.TicketId,
			Action: execution.ActionType,
			Status:     "PENDING",
			CreatedAt:  timestamppb.Now(),
			UpdatedAt:  timestamppb.Now(),
		}
		if err := s.approval.CreateApproval(ctx, approval); err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		payload := map[string]interface{}{"approval_id": approval.Id, "target_id": execution.Id}
		_ = s.publisher.Publish(ctx, "approval.requested", "system", payload)
	}

	return &apiv1.SubmitActionResponse{
		Execution: execution,
	}, nil
}

func (s *ActionServiceServer) ListActionExecutions(ctx context.Context, req *apiv1.ListActionExecutionsRequest) (*apiv1.ListActionExecutionsResponse, error) {
	// Not fully implemented but satisfy interface for now
	return &apiv1.ListActionExecutionsResponse{}, nil
}
