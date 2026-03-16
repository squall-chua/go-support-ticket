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

type TicketRepo struct {
	coll *gmqb.Collection[model.Ticket]
}

func NewTicketRepo(col *mongo.Collection) *TicketRepo {
	return &TicketRepo{coll: gmqb.Wrap[model.Ticket](col)}
}

func (r *TicketRepo) CreateTicket(ctx context.Context, ticket *model.Ticket) error {
	_, err := r.coll.InsertOne(ctx, ticket)
	return err
}

func (r *TicketRepo) GetTicket(ctx context.Context, id string) (*model.Ticket, error) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	return r.coll.FindOne(ctx, gmqb.And(
		gmqb.Eq("_id", oid),
		gmqb.Eq("deleted_at", nil),
	))
}

func (r *TicketRepo) buildTicketUpdate(update model.TicketUpdate) gmqb.Updater {
	f := gmqb.Field[model.Ticket]
	u := gmqb.NewUpdate()

	if update.Title != nil {
		u = u.Set(f("Title"), *update.Title)
	}
	if update.Description != nil {
		u = u.Set(f("Description"), *update.Description)
	}
	if update.TicketType != nil {
		u = u.Set(f("TicketType"), *update.TicketType)
	}
	if update.Status != nil {
		u = u.Set(f("Status"), *update.Status)
	}
	if update.Priority != nil {
		u = u.Set(f("Priority"), *update.Priority)
	}
	if update.AssignedTo != nil {
		u = u.Set(f("AssignedTo"), *update.AssignedTo)
	}
	if update.MergedInto != nil {
		u = u.Set(f("MergedInto"), *update.MergedInto)
	}
	if update.NewComment != nil {
		u = u.Push(f("Comments"), *update.NewComment)
	}

	if len(update.Metadata) > 0 {
		for k, v := range update.Metadata {
			u = u.Set(f("Metadata")+"."+k, v)
		}
	}

	if !u.IsEmpty() {
		u = u.Set(f("UpdatedAt"), update.UpdatedAt)
	}

	return u
}

func (r *TicketRepo) UpdateTicket(ctx context.Context, id string, update model.TicketUpdate) (*model.Ticket, error) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	u := r.buildTicketUpdate(update)
	if u.IsEmpty() {
		return r.GetTicket(ctx, id)
	}

	return r.coll.FindOneAndUpdate(ctx, gmqb.Eq("_id", oid), u, gmqb.WithReturnDocument(options.After))
}

func (r *TicketRepo) UpdateTickets(ctx context.Context, updates map[string]model.TicketUpdate) ([]*model.Ticket, error) {
	if len(updates) == 0 {
		return nil, nil
	}

	var models []gmqb.WriteModel[model.Ticket]
	oids := make([]bson.ObjectID, 0, len(updates))

	for id, update := range updates {
		oid, err := bson.ObjectIDFromHex(id)
		if err != nil {
			continue
		}
		oids = append(oids, oid)

		u := r.buildTicketUpdate(update)
		if u.IsEmpty() {
			continue
		}

		m := gmqb.NewUpdateOneModel[model.Ticket]().
			SetFilter(gmqb.Eq("_id", oid)).
			SetUpdate(u)
		models = append(models, m)
	}

	if len(models) > 0 {
		if _, err := r.coll.BulkWrite(ctx, models); err != nil {
			return nil, err
		}
	}

	// Fetch updated tickets
	tickets, err := r.coll.Find(ctx, gmqb.In("_id", oids))
	if err != nil {
		return nil, err
	}

	res := make([]*model.Ticket, len(tickets))
	for i := range tickets {
		res[i] = &tickets[i]
	}
	return res, nil
}

func (r *TicketRepo) ListTickets(ctx context.Context, filter model.TicketFilter, sorts []model.TicketSort, limit, offset int32) ([]*model.Ticket, int32, error) {
	f := gmqb.Field[model.Ticket]
	q := gmqb.NewFilter()
	if !filter.IncludeDeleted {
		q.Eq(f("DeletedAt"), nil)
	}

	if len(filter.Statuses) > 0 {
		q.In(f("Status"), filter.Statuses)
	}
	if len(filter.AssignedTo) > 0 {
		q.In(f("AssignedTo"), filter.AssignedTo)
	}
	if filter.TitleContains != nil {
		q.Regex(f("Title"), *filter.TitleContains, "i")
	}
	if filter.DescriptionContains != nil {
		q.Regex(f("Description"), *filter.DescriptionContains, "i")
	}
	if len(filter.TicketTypes) > 0 {
		q.In(f("TicketType"), filter.TicketTypes)
	}
	if len(filter.Priorities) > 0 {
		q.In(f("Priority"), filter.Priorities)
	}
	if len(filter.CustomerIDs) > 0 {
		q.In(f("CustomerID"), filter.CustomerIDs)
	}
	if len(filter.CreatedBy) > 0 {
		q.In(f("CreatedBy"), filter.CreatedBy)
	}
	if len(filter.MergedInto) > 0 {
		oids := make([]bson.ObjectID, 0, len(filter.MergedInto))
		for _, id := range filter.MergedInto {
			if oid, err := bson.ObjectIDFromHex(id); err == nil {
				oids = append(oids, oid)
			}
		}
		if len(oids) > 0 {
			q.In(f("MergedInto"), oids)
		}
	}

	for k, v := range filter.Metadata {
		q.Eq(f("Metadata")+"."+k, v)
	}

	var sortFields []gmqb.SortField
	if len(sorts) > 0 {
		for _, s := range sorts {
			sortFields = append(sortFields, gmqb.SortRule(s.Field, s.Order))
		}
	} else {
		sortFields = []gmqb.SortField{
			gmqb.SortRule(f("Priority"), -1),
			gmqb.SortRule(f("CreatedAt"), 1),
		}
	}
	sortSpec := gmqb.SortSpec(sortFields...)

	return listPaginated(ctx, r.coll, q, sortSpec, limit, offset)
}

func (r *TicketRepo) AddComment(ctx context.Context, ticketID string, comment *model.Comment) error {
	oid, err := bson.ObjectIDFromHex(ticketID)
	if err != nil {
		return err
	}
	f := gmqb.Field[model.Ticket]
	_, err = r.coll.UpdateOne(ctx,
		gmqb.Eq("_id", oid),
		gmqb.NewUpdate().Push(f("Comments"), comment),
	)
	return err
}

func (r *TicketRepo) DeleteTicket(ctx context.Context, id string, deletedAt time.Time) error {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	f := gmqb.Field[model.Ticket]
	update := gmqb.NewUpdate().Set(f("DeletedAt"), deletedAt)
	_, err = r.coll.UpdateOne(ctx, gmqb.Eq("_id", oid), update)
	return err
}
