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
	ActionType        string        `json:"action_type" bson:"action_type"`
	Requester         string        `json:"requester" bson:"requester"`
	Status            int32         `json:"status" bson:"status"`
	RequiredApprovals int32         `json:"required_approvals" bson:"required_approvals"`
	EligibleRoles     []string      `json:"eligible_roles" bson:"eligible_roles"`
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
		ActionType:        m.ActionType,
		ExecutionId:       m.ExecutionID,
		Requester:         m.Requester,
		Status:            apiv1.ApprovalStatus(m.Status),
		RequiredApprovals: m.RequiredApprovals,
		EligibleRoles:     m.EligibleRoles,
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
		ActionType:        pb.ActionType,
		Requester:         pb.Requester,
		Status:            int32(pb.Status),
		RequiredApprovals: pb.RequiredApprovals,
		ExecutionID:       pb.ExecutionId,
		EligibleRoles:     pb.EligibleRoles,
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
	Decision  *Decision
}

type ApprovalFilter struct {
	TicketIDs         []string
	ActionTypes       []string
	Requesters        []string
	Statuses          []int32
	ExecutionIDs      []string
	RequiredApprovals []int32
	Approvers         []string
	StartTime         *time.Time
	EndTime           *time.Time
}
type ApprovalConfig struct {
	ApprovalConfigID  bson.ObjectID `json:"approval_config_id" bson:"_id,omitempty"`
	ActionType        *string       `json:"action_type,omitempty" bson:"action_type,omitempty"`
	TicketType        *string       `json:"ticket_type,omitempty" bson:"ticket_type,omitempty"`
	RequiredApprovals int32         `json:"required_approvals" bson:"required_approvals"`
	EligibleRoles     []string      `json:"eligible_roles" bson:"eligible_roles"`
	CreatedAt         time.Time     `json:"created_at" bson:"created_at"`
	UpdatedAt         time.Time     `json:"updated_at" bson:"updated_at"`
	DeletedAt         *time.Time    `json:"deleted_at" bson:"deleted_at,omitempty"`
}

func (m *ApprovalConfig) ToProto() *apiv1.ApprovalConfig {
	pb := &apiv1.ApprovalConfig{
		Id:                m.ApprovalConfigID.Hex(),
		RequiredApprovals: m.RequiredApprovals,
		EligibleRoles:     m.EligibleRoles,
		CreatedAt:         timestamppb.New(m.CreatedAt),
		UpdatedAt:         timestamppb.New(m.UpdatedAt),
	}
	if m.ActionType != nil {
		pb.Target = &apiv1.ApprovalConfig_ActionType{ActionType: *m.ActionType}
	} else if m.TicketType != nil {
		pb.Target = &apiv1.ApprovalConfig_TicketType{TicketType: *m.TicketType}
	}

	if m.DeletedAt != nil {
		pb.DeletedAt = timestamppb.New(*m.DeletedAt)
	}
	return pb
}

func ApprovalConfigFromProto(pb *apiv1.ApprovalConfig) *ApprovalConfig {
	if pb == nil {
		return nil
	}
	m := &ApprovalConfig{
		RequiredApprovals: pb.RequiredApprovals,
		EligibleRoles:     pb.EligibleRoles,
	}
	if pb.Id != "" {
		m.ApprovalConfigID, _ = bson.ObjectIDFromHex(pb.Id)
	}

	switch t := pb.Target.(type) {
	case *apiv1.ApprovalConfig_ActionType:
		m.ActionType = &t.ActionType
	case *apiv1.ApprovalConfig_TicketType:
		m.TicketType = &t.TicketType
	}

	if pb.CreatedAt != nil {
		m.CreatedAt = pb.CreatedAt.AsTime()
	}
	if pb.UpdatedAt != nil {
		m.UpdatedAt = pb.UpdatedAt.AsTime()
	}
	if pb.DeletedAt != nil {
		deletedAt := pb.DeletedAt.AsTime()
		m.DeletedAt = &deletedAt
	}
	return m
}

type ApprovalConfigFilter struct {
	IDs               []string
	ActionTypes       []string
	TicketTypes       []string
	RequiredApprovals *int32
	EligibleRoles     []string
	StartTime         *time.Time
	EndTime           *time.Time
	IncludeDeleted    bool
}

type ApprovalConfigUpdate struct {
	RequiredApprovals *int32
	EligibleRoles     []string
}
