package model

import (
	"time"

	apiv1 "github.com/squall-chua/go-support-ticket/api/v1"
	"go.mongodb.org/mongo-driver/v2/bson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Approval struct {
	ID                bson.ObjectID `json:"id" bson:"_id,omitempty"`
	TicketID          string        `json:"ticket_id" bson:"ticket_id"`
	Action            string        `json:"action" bson:"action"`
	Requester         string        `json:"requester" bson:"requester"`
	Status            int32         `json:"status" bson:"status"`
	RequiredApprovals int32         `json:"required_approvals" bson:"required_approvals"`
	ExecutionID       string        `json:"execution_id" bson:"execution_id"`
	Decisions         []Decision    `json:"decisions" bson:"decisions"`
	CreatedAt         time.Time     `json:"created_at" bson:"created_at"`
	UpdatedAt         time.Time     `json:"updated_at" bson:"updated_at"`
}

type Decision struct {
	Approver  string    `json:"approver" bson:"approver"`
	Approved  bool      `json:"approved" bson:"approved"`
	Reason    string    `json:"reason" bson:"reason"`
	DecidedAt time.Time `json:"decided_at" bson:"decided_at"`
}

func (m *Approval) ToProto() *apiv1.ApprovalRequestData {
	pb := &apiv1.ApprovalRequestData{
		Id:                m.ID.Hex(),
		TicketId:          m.TicketID,
		Action:            m.Action,
		Requester:         m.Requester,
		Status:            apiv1.ApprovalStatus(m.Status),
		RequiredApprovals: m.RequiredApprovals,
		ExecutionId:       m.ExecutionID,
		CreatedAt:         timestamppb.New(m.CreatedAt),
		UpdatedAt:         timestamppb.New(m.UpdatedAt),
	}

	for _, d := range m.Decisions {
		pb.Decisions = append(pb.Decisions, d.ToProto())
	}

	return pb
}

func ApprovalFromProto(pb *apiv1.ApprovalRequestData) *Approval {
	if pb == nil {
		return nil
	}
	id, _ := bson.ObjectIDFromHex(pb.Id)
	m := &Approval{
		ID:                id,
		TicketID:          pb.TicketId,
		Action:            pb.Action,
		Requester:         pb.Requester,
		Status:            int32(pb.Status),
		RequiredApprovals: pb.RequiredApprovals,
		ExecutionID:       pb.ExecutionId,
		CreatedAt:         pb.CreatedAt.AsTime(),
		UpdatedAt:         pb.UpdatedAt.AsTime(),
	}

	for _, d := range pb.Decisions {
		m.Decisions = append(m.Decisions, DecisionFromProto(d))
	}

	return m
}

func (d Decision) ToProto() *apiv1.ApprovalDecision {
	return &apiv1.ApprovalDecision{
		Approver:  d.Approver,
		Approved:  d.Approved,
		Reason:    d.Reason,
		DecidedAt: timestamppb.New(d.DecidedAt),
	}
}

func DecisionFromProto(pb *apiv1.ApprovalDecision) Decision {
	if pb == nil {
		return Decision{}
	}
	return Decision{
		Approver:  pb.Approver,
		Approved:  pb.Approved,
		Reason:    pb.Reason,
		DecidedAt: pb.DecidedAt.AsTime(),
	}
}

type ApprovalUpdate struct {
	Status    *int32
	Decisions *[]Decision
	UpdatedAt time.Time
}

type ApprovalFilter struct {
	TicketIDs         []string
	Actions           []string
	Requesters        []string
	Statuses          []int32
	ExecutionIDs      []string
	RequiredApprovals []int32
	Approvers         []string
	StartTime         *time.Time
	EndTime           *time.Time
}
