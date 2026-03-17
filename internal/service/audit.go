package service

import (
	"context"

	apiv1 "github.com/squall-chua/go-support-ticket/api/v1"
	"github.com/squall-chua/go-support-ticket/internal/repository"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AuditServiceServer struct {
	apiv1.UnimplementedAuditServiceServer
	repo repository.AuditRepository
}

func NewAuditServiceServer(repo repository.AuditRepository) apiv1.AuditServiceServer {
	return &AuditServiceServer{
		repo: repo,
	}
}

func (s *AuditServiceServer) ListAuditTrail(ctx context.Context, req *apiv1.ListAuditTrailRequest) (*apiv1.ListAuditTrailResponse, error) {
	limit, offset, pageNumber := getPaginationParams(req.Pagination)
	logs, total, err := s.repo.ListLogs(ctx, req.TicketId, "", limit, offset)
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
