package repository

import (
	"context"
	"regexp"

	"github.com/squall-chua/gmqb"
	"github.com/squall-chua/go-support-ticket/internal/model"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func toPtrSlice[T any](s []T) []*T {
	res := make([]*T, len(s))
	for i := range s {
		res[i] = &s[i]
	}
	return res
}

func toInterfaceSlice[T any](s []T) []interface{} {
	res := make([]interface{}, len(s))
	for i, v := range s {
		res[i] = v
	}
	return res
}

type paginatedResults[T any] struct {
	Metadata []struct {
		Total int64 `bson:"total"`
	} `bson:"metadata"`
	Data []T `bson:"data"`
}

func listPaginated[T any](ctx context.Context, coll *gmqb.Collection[T], filter gmqb.Filter, sortSpec bson.D, limit, offset int32) ([]*T, int32, error) {
	pipeline := gmqb.NewPipeline().
		Match(filter).
		Facet(map[string]gmqb.Pipeline{
			"metadata": gmqb.NewPipeline().Count("total"),
			"data": gmqb.NewPipeline().
				Sort(sortSpec).
				Skip(int64(offset)).
				Limit(int64(limit)),
		})

	results, err := gmqb.Aggregate[paginatedResults[T]](coll, ctx, pipeline)
	if err != nil {
		return nil, 0, err
	}

	if len(results) == 0 {
		return []*T{}, 0, nil
	}

	res := results[0]
	var total int64
	if len(res.Metadata) > 0 {
		total = res.Metadata[0].Total
	}

	return toPtrSlice(res.Data), int32(total), nil
}

func applyMetadataFilters(q gmqb.Filter, metadataField string, filters []model.MetadataFilter) {
	for _, meta := range filters {
		field := metadataField + "." + meta.Key
		switch meta.Operator {
		case model.OpEqual:
			q.Eq(field, meta.Value)
		case model.OpNotEqual:
			q.Ne(field, meta.Value)
		case model.OpGreaterThan:
			q.Gt(field, meta.Value)
		case model.OpGreaterThanOrEqual:
			q.Gte(field, meta.Value)
		case model.OpLessThan:
			q.Lt(field, meta.Value)
		case model.OpLessThanOrEqual:
			q.Lte(field, meta.Value)
		case model.OpContains:
			if v, ok := meta.Value.(string); ok {
				q.Regex(field, ".*"+regexp.QuoteMeta(v)+".*", "i")
			}
		case model.OpIn:
			q.In(field, meta.Value)
		case model.OpNotIn:
			q.Nin(field, meta.Value)
		case model.OpExists:
			q.Exists(field, true)
		case model.OpNotExists:
			q.Exists(field, false)
		case model.OpStartsWith:
			if v, ok := meta.Value.(string); ok {
				q.Regex(field, "^"+regexp.QuoteMeta(v), "i")
			}
		case model.OpEndsWith:
			if v, ok := meta.Value.(string); ok {
				q.Regex(field, regexp.QuoteMeta(v)+"$", "i")
			}
		case model.OpRegex:
			if v, ok := meta.Value.(string); ok {
				q.Regex(field, v, "i")
			}
		case model.OpIsNull:
			q.Eq(field, nil)
		case model.OpIsNotNull:
			q.Ne(field, nil)
		}
	}
}
