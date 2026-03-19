package repository

import (
	"context"
	"time"

	"github.com/squall-chua/gmqb"
	"github.com/squall-chua/go-support-ticket/internal/model"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
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

func (r *ActionSchemaRepo) GetSchema(ctx context.Context, idOrType string) (*model.ActionSchema, error) {
	f := gmqb.Field[model.ActionSchema]
	q := gmqb.NewFilter().Eq(f("DeletedAt"), nil)
	if oid, err := bson.ObjectIDFromHex(idOrType); err == nil {
		q = q.Eq(f("ID"), oid)
	} else {
		q = q.Eq(f("ActionType"), idOrType)
	}
	return r.coll.FindOne(ctx, q)
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

func (r *ActionSchemaRepo) UpdateSchema(ctx context.Context, id string, update model.ActionSchemaUpdate, returnNew bool) (*model.ActionSchema, error) {
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
		return r.GetSchema(ctx, id)
	}

	u = u.Set(f("UpdatedAt"), time.Now().UTC())

	returnDoc := options.Before
	if returnNew {
		returnDoc = options.After
	}

	q := gmqb.NewFilter().Eq(f("DeletedAt"), nil)
	if oid, err := bson.ObjectIDFromHex(id); err == nil {
		q = q.Eq(f("ID"), oid)
	} else {
		q = q.Eq(f("ActionType"), id)
	}

	return r.coll.FindOneAndUpdate(ctx, q, u, gmqb.WithReturnDocument(returnDoc))
}

func (r *ActionSchemaRepo) DeleteSchema(ctx context.Context, id string) (*model.ActionSchema, error) {
	f := gmqb.Field[model.ActionSchema]
	q := gmqb.NewFilter().Eq(f("DeletedAt"), nil)
	if oid, err := bson.ObjectIDFromHex(id); err == nil {
		q = q.Eq(f("ID"), oid)
	} else {
		q = q.Eq(f("ActionType"), id)
	}

	u := gmqb.NewUpdate().Set(f("DeletedAt"), time.Now().UTC())
	return r.coll.FindOneAndUpdate(ctx, q, u, gmqb.WithReturnDocument(options.Before))
}
