package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/squall-chua/go-event-pubsub/pkg/event"
	apiv1 "github.com/squall-chua/go-support-ticket/api/v1"
	"github.com/squall-chua/go-support-ticket/internal/eventbus"
	"github.com/squall-chua/go-support-ticket/internal/eventconsts"
	"github.com/squall-chua/go-support-ticket/internal/middleware"
	"github.com/squall-chua/go-support-ticket/internal/model"
	"github.com/squall-chua/go-support-ticket/internal/repository"
	"go.mongodb.org/mongo-driver/v2/bson"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ActionServiceServer implements the action service.
type ActionServiceServer struct {
	apiv1.UnimplementedActionServiceServer
	repo      repository.ActionExecutionRepository
	schema    repository.ActionSchemaRepository
	publisher event.Publisher
}

func NewActionServiceServer(
	repo repository.ActionExecutionRepository,
	schema repository.ActionSchemaRepository,
	publisher event.Publisher,
) *ActionServiceServer {
	return &ActionServiceServer{
		repo:      repo,
		schema:    schema,
		publisher: publisher,
	}
}

func (s *ActionServiceServer) ListAvailableActions(ctx context.Context, req *apiv1.ListAvailableActionsRequest) (*apiv1.ListAvailableActionsResponse, error) {
	var limit, offset, pageNumber int32 = 100, 0, 1
	if req.Pagination != nil {
		limit = req.Pagination.PageSize
		pageNumber = req.Pagination.PageNumber
		if pageNumber > 1 {
			offset = (pageNumber - 1) * limit
		} else {
			pageNumber = 1
		}
	}

	filter := model.ActionSchemaFilter{
		ActionTypes: req.ActionTypes,
		DisplayName: req.DisplayName,
	}

	if len(req.Ids) > 0 {
		oids := make([]bson.ObjectID, 0, len(req.Ids))
		for _, id := range req.Ids {
			if oid, err := bson.ObjectIDFromHex(id); err == nil {
				oids = append(oids, oid)
			}
		}
		filter.IDs = oids
	}

	if req.RequireApproval != nil {
		v := req.GetRequireApproval()
		filter.RequireApproval = &v
	}

	if req.TimeRange != nil {
		if req.TimeRange.StartTime != nil {
			t := req.TimeRange.StartTime.AsTime()
			filter.StartTime = &t
		}
		if req.TimeRange.EndTime != nil {
			t := req.TimeRange.EndTime.AsTime()
			filter.EndTime = &t
		}
	}

	schemas, total, err := s.schema.ListSchemas(ctx, filter, limit, offset)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	pbSchemas := make([]*apiv1.ActionSchema, 0, len(schemas))
	for _, sc := range schemas {
		pbSchemas = append(pbSchemas, sc.ToProto())
	}

	return &apiv1.ListAvailableActionsResponse{
		Schemas: pbSchemas,
		Pagination: &apiv1.PageInfo{
			TotalSize:  total,
			PageNumber: pageNumber,
		},
	}, nil
}

func (s *ActionServiceServer) ExecuteAction(ctx context.Context, req *apiv1.ExecuteActionRequest) (*apiv1.ActionExecution, error) {
	schema, err := s.schema.GetSchema(ctx, req.ActionType)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if schema == nil {
		return nil, status.Error(codes.NotFound, "action schema not found")
	}

	params := make(map[string]interface{})
	if req.Parameters != nil {
		b, _ := json.Marshal(req.Parameters)
		json.Unmarshal(b, &params)
	}
	userID, _ := middleware.UserFromContext(ctx)

	execution := &model.ActionExecution{
		ID:            bson.NewObjectID(),
		TicketID:      req.TicketId,
		ActionType:    req.ActionType,
		Parameters:    params,
		ExecutingUser: userID,
		Status:        int32(apiv1.ActionStatus_ACTION_STATUS_PENDING),
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}

	if schema.RequireApproval {
		execution.Status = int32(apiv1.ActionStatus_ACTION_STATUS_PENDING_APPROVAL)
	} else {
		execution.Status = int32(apiv1.ActionStatus_ACTION_STATUS_IN_PROGRESS)
	}

	if err := s.repo.CreateExecution(ctx, execution); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if schema.RequireApproval {
		evt := event.Event{
			EventId:    uuid.NewString(),
			EventType:  eventconsts.ActionExecutionPending,
			EventTime:  time.Now().UTC(),
			Source:     eventconsts.SourceSupportTicket,
			Schema:     eventconsts.SchemaAction,
			ResourceID: execution.ID.Hex(),
			Data:       eventbus.ProtoMarshaler{Message: execution.ToProto()},
		}
		_ = s.publisher.Publish(ctx, &evt)
	}

	return execution.ToProto(), nil
}

func (s *ActionServiceServer) GetActionExecution(ctx context.Context, req *apiv1.GetActionExecutionRequest) (*apiv1.ActionExecution, error) {
	execution, err := s.repo.GetExecution(ctx, req.ExecutionId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if execution == nil {
		return nil, status.Error(codes.NotFound, "execution not found")
	}
	return execution.ToProto(), nil
}

func (s *ActionServiceServer) UpdateActionSchema(ctx context.Context, req *apiv1.UpdateActionSchemaRequest) (*apiv1.UpdateActionSchemaResponse, error) {
	update := model.ActionSchemaUpdate{
		DisplayName:     req.DisplayName,
		Description:     req.Description,
		RequireApproval: req.RequireApproval,
		UpdatedAt:       time.Now().UTC(),
	}

	if len(req.Parameters) > 0 {
		params := make([]model.ActionParameter, 0, len(req.Parameters))
		for _, p := range req.Parameters {
			params = append(params, model.ActionParameterFromProto(p))
		}
		update.Parameters = &params
	}

	if len(req.ResultSchema) > 0 {
		results := make([]model.ActionResultField, 0, len(req.ResultSchema))
		for _, f := range req.ResultSchema {
			results = append(results, model.ActionResultFieldFromProto(f))
		}
		update.ResultSchema = &results
	}

	if err := s.schema.UpdateSchema(ctx, req.ActionType, update); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Fetch updated schema to return
	updated, err := s.schema.GetSchema(ctx, req.ActionType)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if updated == nil {
		return nil, status.Error(codes.NotFound, "action schema not found after update")
	}

	return &apiv1.UpdateActionSchemaResponse{
		Schema: updated.ToProto(),
	}, nil
}

func (s *ActionServiceServer) RegisterHandlers(subscriber event.Subscriber) {
	subscriber.Subscribe(eventconsts.SchemaApproval, eventconsts.ApprovalDecided, s.HandleApprovalDecided)
}

func (s *ActionServiceServer) HandleApprovalDecided(ctx context.Context, evt *event.Event) error {
	var approval apiv1.ApprovalRequestData
	if err := eventbus.UnmarshalPayload(evt.Data, &approval); err != nil {
		return err
	}

	if approval.ExecutionId == "" {
		return nil // No execution to update
	}

	status := int32(apiv1.ActionStatus_ACTION_STATUS_REJECTED)
	if approval.Status == apiv1.ApprovalStatus_APPROVAL_STATUS_APPROVED {
		status = int32(apiv1.ActionStatus_ACTION_STATUS_IN_PROGRESS)
	}

	update := model.ActionExecutionUpdate{
		Status:    &status,
		UpdatedAt: time.Now().UTC(),
	}

	return s.repo.UpdateExecution(ctx, approval.ExecutionId, update)
}
