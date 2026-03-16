package repository

import (
	"context"

	"github.com/squall-chua/gmqb"
	"github.com/squall-chua/go-support-ticket/internal/model"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type ApprovalRepo struct {
	coll *gmqb.Collection[model.Approval]
}

func NewApprovalRepo(col *mongo.Collection) *ApprovalRepo {
	return &ApprovalRepo{coll: gmqb.Wrap[model.Approval](col)}
}

func (r *ApprovalRepo) CreateApproval(ctx context.Context, approval *model.Approval) error {
	_, err := r.coll.InsertOne(ctx, approval)
	return err
}

func (r *ApprovalRepo) GetApproval(ctx context.Context, id string) (*model.Approval, error) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	return r.coll.FindOne(ctx, gmqb.Eq("_id", oid))
}

func (r *ApprovalRepo) UpdateApproval(ctx context.Context, id string, update model.ApprovalUpdate) error {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	f := gmqb.Field[model.Approval]
	u := gmqb.NewUpdate()

	if update.Status != nil {
		u = u.Set(f("Status"), *update.Status)
	}
	if update.Decisions != nil {
		u = u.Set(f("Decisions"), *update.Decisions)
	}

	if u.IsEmpty() {
		return nil
	}

	u = u.Set(f("UpdatedAt"), update.UpdatedAt)

	_, err = r.coll.UpdateOne(ctx, gmqb.Eq("_id", oid), u)
	return err
}



func (r *ApprovalRepo) ListApprovals(ctx context.Context, filter model.ApprovalFilter, limit, offset int32) ([]*model.Approval, int32, error) {
	f := gmqb.Field[model.Approval]
	q := gmqb.NewFilter()

	if len(filter.TicketIDs) > 0 {
		q.In(f("TicketID"), filter.TicketIDs)
	}
	if len(filter.Actions) > 0 {
		q.In(f("Action"), filter.Actions)
	}
	if len(filter.Requesters) > 0 {
		q.In(f("Requester"), filter.Requesters)
	}
	if len(filter.Statuses) > 0 {
		q.In(f("Status"), filter.Statuses)
	}
	if len(filter.ExecutionIDs) > 0 {
		q.In(f("ExecutionID"), filter.ExecutionIDs)
	}
	if len(filter.RequiredApprovals) > 0 {
		q.In(f("RequiredApprovals"), filter.RequiredApprovals)
	}
	if len(filter.Approvers) > 0 {
		q.In("decisions.approver", filter.Approvers)
	}
	if filter.StartTime != nil {
		q.Gte(f("CreatedAt"), *filter.StartTime)
	}
	if filter.EndTime != nil {
		q.Lte(f("CreatedAt"), *filter.EndTime)
	}

	return listPaginated(ctx, r.coll, q, gmqb.Desc(f("CreatedAt")), limit, offset)
}
