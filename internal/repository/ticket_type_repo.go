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

type TicketTypeRepo struct {
	coll *gmqb.Collection[model.TicketType]
}

func NewTicketTypeRepo(col *mongo.Collection) *TicketTypeRepo {
	return &TicketTypeRepo{coll: gmqb.Wrap[model.TicketType](col)}
}

func (r *TicketTypeRepo) CreateType(ctx context.Context, tType *model.TicketType) error {
	now := time.Now().UTC()
	tType.CreatedAt = now
	tType.UpdatedAt = now
	_, err := r.coll.InsertOne(ctx, tType)
	return err
}

func (r *TicketTypeRepo) GetType(ctx context.Context, idOrName string) (*model.TicketType, error) {
	f := gmqb.Field[model.TicketType]
	filter := gmqb.NewFilter().Eq(f("DeletedAt"), nil)

	if oid, err := bson.ObjectIDFromHex(idOrName); err == nil {
		filter.Eq(f("ID"), oid)
	} else {
		filter.Eq(f("Name"), idOrName)
	}

	return r.coll.FindOne(ctx, filter)
}

func (r *TicketTypeRepo) ListTypes(ctx context.Context, filter model.TicketTypeFilter, sorts []model.TicketTypeSort, limit, offset int32) ([]*model.TicketType, int32, error) {
	f := gmqb.Field[model.TicketType]
	q := gmqb.NewFilter()
	if !filter.IncludeDeleted {
		q.Eq(f("DeletedAt"), nil)
	}

	if filter.Name != nil {
		q.Regex(f("Name"), *filter.Name, "i")
	}
	if filter.DisplayName != nil {
		q.Regex(f("DisplayName"), *filter.DisplayName, "i")
	}
	if filter.Description != nil {
		q.Regex(f("Description"), *filter.Description, "i")
	}
	if filter.RequireApproval != nil {
		q.Eq(f("RequireApproval"), *filter.RequireApproval)
	}
	if len(filter.VisibleRoles) > 0 {
		q.In(f("VisibleRoles"), filter.VisibleRoles)
	}
	if filter.Activated != nil {
		q.Eq(f("Activated"), *filter.Activated)
	}

	sortSpec := bson.D{}
	if len(sorts) > 0 {
		for _, s := range sorts {
			sortSpec = append(sortSpec, bson.E{Key: s.Field, Value: s.Order})
		}
	} else {
		sortSpec = gmqb.Desc(f("CreatedAt"))
	}

	return listPaginated(ctx, r.coll, q, sortSpec, limit, offset)
}

func (r *TicketTypeRepo) UpdateType(ctx context.Context, id string, update model.TicketTypeUpdate) (*model.TicketType, error) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	f := gmqb.Field[model.TicketType]
	u := gmqb.NewUpdate()

	if update.DisplayName != nil {
		u = u.Set(f("DisplayName"), *update.DisplayName)
	}
	if update.Description != nil {
		u = u.Set(f("Description"), *update.Description)
	}
	if update.RequireApproval != nil {
		u = u.Set(f("RequireApproval"), *update.RequireApproval)
	}
	if update.VisibleRoles != nil {
		u = u.Set(f("VisibleRoles"), update.VisibleRoles)
	}
	if update.Activated != nil {
		u = u.Set(f("Activated"), *update.Activated)
	}

	if u.IsEmpty() {
		return r.GetType(ctx, id)
	}

	u = u.Set(f("UpdatedAt"), time.Now().UTC())

	return r.coll.FindOneAndUpdate(ctx, gmqb.Eq(f("ID"), oid), u, gmqb.WithReturnDocument(options.After))
}

func (r *TicketTypeRepo) DeleteType(ctx context.Context, id string) error {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	f := gmqb.Field[model.TicketType]
	update := gmqb.NewUpdate().Set(f("DeletedAt"), time.Now().UTC())
	_, err = r.coll.UpdateOne(ctx, gmqb.Eq(f("ID"), oid), update)
	return err
}
