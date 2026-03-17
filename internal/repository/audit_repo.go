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
	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now().UTC()
	}
	_, err := r.coll.InsertOne(ctx, log)
	return err
}

func (r *AuditRepo) ListLogs(ctx context.Context, ticketID, action string, limit, offset int32) ([]*model.AuditLog, int32, error) {
	f := gmqb.Field[model.AuditLog]
	filter := gmqb.NewFilter()

	if ticketID != "" {
		filter.Eq(f("TicketID"), ticketID)
	}
	if action != "" {
		filter.Eq(f("Action"), action)
	}

	return listPaginated(ctx, r.coll, filter, gmqb.Desc(f("CreatedAt")), limit, offset)
}
