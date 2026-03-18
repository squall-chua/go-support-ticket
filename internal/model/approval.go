package model

import (
	"time"

	apiv1 "github.com/squall-chua/go-support-ticket/api/v1"
	"go.mongodb.org/mongo-driver/v2/bson"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Approval struct {
	ID                bson.ObjectID  `json:"id" bson:"_id,omitempty"`
	TicketID          string         `json:"ticket_id" bson:"ticket_id"`
	TicketType        string         `json:"ticket_type" bson:"ticket_type"`
	ActionType        string         `json:"action_type" bson:"action_type"`
	Requester         string         `json:"requester" bson:"requester"`
	Origin            string         `json:"origin" bson:"origin"`
	Status            int32          `json:"status" bson:"status"`
	RequiredApprovals int32          `json:"required_approvals" bson:"required_approvals"`
	EligibleRoles     []string       `json:"eligible_roles" bson:"eligible_roles"`
	TargetID          string         `json:"target_id" bson:"target_id"`
	Decisions         []Decision     `json:"decisions" bson:"decisions"`
	Metadata          map[string]any `json:"metadata" bson:"metadata"`
	CreatedAt         time.Time      `json:"created_at" bson:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at" bson:"updated_at"`
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
		TicketType:        m.TicketType,
		ActionType:        m.ActionType,
		TargetId:          m.TargetID,
		Requester:         m.Requester,
		Origin:            m.Origin,
		Status:            apiv1.ApprovalStatus(m.Status),
		RequiredApprovals: m.RequiredApprovals,
		EligibleRoles:     m.EligibleRoles,
		CreatedAt:         timestamppb.New(m.CreatedAt),
		UpdatedAt:         timestamppb.New(m.UpdatedAt),
	}

	if m.Metadata != nil {
		pb.Metadata = make(map[string]*structpb.Value)
		for k, v := range m.Metadata {
			if val, err := structpb.NewValue(v); err == nil {
				pb.Metadata[k] = val
			}
		}
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
		TicketType:        pb.TicketType,
		ActionType:        pb.ActionType,
		Requester:         pb.Requester,
		Origin:            pb.Origin,
		Status:            int32(pb.Status),
		RequiredApprovals: pb.RequiredApprovals,
		TargetID:          pb.TargetId,
		EligibleRoles:     pb.EligibleRoles,
		CreatedAt:         pb.CreatedAt.AsTime(),
		UpdatedAt:         pb.UpdatedAt.AsTime(),
	}

	if pb.Metadata != nil {
		m.Metadata = make(map[string]any)
		for k, v := range pb.Metadata {
			m.Metadata[k] = v.AsInterface()
		}
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
	Status   *int32
	Decision *Decision
}

type ApprovalFilter struct {
	TicketIDs         []string
	TicketTypes       []string
	ActionTypes       []string
	Requesters        []string
	Statuses          []int32
	Origins           []string
	TargetIDs         []string
	RequiredApprovals []int32
	Approvers         []string
	StartTime         *time.Time
	EndTime           *time.Time
}
type ApprovalConfig struct {
	ApprovalConfigID  bson.ObjectID `json:"approval_config_id" bson:"_id,omitempty"`
	TicketType        string        `json:"ticket_type" bson:"ticket_type"`
	ActionType        string        `json:"action_type" bson:"action_type"`
	RequiredApprovals int32         `json:"required_approvals" bson:"required_approvals"`
	EligibleRoles     []string      `json:"eligible_roles" bson:"eligible_roles"`
	CreatedAt         time.Time     `json:"created_at" bson:"created_at"`
	UpdatedAt         time.Time     `json:"updated_at" bson:"updated_at"`
	DeletedAt         *time.Time    `json:"deleted_at" bson:"deleted_at,omitempty"`
}

func (m *ApprovalConfig) ToProto() *apiv1.ApprovalConfig {
	pb := &apiv1.ApprovalConfig{
		ActionType:        m.ActionType,
		TicketType:        m.TicketType,
		RequiredApprovals: m.RequiredApprovals,
		EligibleRoles:     m.EligibleRoles,
		CreatedAt:         timestamppb.New(m.CreatedAt),
		UpdatedAt:         timestamppb.New(m.UpdatedAt),
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
		TicketType:        pb.TicketType,
		ActionType:        pb.ActionType,
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


