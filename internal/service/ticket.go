package service

import (
	"context"
	"strconv"

	apiv1 "github.com/squall-chua/go-support-ticket/api/v1"
	"github.com/squall-chua/go-support-ticket/internal/repository"
	"github.com/squall-chua/go-support-ticket/pkg/event"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type TicketServiceServer struct {
	apiv1.UnimplementedTicketServiceServer
	repo      repository.TicketRepository
	publisher event.Publisher
}

func NewTicketServiceServer(repo repository.TicketRepository, publisher event.Publisher) apiv1.TicketServiceServer {
	return &TicketServiceServer{
		repo:      repo,
		publisher: publisher,
	}
}

func (s *TicketServiceServer) CreateTicketType(ctx context.Context, req *apiv1.CreateTicketTypeRequest) (*apiv1.CreateTicketTypeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (s *TicketServiceServer) ListTicketTypes(ctx context.Context, req *apiv1.ListTicketTypesRequest) (*apiv1.ListTicketTypesResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (s *TicketServiceServer) CreateTicket(ctx context.Context, req *apiv1.CreateTicketRequest) (*apiv1.CreateTicketResponse, error) {
	ticket := &apiv1.Ticket{
		Title:       req.Title,
		Description: req.Description,
		Priority:    req.Priority,
		Status:      "OPEN",
		TicketType:  req.TicketType,
		CustomerId:  req.CustomerId,
		CreatedAt:   timestamppb.Now(),
		UpdatedAt:   timestamppb.Now(),
	}

	if err := s.repo.CreateTicket(ctx, ticket); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	payload := map[string]interface{}{"ticket_id": ticket.Id, "status": ticket.Status}
	_ = s.publisher.Publish(ctx, "ticket.created", "system", payload)

	return &apiv1.CreateTicketResponse{
		Ticket: ticket,
	}, nil
}

func (s *TicketServiceServer) GetTicket(ctx context.Context, req *apiv1.GetTicketRequest) (*apiv1.GetTicketResponse, error) {
	ticket, err := s.repo.GetTicket(ctx, req.TicketId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if ticket == nil {
		return nil, status.Error(codes.NotFound, "ticket not found")
	}
	return &apiv1.GetTicketResponse{
		Ticket: ticket,
	}, nil
}

func (s *TicketServiceServer) UpdateTicket(ctx context.Context, req *apiv1.UpdateTicketRequest) (*apiv1.UpdateTicketResponse, error) {
	update := &apiv1.Ticket{
		Id:          req.TicketId,
		Title:       req.Title,
		Description: req.Description,
		Priority:    req.Priority,
		UpdatedAt:   timestamppb.Now(),
	}

	if err := s.repo.UpdateTicket(ctx, update); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	ticket, _ := s.repo.GetTicket(ctx, req.TicketId)

	return &apiv1.UpdateTicketResponse{
		Ticket: ticket,
	}, nil
}

func (s *TicketServiceServer) TransitionStatus(ctx context.Context, req *apiv1.TransitionStatusRequest) (*apiv1.TransitionStatusResponse, error) {
	update := &apiv1.Ticket{
		Id:        req.TicketId,
		Status:    req.NewStatus,
		UpdatedAt: timestamppb.Now(),
	}

	if err := s.repo.UpdateTicket(ctx, update); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	ticket, _ := s.repo.GetTicket(ctx, req.TicketId)

	payload := map[string]interface{}{"ticket_id": ticket.Id, "new_status": ticket.Status}
	_ = s.publisher.Publish(ctx, "ticket.status.updated", "system", payload)

	return &apiv1.TransitionStatusResponse{
		Ticket: ticket,
	}, nil
}

func (s *TicketServiceServer) AssignTicket(ctx context.Context, req *apiv1.AssignTicketRequest) (*apiv1.AssignTicketResponse, error) {
	update := &apiv1.Ticket{
		Id:         req.TicketId,
		AssignedTo: req.AssignTo,
		UpdatedAt:  timestamppb.Now(),
	}

	if err := s.repo.UpdateTicket(ctx, update); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	ticket, _ := s.repo.GetTicket(ctx, req.TicketId)

	payload := map[string]interface{}{"ticket_id": ticket.Id, "assignee_id": ticket.AssignedTo}
	_ = s.publisher.Publish(ctx, "ticket.assigned", "system", payload)

	return &apiv1.AssignTicketResponse{
		Ticket: ticket,
	}, nil
}

func (s *TicketServiceServer) DistributeTickets(ctx context.Context, req *apiv1.DistributeTicketsRequest) (*apiv1.DistributeTicketsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (s *TicketServiceServer) MergeTickets(ctx context.Context, req *apiv1.MergeTicketsRequest) (*apiv1.MergeTicketsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (s *TicketServiceServer) ListTickets(ctx context.Context, req *apiv1.ListTicketsRequest) (*apiv1.ListTicketsResponse, error) {
	var limit, offset int32 = 100, 0
	if req.Pagination != nil {
		limit = req.Pagination.PageSize
		if parsedOffset, err := strconv.Atoi(req.Pagination.PageToken); err == nil {
			offset = int32(parsedOffset)
		}
	}
	tickets, total, err := s.repo.ListTickets(ctx, req.Status, req.AssignedTo, limit, offset)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &apiv1.ListTicketsResponse{
		Tickets: tickets,
		Pagination: &apiv1.PaginationResponse{
			TotalSize: total,
		},
	}, nil
}
