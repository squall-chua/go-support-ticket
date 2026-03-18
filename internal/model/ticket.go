package model

import (
	"time"

	apiv1 "github.com/squall-chua/go-support-ticket/api/v1"
	"go.mongodb.org/mongo-driver/v2/bson"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Ticket struct {
	ID              bson.ObjectID `json:"id" bson:"_id,omitempty"`
	Title           string        `json:"title" bson:"title"`
	Description     string        `json:"description" bson:"description"`
	TicketType      string        `json:"ticket_type" bson:"ticket_type"`
	RequireApproval bool          `json:"require_approval" bson:"require_approval"`
	VisibleRoles    []string      `json:"visible_roles" bson:"visible_roles"`
	Status          int32         `json:"status" bson:"status"`
	Priority        int32         `json:"priority" bson:"priority"`
	CustomerID      string        `json:"customer_id" bson:"customer_id"`
	CreatedBy       string        `json:"created_by" bson:"created_by"`
	AssignedTo      string        `json:"assigned_to" bson:"assigned_to"`
	MergedInto      bson.ObjectID `json:"merged_into" bson:"merged_into"`
	Comments        []Comment     `json:"comments" bson:"comments"`
	Metadata        Metadata      `json:"metadata" bson:"metadata"`
	CreatedAt       time.Time     `json:"created_at" bson:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at" bson:"updated_at"`
	DeletedAt       *time.Time    `json:"deleted_at,omitempty" bson:"deleted_at,omitempty"`
}

type TicketUpdate struct {
	Title           *string        `json:"title,omitempty"`
	Description     *string        `json:"description,omitempty"`
	TicketType      *string        `json:"ticket_type,omitempty"`
	RequireApproval *bool          `json:"require_approval,omitempty"`
	VisibleRoles    []string       `json:"visible_roles,omitempty"`
	Status          *int32         `json:"status,omitempty"`
	Priority        *int32         `json:"priority,omitempty"`
	AssignedTo      *string        `json:"assigned_to,omitempty"`
	MergedInto      *bson.ObjectID `json:"merged_into,omitempty"`
	NewComment      *Comment       `json:"new_comment,omitempty"`
	Metadata        Metadata       `json:"metadata,omitempty"`
}

type TicketFilter struct {
	Statuses            []int32
	AssignedTo          []string
	TitleContains       *string
	DescriptionContains *string
	TicketTypes         []string
	Priorities          []int32
	CustomerIDs         []string
	CreatedBy           []string
	MergedInto          []string
	Metadata            Metadata
	IncludeDeleted      bool
	UserRoles           []string
}

type TicketSort struct {
	Field string
	Order int // 1 for asc, -1 for desc
}

type Comment struct {
	ID        bson.ObjectID `json:"id" bson:"id,omitempty"`
	Author    string        `json:"author" bson:"author"`
	Content   string        `json:"content" bson:"content"`
	CreatedAt time.Time     `json:"created_at" bson:"created_at"`
}

type TicketType struct {
	ID              bson.ObjectID `json:"id" bson:"_id,omitempty"`
	Name            string        `json:"name" bson:"name"`
	DisplayName     string        `json:"display_name" bson:"display_name"`
	Description     string        `json:"description" bson:"description"`
	RequireApproval bool          `json:"require_approval" bson:"require_approval"`
	VisibleRoles    []string      `json:"visible_roles" bson:"visible_roles"`
	Activated       bool          `json:"activated" bson:"activated"`
	CreatedAt       time.Time     `json:"created_at" bson:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at" bson:"updated_at"`
	DeletedAt       *time.Time    `json:"deleted_at,omitempty" bson:"deleted_at,omitempty"`
}

type TicketTypeUpdate struct {
	DisplayName     *string
	Description     *string
	RequireApproval *bool
	VisibleRoles    []string
	Activated       *bool
}

type TicketTypeFilter struct {
	Name            *string
	DisplayName     *string
	Description     *string
	RequireApproval *bool
	VisibleRoles    []string
	Activated       *bool
	IncludeDeleted  bool
}

type TicketTypeSort struct {
	Field string
	Order int // 1 for asc, -1 for desc
}

type Metadata map[string]interface{}

func (m *TicketType) ToProto() *apiv1.TicketType {
	if m == nil {
		return nil
	}
	t := &apiv1.TicketType{
		Id:              m.ID.Hex(),
		Name:            m.Name,
		DisplayName:     m.DisplayName,
		Description:     m.Description,
		RequireApproval: m.RequireApproval,
		VisibleRoles:    m.VisibleRoles,
		Activated:       m.Activated,
		CreatedAt:       timestamppb.New(m.CreatedAt),
		UpdatedAt:       timestamppb.New(m.UpdatedAt),
	}
	if m.DeletedAt != nil {
		t.DeletedAt = timestamppb.New(*m.DeletedAt)
	}
	return t
}

func (m *Ticket) ToProto() (*apiv1.Ticket, error) {
	t := &apiv1.Ticket{
		Id:              m.ID.Hex(),
		Title:           m.Title,
		Description:     m.Description,
		TicketType:      m.TicketType,
		RequireApproval: m.RequireApproval,
		VisibleRoles:    m.VisibleRoles,
		Status:          apiv1.TicketStatus(m.Status),
		Priority:        apiv1.TicketPriority(m.Priority),
		CustomerId:      m.CustomerID,
		CreatedBy:       m.CreatedBy,
		AssignedTo:      m.AssignedTo,
		MergedInto: func() string {
			if m.MergedInto.IsZero() {
				return ""
			}
			return m.MergedInto.Hex()
		}(),
		CreatedAt: timestamppb.New(m.CreatedAt),
		UpdatedAt: timestamppb.New(m.UpdatedAt),
		DeletedAt: func() *timestamppb.Timestamp {
			if m.DeletedAt == nil {
				return nil
			}
			return timestamppb.New(*m.DeletedAt)
		}(),
	}

	for _, c := range m.Comments {
		t.Comments = append(t.Comments, c.ToProto())
	}

	if len(m.Metadata) > 0 {
		t.Metadata = make(map[string]*structpb.Value)
		for k, v := range m.Metadata {
			sv, err := structpb.NewValue(v)
			if err == nil {
				t.Metadata[k] = sv
			}
		}
	}

	return t, nil
}

func TicketFromProto(pb *apiv1.Ticket) *Ticket {
	if pb == nil {
		return nil
	}
	id, _ := bson.ObjectIDFromHex(pb.Id)
	t := &Ticket{
		ID:              id,
		Title:           pb.Title,
		Description:     pb.Description,
		TicketType:      pb.TicketType,
		RequireApproval: pb.RequireApproval,
		VisibleRoles:    pb.VisibleRoles,
		Status:          int32(pb.Status),
		Priority:        int32(pb.Priority),
		CustomerID:      pb.CustomerId,
		CreatedBy:       pb.CreatedBy,
		AssignedTo:      pb.AssignedTo,
		MergedInto:      func() bson.ObjectID { oid, _ := bson.ObjectIDFromHex(pb.MergedInto); return oid }(),
		CreatedAt:       pb.CreatedAt.AsTime(),
		UpdatedAt:       pb.UpdatedAt.AsTime(),
		DeletedAt: func() *time.Time {
			if pb.DeletedAt == nil {
				return nil
			}
			t := pb.DeletedAt.AsTime()
			return &t
		}(),
	}

	for _, c := range pb.Comments {
		t.Comments = append(t.Comments, CommentFromProto(c))
	}

	if len(pb.Metadata) > 0 {
		t.Metadata = make(Metadata)
		for k, v := range pb.Metadata {
			t.Metadata[k] = v.AsInterface()
		}
	}

	return t
}

func TicketTypeFromProto(pb *apiv1.TicketType) *TicketType {
	if pb == nil {
		return nil
	}
	id, _ := bson.ObjectIDFromHex(pb.Id)
	return &TicketType{
		ID:              id,
		Name:            pb.Name,
		DisplayName:     pb.DisplayName,
		Description:     pb.Description,
		RequireApproval: pb.RequireApproval,
		VisibleRoles:    pb.VisibleRoles,
		Activated:       pb.Activated,
		DeletedAt: func() *time.Time {
			if pb.DeletedAt == nil {
				return nil
			}
			t := pb.DeletedAt.AsTime()
			return &t
		}(),
	}
}

func (c Comment) ToProto() *apiv1.Comment {
	return &apiv1.Comment{
		Id: func() string {
			if c.ID.IsZero() {
				return ""
			}
			return c.ID.Hex()
		}(),
		Author:    c.Author,
		Content:   c.Content,
		CreatedAt: timestamppb.New(c.CreatedAt),
	}
}

func CommentFromProto(pb *apiv1.Comment) Comment {
	if pb == nil {
		return Comment{}
	}
	id, _ := bson.ObjectIDFromHex(pb.Id)
	return Comment{
		ID:        id,
		Author:    pb.Author,
		Content:   pb.Content,
		CreatedAt: pb.CreatedAt.AsTime(),
	}
}
