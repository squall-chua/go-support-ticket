package repository

import (
	"context"
	"time"

	apiv1 "github.com/squall-chua/go-support-ticket/api/v1"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type TicketRepository interface {
	CreateTicket(ctx context.Context, ticket *apiv1.Ticket) error
	GetTicket(ctx context.Context, id string) (*apiv1.Ticket, error)
	UpdateTicket(ctx context.Context, ticket *apiv1.Ticket) error
	ListTickets(ctx context.Context, status, assignedTo string, limit, offset int32) ([]*apiv1.Ticket, int32, error)
}

type mongoTicketRepo struct {
	col *mongo.Collection
}

func NewTicketRepository(connector DBConnector) TicketRepository {
	return &mongoTicketRepo{
		col: connector.Mongo().Collection("tickets"),
	}
}

func (r *mongoTicketRepo) CreateTicket(ctx context.Context, ticket *apiv1.Ticket) error {
	_, err := r.col.InsertOne(ctx, ticket)
	return err
}

func (r *mongoTicketRepo) GetTicket(ctx context.Context, id string) (*apiv1.Ticket, error) {
	var ticket apiv1.Ticket
	err := r.col.FindOne(ctx, bson.M{"id": id}).Decode(&ticket)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // Or custom not found error
		}
		return nil, err
	}
	return &ticket, nil
}

func (r *mongoTicketRepo) UpdateTicket(ctx context.Context, ticket *apiv1.Ticket) error {
	ticket.UpdatedAt = timestamppb.New(time.Now())
	_, err := r.col.ReplaceOne(ctx, bson.M{"id": ticket.Id}, ticket)
	return err
}

func (r *mongoTicketRepo) ListTickets(ctx context.Context, status, assignedTo string, limit, offset int32) ([]*apiv1.Ticket, int32, error) {
	filter := bson.M{}
	if status != "" {
		filter["status"] = status
	}
	if assignedTo != "" {
		filter["assigned_to"] = assignedTo
	}

	opts := options.Find().SetSkip(int64(offset))
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}

	cursor, err := r.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var tickets []*apiv1.Ticket
	for cursor.Next(ctx) {
		var t apiv1.Ticket
		if err := cursor.Decode(&t); err != nil {
			return nil, 0, err
		}
		tickets = append(tickets, &t)
	}

	total, err := r.col.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	return tickets, int32(total), nil
}
