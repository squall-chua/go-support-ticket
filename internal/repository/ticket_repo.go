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
	now := time.Now().UTC()
	ticket.CreatedAt = now
	ticket.UpdatedAt = now
	_, err := r.coll.InsertOne(ctx, ticket)
	return err
}

func (r *TicketRepo) GetTicket(ctx context.Context, id string, userID string, roles []string) (*model.Ticket, error) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	f := gmqb.Field[model.Ticket]

	query := gmqb.And(
		gmqb.Eq(f("ID"), oid),
		gmqb.Eq(f("DeletedAt"), nil),
		r.buildVisibilityFilter(userID, roles),
	)

	return r.coll.FindOne(ctx, query)
}

func (r *TicketRepo) buildVisibilityFilter(userID string, roles []string) gmqb.Filter {
	f := gmqb.Field[model.Ticket]
	var roleFilter gmqb.Filter
	if len(roles) == 0 {
		roleFilter = gmqb.Or(
			gmqb.Eq(f("VisibleRoles"), nil),
			gmqb.Size(f("VisibleRoles"), 0),
		)
	} else {
		roleFilter = gmqb.Or(
			gmqb.Eq(f("VisibleRoles"), nil),
			gmqb.Size(f("VisibleRoles"), 0),
			gmqb.In(f("VisibleRoles"), toInterfaceSlice(roles)...),
		)
	}

	if userID != "" {
		return gmqb.Or(
			roleFilter,
			gmqb.Eq(f("CustomerID"), userID),
			gmqb.Eq(f("CreatedBy"), userID),
			gmqb.Eq(f("AssignedTo"), userID),
		)
	}
	return roleFilter
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
	if update.RequireApproval != nil {
		u = u.Set(f("RequireApproval"), *update.RequireApproval)
	}
	if update.VisibleRoles != nil {
		u = u.Set(f("VisibleRoles"), update.VisibleRoles)
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
		u = u.Set(f("UpdatedAt"), time.Now().UTC())
	}

	return u
}

func (r *TicketRepo) UpdateTicket(ctx context.Context, id string, update model.TicketUpdate, userID string, roles []string) (*model.Ticket, error) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	u := r.buildTicketUpdate(update)
	if u.IsEmpty() {
		return r.GetTicket(ctx, id, userID, roles)
	}

	f := gmqb.Field[model.Ticket]
	query := gmqb.And(gmqb.Eq(f("ID"), oid), r.buildVisibilityFilter(userID, roles))
	return r.coll.FindOneAndUpdate(ctx, query, u, gmqb.WithReturnDocument(options.After))
}

func (r *TicketRepo) UpdateTickets(ctx context.Context, updates map[string]model.TicketUpdate, userID string, roles []string) ([]*model.Ticket, error) {
	if len(updates) == 0 {
		return nil, nil
	}

	f := gmqb.Field[model.Ticket]
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

		filter := gmqb.And(gmqb.Eq(f("ID"), oid), r.buildVisibilityFilter(userID, roles))

		m := gmqb.NewUpdateOneModel[model.Ticket]().
			SetFilter(filter).
			SetUpdate(u)
		models = append(models, m)
	}

	if len(models) > 0 {
		if _, err := r.coll.BulkWrite(ctx, models); err != nil {
			return nil, err
		}
	}

	// Fetch updated tickets
	query := gmqb.And(gmqb.In(f("ID"), oids), r.buildVisibilityFilter(userID, roles))
	tickets, err := r.coll.Find(ctx, query)
	if err != nil {
		return nil, err
	}

	res := make([]*model.Ticket, len(tickets))
	for i := range tickets {
		res[i] = &tickets[i]
	}
	return res, nil
}

func (r *TicketRepo) ListTickets(ctx context.Context, filter model.TicketFilter, sorts []model.TicketSort, userID string, roles []string, limit, offset int32) ([]*model.Ticket, int32, error) {
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

	visibilityFilter := r.buildVisibilityFilter(userID, roles)

	finalFilter := gmqb.And(q, visibilityFilter)

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

	return listPaginated(ctx, r.coll, finalFilter, sortSpec, limit, offset)
}

func (r *TicketRepo) AddComment(ctx context.Context, ticketID string, comment *model.Comment, userID string, roles []string) error {
	oid, err := bson.ObjectIDFromHex(ticketID)
	if err != nil {
		return err
	}
	if comment.CreatedAt.IsZero() {
		comment.CreatedAt = time.Now().UTC()
	}

	f := gmqb.Field[model.Ticket]
	query := gmqb.And(gmqb.Eq(f("ID"), oid), r.buildVisibilityFilter(userID, roles))

	_, err = r.coll.UpdateOne(ctx,
		query,
		gmqb.NewUpdate().
			Push(f("Comments"), comment).
			Set(f("UpdatedAt"), time.Now().UTC()),
	)
	return err
}

func (r *TicketRepo) DeleteTicket(ctx context.Context, id string, userID string, roles []string) error {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	f := gmqb.Field[model.Ticket]
	query := gmqb.And(gmqb.Eq(f("ID"), oid), r.buildVisibilityFilter(userID, roles))

	update := gmqb.NewUpdate().Set(f("DeletedAt"), time.Now().UTC())
	_, err = r.coll.UpdateOne(ctx, query, update)
	return err
}
