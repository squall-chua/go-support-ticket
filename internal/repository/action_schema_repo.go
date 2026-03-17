package repository

import (
	"context"
	"time"

	"github.com/squall-chua/gmqb"
	"github.com/squall-chua/go-support-ticket/internal/model"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type ActionSchemaRepo struct {
	coll *gmqb.Collection[model.ActionSchema]
}

func NewActionSchemaRepo(col *mongo.Collection) *ActionSchemaRepo {
	return &ActionSchemaRepo{coll: gmqb.Wrap[model.ActionSchema](col)}
}

func (r *ActionSchemaRepo) CreateSchema(ctx context.Context, schema *model.ActionSchema) error {
	now := time.Now().UTC()
	schema.CreatedAt = now
	schema.UpdatedAt = now
	_, err := r.coll.InsertOne(ctx, schema)
	return err
}

func (r *ActionSchemaRepo) GetSchema(ctx context.Context, actionType string) (*model.ActionSchema, error) {
	f := gmqb.Field[model.ActionSchema]
	return r.coll.FindOne(ctx, gmqb.And(
		gmqb.Eq(f("ActionType"), actionType),
		gmqb.Eq(f("DeletedAt"), nil),
	))
}

func (r *ActionSchemaRepo) ListSchemas(ctx context.Context, filter model.ActionSchemaFilter, limit, offset int32) ([]*model.ActionSchema, int32, error) {
	f := gmqb.Field[model.ActionSchema]
	q := gmqb.NewFilter()

	if !filter.IncludeDeleted {
		q.Eq(f("DeletedAt"), nil)
	}

	if len(filter.IDs) > 0 {
		q = q.In(f("ID"), toInterfaceSlice(filter.IDs)...)
	}
	if len(filter.ActionTypes) > 0 {
		q = q.In(f("ActionType"), toInterfaceSlice(filter.ActionTypes)...)
	}
	if filter.DisplayName != "" {
		q = q.Regex(f("DisplayName"), filter.DisplayName, "i")
	}
	if filter.RequireApproval != nil {
		q = q.Eq(f("RequireApproval"), *filter.RequireApproval)
	}
	if filter.StartTime != nil {
		q = q.Gte(f("CreatedAt"), *filter.StartTime)
	}
	if filter.EndTime != nil {
		q = q.Lt(f("CreatedAt"), *filter.EndTime)
	}

	return listPaginated(ctx, r.coll, q, gmqb.Desc(f("CreatedAt")), limit, offset)
}

func (r *ActionSchemaRepo) UpdateSchema(ctx context.Context, actionType string, update model.ActionSchemaUpdate) error {
	f := gmqb.Field[model.ActionSchema]
	u := gmqb.NewUpdate()

	if update.DisplayName != nil {
		u = u.Set(f("DisplayName"), *update.DisplayName)
	}
	if update.Description != nil {
		u = u.Set(f("Description"), *update.Description)
	}
	if update.Parameters != nil {
		u = u.Set(f("Parameters"), *update.Parameters)
	}
	if update.ResultSchema != nil {
		u = u.Set(f("ResultSchema"), *update.ResultSchema)
	}
	if update.RequireApproval != nil {
		u = u.Set(f("RequireApproval"), *update.RequireApproval)
	}

	if u.IsEmpty() {
		return nil
	}

	u = u.Set(f("UpdatedAt"), time.Now().UTC())

	_, err := r.coll.UpdateOne(ctx, gmqb.And(
		gmqb.Eq(f("ActionType"), actionType),
		gmqb.Eq(f("DeletedAt"), nil),
	), u)
	return err
}

func (r *ActionSchemaRepo) DeleteSchema(ctx context.Context, actionType string) error {
	f := gmqb.Field[model.ActionSchema]
	u := gmqb.NewUpdate().Set(f("DeletedAt"), time.Now().UTC())
	_, err := r.coll.UpdateOne(ctx, gmqb.And(
		gmqb.Eq(f("ActionType"), actionType),
		gmqb.Eq(f("DeletedAt"), nil),
	), u)
	return err
}
