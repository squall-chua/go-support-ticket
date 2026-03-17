package repository

import (
	"context"
	"time"

	"github.com/squall-chua/gmqb"
	"github.com/squall-chua/go-support-ticket/internal/model"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type ActionExecutionRepo struct {
	coll *gmqb.Collection[model.ActionExecution]
}

func NewActionExecutionRepo(col *mongo.Collection) *ActionExecutionRepo {
	return &ActionExecutionRepo{coll: gmqb.Wrap[model.ActionExecution](col)}
}

func (r *ActionExecutionRepo) CreateExecution(ctx context.Context, e *model.ActionExecution) error {
	now := time.Now().UTC()
	e.CreatedAt = now
	e.UpdatedAt = now
	_, err := r.coll.InsertOne(ctx, e)
	return err
}

func (r *ActionExecutionRepo) GetExecution(ctx context.Context, id string) (*model.ActionExecution, error) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	return r.coll.FindOne(ctx, gmqb.Eq("_id", oid))
}

func (r *ActionExecutionRepo) UpdateExecution(ctx context.Context, id string, update model.ActionExecutionUpdate) error {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	f := gmqb.Field[model.ActionExecution]
	u := gmqb.NewUpdate()

	if update.Status != nil {
		u.Set(f("Status"), *update.Status)
	}
	if update.Result != nil {
		u.Set(f("Result"), update.Result)
	}
	if update.CompletedAt != nil {
		// This field is also in Result, but we might keep it top-level for some reason? 
		// Actually I removed it from ActionExecution struct in model.
		// So CompletedAt should be part of Result.
		// I'll skip setting it top-level since it's gone from the struct.
	}
	u.Set(f("UpdatedAt"), time.Now().UTC())

	_, err = r.coll.UpdateOne(ctx, gmqb.Eq("_id", oid), u)
	return err
}


func (r *ActionExecutionRepo) ListExecutions(ctx context.Context, filter model.ActionExecutionFilter, limit, offset int32) ([]*model.ActionExecution, int32, error) {
	f := gmqb.Field[model.ActionExecution]
	mqbFilter := gmqb.NewFilter()

	if len(filter.IDs) > 0 {
		mqbFilter.In("_id", filter.IDs)
	}
	if len(filter.TicketIDs) > 0 {
		mqbFilter.In(f("TicketID"), filter.TicketIDs)
	}
	if len(filter.ActionTypes) > 0 {
		mqbFilter.In(f("ActionType"), filter.ActionTypes)
	}
	if len(filter.Statuses) > 0 {
		mqbFilter.In(f("Status"), filter.Statuses)
	}
	if len(filter.ExecutingUsers) > 0 {
		mqbFilter.In(f("ExecutingUser"), filter.ExecutingUsers)
	}
	if filter.StartTime != nil {
		mqbFilter.Gte(f("CreatedAt"), *filter.StartTime)
	}
	if filter.EndTime != nil {
		mqbFilter.Lte(f("CreatedAt"), *filter.EndTime)
	}

	return listPaginated(ctx, r.coll, mqbFilter, gmqb.Desc(f("CreatedAt")), limit, offset)
}
