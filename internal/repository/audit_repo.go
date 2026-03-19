package repository

import (
	"context"
	"time"

	"github.com/squall-chua/gmqb"
	"github.com/squall-chua/go-support-ticket/internal/model"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type AuditRepo struct {
	coll *gmqb.Collection[model.AuditLog]
}

func NewAuditRepo(col *mongo.Collection) *AuditRepo {
	return &AuditRepo{coll: gmqb.Wrap[model.AuditLog](col)}
}

func (r *AuditRepo) CreateLog(ctx context.Context, log *model.AuditLog) error {
	if log.EventTime.IsZero() {
		log.EventTime = time.Now().UTC()
	}
	_, err := r.coll.InsertOne(ctx, log)
	return err
}

func (r *AuditRepo) ListLogs(ctx context.Context, filter model.AuditLogFilter, limit, offset int32) ([]*model.AuditLog, int32, error) {
	f := gmqb.Field[model.AuditLog]
	q := gmqb.NewFilter()

	if len(filter.EventIDs) > 0 {
		q.In(f("EventID"), filter.EventIDs)
	}
	if len(filter.EventTypes) > 0 {
		q.In(f("EventType"), filter.EventTypes)
	}
	if len(filter.Users) > 0 {
		q.In(f("User"), filter.Users)
	}
	if len(filter.Sources) > 0 {
		q.In(f("Source"), filter.Sources)
	}
	if len(filter.Schemas) > 0 {
		q.In(f("Schema"), filter.Schemas)
	}
	if len(filter.ResourceIDs) > 0 {
		q.In(f("ResourceID"), filter.ResourceIDs)
	}
	if !filter.StartTime.IsZero() {
		q.Gte(f("EventTime"), filter.StartTime)
	}
	if !filter.EndTime.IsZero() {
		q.Lt(f("EventTime"), filter.EndTime)
	}

	applyMetadataFilters(q, "metadata", filter.Metadata)

	return listPaginated(ctx, r.coll, q, gmqb.Desc(f("EventTime")), limit, offset)
}
