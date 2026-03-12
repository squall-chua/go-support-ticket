package repository

import (
	"context"

	apiv1 "github.com/squall-chua/go-support-ticket/api/v1"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ActionExecutionRepository interface {
	CreateExecution(ctx context.Context, exec *apiv1.ActionExecution) error
	GetExecution(ctx context.Context, id string) (*apiv1.ActionExecution, error)
	UpdateExecution(ctx context.Context, exec *apiv1.ActionExecution) error
	ListExecutions(ctx context.Context, ticketID string, limit, offset int32) ([]*apiv1.ActionExecution, int32, error)
}

type mongoActionExecutionRepo struct {
	col *mongo.Collection
}

func NewActionExecutionRepository(connector DBConnector) ActionExecutionRepository {
	return &mongoActionExecutionRepo{
		col: connector.Mongo().Collection("action_executions"),
	}
}

func (r *mongoActionExecutionRepo) CreateExecution(ctx context.Context, exec *apiv1.ActionExecution) error {
	_, err := r.col.InsertOne(ctx, exec)
	return err
}

func (r *mongoActionExecutionRepo) GetExecution(ctx context.Context, id string) (*apiv1.ActionExecution, error) {
	var exec apiv1.ActionExecution
	if err := r.col.FindOne(ctx, bson.M{"id": id}).Decode(&exec); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &exec, nil
}

func (r *mongoActionExecutionRepo) UpdateExecution(ctx context.Context, exec *apiv1.ActionExecution) error {
	_, err := r.col.ReplaceOne(ctx, bson.M{"id": exec.Id}, exec)
	return err
}

func (r *mongoActionExecutionRepo) ListExecutions(ctx context.Context, ticketID string, limit, offset int32) ([]*apiv1.ActionExecution, int32, error) {
	filter := bson.M{}
	if ticketID != "" {
		filter["ticket_id"] = ticketID
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

	var execs []*apiv1.ActionExecution
	if err = cursor.All(ctx, &execs); err != nil {
		return nil, 0, err
	}

	total, err := r.col.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	return execs, int32(total), nil
}
