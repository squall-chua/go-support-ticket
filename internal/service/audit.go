package service

import (
	"context"
	"strconv"

	apiv1 "github.com/squall-chua/go-support-ticket/api/v1"
	"github.com/squall-chua/go-support-ticket/internal/repository"
	"github.com/squall-chua/go-support-ticket/pkg/event"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AuditServiceServer struct {
	apiv1.UnimplementedAuditServiceServer
	repo      repository.AuditRepository
	publisher event.Publisher
}

func NewAuditServiceServer(repo repository.AuditRepository, publisher event.Publisher) apiv1.AuditServiceServer {
	return &AuditServiceServer{
		repo:      repo,
		publisher: publisher,
	}
}

func (s *AuditServiceServer) ListAuditTrail(ctx context.Context, req *apiv1.ListAuditTrailRequest) (*apiv1.ListAuditTrailResponse, error) {
	var limit, offset int32 = 100, 0
	if req.Pagination != nil {
		limit = req.Pagination.PageSize
		if parsedOffset, err := strconv.Atoi(req.Pagination.PageToken); err == nil {
			offset = int32(parsedOffset)
		}
	}
	logs, total, err := s.repo.ListLogs(ctx, req.TicketId, "", limit, offset)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &apiv1.ListAuditTrailResponse{
		Entries: logs,
		Pagination: &apiv1.PaginationResponse{
			TotalSize: total,
		},
	}, nil
}
