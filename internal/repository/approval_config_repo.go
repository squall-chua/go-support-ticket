package repository

import (
	"context"
	"time"

	"github.com/squall-chua/gmqb"
	"github.com/squall-chua/go-support-ticket/internal/model"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type ApprovalConfigRepo struct {
	coll *gmqb.Collection[model.ApprovalConfig]
}

func NewApprovalConfigRepo(col *mongo.Collection) *ApprovalConfigRepo {
	return &ApprovalConfigRepo{coll: gmqb.Wrap[model.ApprovalConfig](col)}
}

func (r *ApprovalConfigRepo) CreateConfig(ctx context.Context, config *model.ApprovalConfig) error {
	now := time.Now().UTC()
	config.CreatedAt = now
	config.UpdatedAt = now
	_, err := r.coll.InsertOne(ctx, config)
	return err
}

func (r *ApprovalConfigRepo) GetConfig(ctx context.Context, ticketType, actionType string) (*model.ApprovalConfig, error) {
	f := gmqb.Field[model.ApprovalConfig]
	q := gmqb.NewFilter().Eq(f("DeletedAt"), nil)
	q.Eq(f("TicketType"), ticketType).Eq(f("ActionType"), actionType)
	return r.coll.FindOne(ctx, q)
}

func (r *ApprovalConfigRepo) UpdateConfig(ctx context.Context, ticketType, actionType string, update model.ApprovalConfigUpdate, returnNew bool) (*model.ApprovalConfig, error) {
	f := gmqb.Field[model.ApprovalConfig]
	q := gmqb.NewFilter().Eq(f("DeletedAt"), nil)
	q.Eq(f("TicketType"), ticketType).Eq(f("ActionType"), actionType)

	u := gmqb.NewUpdate().Set(f("UpdatedAt"), time.Now().UTC())
	if update.RequiredApprovals != nil {
		u.Set(f("RequiredApprovals"), *update.RequiredApprovals)
	}
	if update.EligibleRoles != nil {
		u.Set(f("EligibleRoles"), update.EligibleRoles)
	}

	returnDoc := options.Before
	if returnNew {
		returnDoc = options.After
	}

	return r.coll.FindOneAndUpdate(ctx, q, u, gmqb.WithReturnDocument(returnDoc))
}

func (r *ApprovalConfigRepo) DeleteConfig(ctx context.Context, ticketType, actionType string) (*model.ApprovalConfig, error) {
	f := gmqb.Field[model.ApprovalConfig]
	q := gmqb.NewFilter().Eq(f("DeletedAt"), nil)
	q.Eq(f("TicketType"), ticketType).Eq(f("ActionType"), actionType)

	u := gmqb.NewUpdate().Set(f("DeletedAt"), time.Now().UTC())
	return r.coll.FindOneAndUpdate(ctx, q, u)
}

func (r *ApprovalConfigRepo) ListConfigs(ctx context.Context, filter model.ApprovalConfigFilter, limit, offset int32) ([]*model.ApprovalConfig, int32, error) {
	f := gmqb.Field[model.ApprovalConfig]
	q := gmqb.NewFilter()
	if !filter.IncludeDeleted {
		q.Eq(f("DeletedAt"), nil)
	}

	if len(filter.ActionTypes) > 0 {
		q.In(f("ActionType"), filter.ActionTypes)
	}
	if len(filter.TicketTypes) > 0 {
		q.In(f("TicketType"), filter.TicketTypes)
	}
	if filter.RequiredApprovals != nil {
		q.Eq(f("RequiredApprovals"), *filter.RequiredApprovals)
	}
	if len(filter.EligibleRoles) > 0 {
		// In MongoDB, $in on an array field matches if any element of the array is in the given list
		q.In(f("EligibleRoles"), filter.EligibleRoles)
	}
	if filter.StartTime != nil {
		q.Gte(f("CreatedAt"), *filter.StartTime)
	}
	if filter.EndTime != nil {
		q.Lte(f("CreatedAt"), *filter.EndTime)
	}

	return listPaginated(ctx, r.coll, q, nil, limit, offset)
}
