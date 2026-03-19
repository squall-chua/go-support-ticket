package model

import (
	"time"

	apiv1 "github.com/squall-chua/go-support-ticket/api/v1"
	"github.com/squall-chua/go-support-ticket/internal/eventbus"
	"go.mongodb.org/mongo-driver/v2/bson"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AuditLog struct {
	ID         bson.ObjectID           `json:"id" bson:"_id,omitempty"`
	EventID    string                  `json:"event_id" bson:"event_id"`
	EventType  string                  `json:"event_type" bson:"event_type"`
	EventTime  time.Time               `json:"event_time" bson:"event_time"`
	User       string                  `json:"user" bson:"user"`
	Source     string                  `json:"source" bson:"source"`
	Schema     string                  `json:"schema" bson:"schema"`
	ResourceID string                  `json:"resource_id" bson:"resource_id"`
	Data       eventbus.ProtoMarshaler `json:"data" bson:"data"`
	Metadata   map[string]interface{}  `json:"metadata" bson:"metadata"`
}

func (m *AuditLog) ToProto() *apiv1.AuditEntry {
	pb := &apiv1.AuditEntry{
		Id:         m.ID.Hex(),
		EventId:    m.EventID,
		EventType:  m.EventType,
		EventTime:  timestamppb.New(m.EventTime),
		User:       m.User,
		Source:     m.Source,
		Schema:     m.Schema,
		ResourceId: m.ResourceID,
		Data:       m.Data.Message.(*anypb.Any),
	}

	if len(m.Metadata) > 0 {
		pb.Metadata = make(map[string]*structpb.Value)
		for k, v := range m.Metadata {
			sv, err := structpb.NewValue(v)
			if err == nil {
				pb.Metadata[k] = sv
			}
		}
	}

	return pb
}

func AuditLogFromProto(pb *apiv1.AuditEntry) *AuditLog {
	if pb == nil {
		return nil
	}
	id, _ := bson.ObjectIDFromHex(pb.Id)
	m := &AuditLog{
		ID:         id,
		EventID:    pb.EventId,
		EventType:  pb.EventType,
		EventTime:  pb.EventTime.AsTime(),
		User:       pb.User,
		Source:     pb.Source,
		Schema:     pb.Schema,
		ResourceID: pb.ResourceId,
		Data:       eventbus.ProtoMarshaler{Message: pb.Data},
	}

	if len(pb.Metadata) > 0 {
		m.Metadata = make(map[string]interface{})
		for k, v := range pb.Metadata {
			m.Metadata[k] = v.AsInterface()
		}
	}

	return m
}

type AuditLogFilter struct {
	EventIDs    []string
	EventTypes  []string
	Users       []string
	Sources     []string
	Schemas     []string
	ResourceIDs []string
	Metadata    []MetadataFilter
	StartTime   time.Time
	EndTime     time.Time
}

type MetadataFilter struct {
	Key      string
	Operator MetadataOperator
	Value    interface{}
}

type MetadataOperator int32

const (
	OpUnspecified MetadataOperator = iota
	OpEqual
	OpNotEqual
	OpGreaterThan
	OpLessThan
	OpGreaterThanOrEqual
	OpLessThanOrEqual
	OpContains
	OpIn
	OpNotIn
	OpExists
	OpNotExists
	OpStartsWith
	OpEndsWith
	OpRegex
	OpIsNull
	OpIsNotNull
)
