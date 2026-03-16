package repository

import (
	"context"

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
	if update.ResultMetadata != nil {
		u.Set(f("ResultMetadata"), *update.ResultMetadata)
	}
	if update.Error != nil {
		u.Set(f("Error"), *update.Error)
	}
	if update.Logs != nil {
		u.Set(f("Logs"), *update.Logs)
	}
	if update.CompletedAt != nil {
		u.Set(f("CompletedAt"), *update.CompletedAt)
	}
	u.Set(f("UpdatedAt"), update.UpdatedAt)

	_, err = r.coll.UpdateOne(ctx, gmqb.Eq("_id", oid), u)
	return err
}


func (r *ActionExecutionRepo) ListExecutions(ctx context.Context, ticketID, actionType string, limit, offset int32) ([]*model.ActionExecution, int32, error) {
	f := gmqb.Field[model.ActionExecution]
	filter := gmqb.NewFilter()

	if ticketID != "" {
		filter.Eq(f("TicketID"), ticketID)
	}
	if actionType != "" {
		filter.Eq(f("ActionType"), actionType)
	}

	return listPaginated(ctx, r.coll, filter, gmqb.Desc(f("CreatedAt")), limit, offset)
}
