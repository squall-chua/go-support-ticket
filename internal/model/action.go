package model

import (
	"time"

	apiv1 "github.com/squall-chua/go-support-ticket/api/v1"
	"go.mongodb.org/mongo-driver/v2/bson"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ActionSchema struct {
	ID              bson.ObjectID       `json:"id" bson:"_id,omitempty"`
	ActionType      string              `json:"action_type" bson:"action_type"`
	DisplayName     string              `json:"display_name" bson:"display_name"`
	Description     string              `json:"description" bson:"description"`
	Parameters      []ActionParameter   `json:"parameters" bson:"parameters"`
	ResultSchema    []ActionResultField `json:"result_schema" bson:"result_schema"`
	RequireApproval bool                `json:"require_approval" bson:"require_approval"`
	CreatedAt       time.Time           `json:"created_at" bson:"created_at"`
	UpdatedAt       time.Time           `json:"updated_at" bson:"updated_at"`
}

type ActionParameter struct {
	Name         string      `json:"name" bson:"name"`
	Type         int32       `json:"type" bson:"type"`
	Required     bool        `json:"required" bson:"required"`
	EnumValues   []string    `json:"enum_values" bson:"enum_values"`
	Description  string      `json:"description" bson:"description"`
	DefaultValue interface{} `json:"default_value" bson:"default_value"`
}

type ActionResultField struct {
	Name        string              `json:"name" bson:"name"`
	Type        int32               `json:"type" bson:"type"`
	Description string              `json:"description" bson:"description"`
	Children    []ActionResultField `json:"children" bson:"children"`
}

type ActionExecution struct {
	ID             bson.ObjectID          `json:"id" bson:"_id,omitempty"`
	TicketID       string                 `json:"ticket_id" bson:"ticket_id"`
	ActionType     string                 `json:"action_type" bson:"action_type"`
	Status         int32                  `json:"status" bson:"status"`
	Parameters     map[string]interface{} `json:"parameters" bson:"parameters"`
	ResultMetadata map[string]interface{} `json:"result_metadata" bson:"result_metadata"`
	Error          string                 `json:"error" bson:"error"`
	Logs           string                 `json:"logs" bson:"logs"`
	ExecutingUser  string                 `json:"executing_user" bson:"executing_user"`
	ExecuteAt      time.Time              `json:"execute_at" bson:"execute_at"`
	CompletedAt    time.Time              `json:"completed_at" bson:"completed_at"`
	CreatedAt      time.Time              `json:"created_at" bson:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at" bson:"updated_at"`
}

func (m *ActionSchema) ToProto() *apiv1.ActionSchema {
	pb := &apiv1.ActionSchema{
		Id:              m.ID.Hex(),
		ActionType:      m.ActionType,
		DisplayName:     m.DisplayName,
		Description:     m.Description,
		RequireApproval: m.RequireApproval,
		CreatedAt:       timestamppb.New(m.CreatedAt),
		UpdatedAt:       timestamppb.New(m.UpdatedAt),
	}

	for _, p := range m.Parameters {
		pb.Parameters = append(pb.Parameters, p.ToProto())
	}

	for _, f := range m.ResultSchema {
		pb.ResultSchema = append(pb.ResultSchema, f.ToProto())
	}

	return pb
}

func ActionSchemaFromProto(pb *apiv1.ActionSchema) *ActionSchema {
	if pb == nil {
		return nil
	}
	id, _ := bson.ObjectIDFromHex(pb.Id)
	m := &ActionSchema{
		ID:              id,
		ActionType:      pb.ActionType,
		DisplayName:     pb.DisplayName,
		Description:     pb.Description,
		RequireApproval: pb.RequireApproval,
		CreatedAt:       pb.CreatedAt.AsTime(),
		UpdatedAt:       pb.UpdatedAt.AsTime(),
	}

	for _, p := range pb.Parameters {
		m.Parameters = append(m.Parameters, ActionParameterFromProto(p))
	}

	for _, f := range pb.ResultSchema {
		m.ResultSchema = append(m.ResultSchema, ActionResultFieldFromProto(f))
	}

	return m
}

func (m *ActionExecution) ToProto() *apiv1.ActionExecution {
	pb := &apiv1.ActionExecution{
		Id:            m.ID.Hex(),
		TicketId:      m.TicketID,
		ActionType:    m.ActionType,
		Status:        apiv1.ActionStatus(m.Status),
		Error:         m.Error,
		Logs:          m.Logs,
		ExecutingUser: m.ExecutingUser,
		ExecuteAt:     timestamppb.New(m.ExecuteAt),
		CompletedAt:   timestamppb.New(m.CompletedAt),
		CreatedAt:     timestamppb.New(m.CreatedAt),
		UpdatedAt:     timestamppb.New(m.UpdatedAt),
	}

	if len(m.Parameters) > 0 {
		pb.Parameters = make(map[string]*structpb.Value)
		for k, v := range m.Parameters {
			if val, err := structpb.NewValue(v); err == nil {
				pb.Parameters[k] = val
			}
		}
	}

	if len(m.ResultMetadata) > 0 {
		pb.ResultMetadata = make(map[string]*structpb.Value)
		for k, v := range m.ResultMetadata {
			if val, err := structpb.NewValue(v); err == nil {
				pb.ResultMetadata[k] = val
			}
		}
	}

	return pb
}

func ActionExecutionFromProto(pb *apiv1.ActionExecution) *ActionExecution {
	if pb == nil {
		return nil
	}
	id, _ := bson.ObjectIDFromHex(pb.Id)
	m := &ActionExecution{
		ID:            id,
		TicketID:      pb.TicketId,
		ActionType:    pb.ActionType,
		Status:        int32(pb.Status),
		Error:         pb.Error,
		Logs:          pb.Logs,
		ExecutingUser: pb.ExecutingUser,
		ExecuteAt:     pb.ExecuteAt.AsTime(),
		CompletedAt:   pb.CompletedAt.AsTime(),
		CreatedAt:     pb.CreatedAt.AsTime(),
		UpdatedAt:     pb.UpdatedAt.AsTime(),
	}

	if len(pb.Parameters) > 0 {
		m.Parameters = make(map[string]interface{})
		for k, v := range pb.Parameters {
			m.Parameters[k] = v.AsInterface()
		}
	}

	if len(pb.ResultMetadata) > 0 {
		m.ResultMetadata = make(map[string]interface{})
		for k, v := range pb.ResultMetadata {
			m.ResultMetadata[k] = v.AsInterface()
		}
	}

	return m
}
func (m ActionParameter) ToProto() *apiv1.ActionParameter {
	pb := &apiv1.ActionParameter{
		Name:        m.Name,
		Type:        apiv1.FieldType(m.Type),
		Required:    m.Required,
		EnumValues:  m.EnumValues,
		Description: m.Description,
	}
	if m.DefaultValue != nil {
		if val, err := structpb.NewValue(m.DefaultValue); err == nil {
			pb.DefaultValue = val
		}
	}
	return pb
}

func ActionParameterFromProto(pb *apiv1.ActionParameter) ActionParameter {
	if pb == nil {
		return ActionParameter{}
	}
	m := ActionParameter{
		Name:        pb.Name,
		Type:        int32(pb.Type),
		Required:    pb.Required,
		EnumValues:  pb.EnumValues,
		Description: pb.Description,
	}
	if pb.DefaultValue != nil {
		m.DefaultValue = pb.DefaultValue.AsInterface()
	}
	return m
}

func (m ActionResultField) ToProto() *apiv1.ActionResultField {
	pb := &apiv1.ActionResultField{
		Name:        m.Name,
		Type:        apiv1.FieldType(m.Type),
		Description: m.Description,
	}
	for _, child := range m.Children {
		pb.Children = append(pb.Children, child.ToProto())
	}
	return pb
}

func ActionResultFieldFromProto(pb *apiv1.ActionResultField) ActionResultField {
	if pb == nil {
		return ActionResultField{}
	}
	m := ActionResultField{
		Name:        pb.Name,
		Type:        int32(pb.Type),
		Description: pb.Description,
	}
	for _, child := range pb.Children {
		m.Children = append(m.Children, ActionResultFieldFromProto(child))
	}
	return m
}

type ActionSchemaFilter struct {
	IDs             []bson.ObjectID
	ActionTypes     []string
	DisplayName     string
	RequireApproval *bool
	StartTime       *time.Time
	EndTime         *time.Time
}

type ActionSchemaUpdate struct {
	DisplayName     *string
	Description     *string
	Parameters      *[]ActionParameter
	ResultSchema    *[]ActionResultField
	RequireApproval *bool
	UpdatedAt       time.Time
}
type ActionExecutionUpdate struct {
	Status         *int32
	ResultMetadata *map[string]interface{}
	Error          *string
	Logs           *string
	CompletedAt    *time.Time
	UpdatedAt      time.Time
}

type ActionExecutionFilter struct {
	IDs         []bson.ObjectID
	TicketIDs   []string
	ActionTypes []string
	Statuses    []int32
	StartTime   *time.Time
	EndTime     *time.Time
}
