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

type ApprovalServiceServer struct {
	apiv1.UnimplementedApprovalServiceServer
	repo      repository.ApprovalRepository
	publisher event.Publisher
}

func NewApprovalServiceServer(repo repository.ApprovalRepository, publisher event.Publisher) apiv1.ApprovalServiceServer {
	return &ApprovalServiceServer{
		repo:      repo,
		publisher: publisher,
	}
}

func (s *ApprovalServiceServer) DecideApproval(ctx context.Context, req *apiv1.DecideApprovalRequest) (*apiv1.DecideApprovalResponse, error) {
	approval, err := s.repo.GetApproval(ctx, req.ApprovalRequestId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if approval == nil {
		return nil, status.Error(codes.NotFound, "approval not found")
	}

	if approval.Status != "PENDING" {
		return nil, status.Error(codes.FailedPrecondition, "approval is not pending")
	}

	var approvalStatus string
	if req.Approve {
		approvalStatus = "APPROVED"
	} else {
		approvalStatus = "REJECTED"
	}

	approval.Status = approvalStatus
	approval.UpdatedAt = timestamppb.Now()

	if err := s.repo.UpdateApproval(ctx, approval); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	payload := map[string]interface{}{
		"approval_id": approval.Id,
		"status":      approvalStatus,
	}
	_ = s.publisher.Publish(ctx, "approval.processed", "system", payload)

	return &apiv1.DecideApprovalResponse{
		Request: approval,
	}, nil
}

func (s *ApprovalServiceServer) ListPendingApprovals(ctx context.Context, req *apiv1.ListPendingApprovalsRequest) (*apiv1.ListPendingApprovalsResponse, error) {
	return &apiv1.ListPendingApprovalsResponse{}, nil
}
