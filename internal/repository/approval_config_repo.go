package repository

import (
	"context"
	"time"

	"github.com/squall-chua/gmqb"
	"github.com/squall-chua/go-support-ticket/internal/model"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
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

func (r *ApprovalConfigRepo) GetConfig(ctx context.Context, id, actionType, ticketType string) (*model.ApprovalConfig, error) {
	f := gmqb.Field[model.ApprovalConfig]
	q := gmqb.NewFilter().Eq(f("DeletedAt"), nil)
	if id != "" {
		oid, err := bson.ObjectIDFromHex(id)
		if err != nil {
			return nil, err
		}
		q.Eq(f("ID"), oid)
	} else if actionType != "" {
		q.Eq(f("ActionType"), actionType)
	} else if ticketType != "" {
		q.Eq(f("TicketType"), ticketType)
	} else {
		return nil, nil
	}
	return r.coll.FindOne(ctx, q)
}

func (r *ApprovalConfigRepo) UpdateConfig(ctx context.Context, id, actionType, ticketType string, update model.ApprovalConfigUpdate) (*model.ApprovalConfig, error) {
	f := gmqb.Field[model.ApprovalConfig]
	q := gmqb.NewFilter().Eq(f("DeletedAt"), nil)
	if id != "" {
		oid, err := bson.ObjectIDFromHex(id)
		if err != nil {
			return nil, err
		}
		q.Eq(f("ID"), oid)
	} else if actionType != "" {
		q.Eq(f("ActionType"), actionType)
	} else if ticketType != "" {
		q.Eq(f("TicketType"), ticketType)
	} else {
		return nil, nil
	}

	u := gmqb.NewUpdate().Set(f("UpdatedAt"), time.Now().UTC())
	if update.RequiredApprovals != nil {
		u.Set(f("RequiredApprovals"), *update.RequiredApprovals)
	}
	if update.EligibleRoles != nil {
		u.Set(f("EligibleRoles"), update.EligibleRoles)
	}

	return r.coll.FindOneAndUpdate(ctx, q, u)
}

func (r *ApprovalConfigRepo) DeleteConfig(ctx context.Context, id, actionType, ticketType string) (*model.ApprovalConfig, error) {
	f := gmqb.Field[model.ApprovalConfig]
	q := gmqb.NewFilter().Eq(f("DeletedAt"), nil)
	if id != "" {
		oid, err := bson.ObjectIDFromHex(id)
		if err != nil {
			return nil, err
		}
		q.Eq(f("ID"), oid)
	} else if actionType != "" {
		q.Eq(f("ActionType"), actionType)
	} else if ticketType != "" {
		q.Eq(f("TicketType"), ticketType)
	} else {
		return nil, nil
	}

	u := gmqb.NewUpdate().Set(f("DeletedAt"), time.Now().UTC())
	return r.coll.FindOneAndUpdate(ctx, q, u)
}

func (r *ApprovalConfigRepo) ListConfigs(ctx context.Context, filter model.ApprovalConfigFilter, limit, offset int32) ([]*model.ApprovalConfig, int32, error) {
	f := gmqb.Field[model.ApprovalConfig]
	q := gmqb.NewFilter()
	if !filter.IncludeDeleted {
		q.Eq(f("DeletedAt"), nil)
	}

	if len(filter.IDs) > 0 {
		var oids []bson.ObjectID
		for _, id := range filter.IDs {
			if oid, err := bson.ObjectIDFromHex(id); err == nil {
				oids = append(oids, oid)
			}
		}
		if len(oids) > 0 {
			q.In(f("ID"), oids)
		}
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
