package model

import (
	"time"
 
	apiv1 "github.com/squall-chua/go-support-ticket/api/v1"
	"go.mongodb.org/mongo-driver/v2/bson"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AuditLog struct {
	ID        bson.ObjectID          `json:"id" bson:"_id,omitempty"`
	TicketID  string                 `json:"ticket_id" bson:"ticket_id"`
	UserID    string                 `json:"user_id" bson:"user_id"`
	Action    string                 `json:"action" bson:"action"`
	Details   string                 `json:"details" bson:"details"`
	Metadata  map[string]interface{} `json:"metadata" bson:"metadata"`
	CreatedAt time.Time              `json:"created_at" bson:"created_at"`
}

func (m *AuditLog) ToProto() *apiv1.AuditEntry {
	pb := &apiv1.AuditEntry{
		Id:        m.ID.Hex(),
		TicketId:  m.TicketID,
		UserId:    m.UserID,
		Action:    m.Action,
		Details:   m.Details,
		CreatedAt: timestamppb.New(m.CreatedAt),
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
		ID:        id,
		TicketID:  pb.TicketId,
		UserID:    pb.UserId,
		Action:    pb.Action,
		Details:   pb.Details,
		CreatedAt: pb.CreatedAt.AsTime(),
	}

	if len(pb.Metadata) > 0 {
		m.Metadata = make(map[string]interface{})
		for k, v := range pb.Metadata {
			m.Metadata[k] = v.AsInterface()
		}
	}

	return m
}
