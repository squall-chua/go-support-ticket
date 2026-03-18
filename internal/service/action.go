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
	}

	if schema.RequireApproval {
		execution.Status = int32(apiv1.ActionStatus_ACTION_STATUS_PENDING_APPROVAL)
	} else {
		execution.Status = int32(apiv1.ActionStatus_ACTION_STATUS_IN_PROGRESS)
	}

	if err := s.repo.CreateExecution(ctx, execution); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	eventType := eventconsts.ActionExecutionTriggered
	if schema.RequireApproval {
		eventType = eventconsts.ActionExecutionPending
	}

	evt := event.Event{
		EventId:    uuid.NewString(),
		EventType:  eventType,
		EventTime:  time.Now().UTC(),
		Source:     eventconsts.SourceAction,
		Schema:     eventconsts.SchemaAction,
		ResourceID: execution.ID.Hex(),
		Data:       eventbus.ProtoMarshaler{Message: execution.ToProto()},
	}
	_ = s.publisher.Publish(ctx, &evt)

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

func (s *ActionServiceServer) ListActionExecutions(ctx context.Context, req *apiv1.ListActionExecutionsRequest) (*apiv1.ListActionExecutionsResponse, error) {
	limit, offset, pageNumber := getPaginationParams(req.Pagination)

	filter := model.ActionExecutionFilter{
		TicketIDs:      req.TicketIds,
		ActionTypes:    req.ActionTypes,
		ExecutingUsers: req.ExecutingUsers,
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

	if len(req.Statuses) > 0 {
		statuses := make([]int32, 0, len(req.Statuses))
		for _, st := range req.Statuses {
			statuses = append(statuses, int32(st))
		}
		filter.Statuses = statuses
	}

	filter.StartTime, filter.EndTime = getTimeRange(req.TimeRange)

	executions, total, err := s.repo.ListExecutions(ctx, filter, limit, offset)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	pbExecutions := make([]*apiv1.ActionExecution, 0, len(executions))
	for _, e := range executions {
		pbExecutions = append(pbExecutions, e.ToProto())
	}

	return &apiv1.ListActionExecutionsResponse{
		Executions: pbExecutions,
		Pagination: &apiv1.PageInfo{
			TotalSize:  total,
			PageNumber: pageNumber,
		},
	}, nil
}

func (s *ActionServiceServer) CreateActionSchema(ctx context.Context, req *apiv1.CreateActionSchemaRequest) (*apiv1.CreateActionSchemaResponse, error) {
	// Check if it already exists
	existing, err := s.schema.GetSchema(ctx, req.ActionType)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if existing != nil {
		return nil, status.Error(codes.AlreadyExists, "action schema already exists")
	}

	params := make([]model.ActionParameter, 0, len(req.Parameters))
	for _, p := range req.Parameters {
		params = append(params, model.ActionParameterFromProto(p))
	}

	results := make([]model.ActionResultField, 0, len(req.ResultSchema))
	for _, f := range req.ResultSchema {
		results = append(results, model.ActionResultFieldFromProto(f))
	}

	schema := &model.ActionSchema{
		ID:              bson.NewObjectID(),
		ActionType:      req.ActionType,
		DisplayName:     req.DisplayName,
		Description:     req.Description,
		Parameters:      params,
		ResultSchema:    results,
		RequireApproval: req.RequireApproval,
	}

	if err := s.schema.CreateSchema(ctx, schema); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	evt := event.Event{
		EventId:    uuid.NewString(),
		EventType:  eventconsts.ActionSchemaCreated,
		EventTime:  time.Now().UTC(),
		Source:     eventconsts.SourceAction,
		Schema:     eventconsts.SchemaAction,
		ResourceID: schema.ActionType,
		Data:       eventbus.ProtoMarshaler{Message: schema.ToProto()},
	}
	_ = s.publisher.Publish(ctx, &evt)

	return &apiv1.CreateActionSchemaResponse{
		Schema: schema.ToProto(),
	}, nil
}

func (s *ActionServiceServer) ListActionSchemas(ctx context.Context, req *apiv1.ListActionSchemasRequest) (*apiv1.ListActionSchemasResponse, error) {
	limit, offset, pageNumber := getPaginationParams(req.Pagination)

	filter := model.ActionSchemaFilter{
		ActionTypes:     req.ActionTypes,
		DisplayName:     req.DisplayName,
		RequireApproval: req.RequireApproval,
		IncludeDeleted:  req.IncludeDeleted,
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

	filter.StartTime, filter.EndTime = getTimeRange(req.TimeRange)

	schemas, total, err := s.schema.ListSchemas(ctx, filter, limit, offset)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	pbSchemas := make([]*apiv1.ActionSchema, 0, len(schemas))
	for _, sc := range schemas {
		pbSchemas = append(pbSchemas, sc.ToProto())
	}

	return &apiv1.ListActionSchemasResponse{
		Schemas: pbSchemas,
		Pagination: &apiv1.PageInfo{
			TotalSize:  total,
			PageNumber: pageNumber,
		},
	}, nil
}

func (s *ActionServiceServer) UpdateActionSchema(ctx context.Context, req *apiv1.UpdateActionSchemaRequest) (*apiv1.UpdateActionSchemaResponse, error) {
	update := model.ActionSchemaUpdate{
		DisplayName:     req.DisplayName,
		Description:     req.Description,
		RequireApproval: req.RequireApproval,
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

	evt := event.Event{
		EventId:    uuid.NewString(),
		EventType:  eventconsts.ActionSchemaUpdated,
		EventTime:  time.Now().UTC(),
		Source:     eventconsts.SourceAction,
		Schema:     eventconsts.SchemaAction,
		ResourceID: updated.ActionType,
		Data:       eventbus.ProtoMarshaler{Message: updated.ToProto()},
	}
	_ = s.publisher.Publish(ctx, &evt)

	return &apiv1.UpdateActionSchemaResponse{
		Schema: updated.ToProto(),
	}, nil
}

func (s *ActionServiceServer) DeleteActionSchema(ctx context.Context, req *apiv1.DeleteActionSchemaRequest) (*apiv1.DeleteActionSchemaResponse, error) {
	if err := s.schema.DeleteSchema(ctx, req.ActionType); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	evt := event.Event{
		EventId:    uuid.NewString(),
		EventType:  eventconsts.ActionSchemaDeleted,
		EventTime:  time.Now().UTC(),
		Source:     eventconsts.SourceAction,
		Schema:     eventconsts.SchemaAction,
		ResourceID: req.ActionType,
	}
	_ = s.publisher.Publish(ctx, &evt)
	return &apiv1.DeleteActionSchemaResponse{}, nil
}

func (s *ActionServiceServer) RegisterHandlers(subscriber event.Subscriber) {
	subscriber.Subscribe(eventconsts.SchemaApproval, eventconsts.ApprovalDecided, s.HandleApprovalDecided)
	subscriber.Subscribe(eventconsts.SchemaAction, eventconsts.ActionExecutionExecuted, s.HandleActionExecutionExecuted)
}

func (s *ActionServiceServer) HandleApprovalDecided(ctx context.Context, evt *event.Event) error {
	var approval apiv1.ApprovalRequestData
	if err := eventbus.UnmarshalPayload(evt.Data, &approval); err != nil {
		return err
	}

	if approval.Origin != eventconsts.SourceAction {
		return nil
	}

	status := int32(apiv1.ActionStatus_ACTION_STATUS_REJECTED)
	if approval.Status == apiv1.ApprovalStatus_APPROVAL_STATUS_APPROVED {
		status = int32(apiv1.ActionStatus_ACTION_STATUS_IN_PROGRESS)
	}

	update := model.ActionExecutionUpdate{
		Status: &status,
	}

	if err := s.repo.UpdateExecution(ctx, approval.TargetId, update); err != nil {
		return err
	}

	if status == int32(apiv1.ActionStatus_ACTION_STATUS_IN_PROGRESS) {
		execution, err := s.repo.GetExecution(ctx, approval.TargetId)
		if err != nil {
			return err
		}
		if execution != nil {
			triggeredEvt := event.Event{
				EventId:    uuid.NewString(),
				EventType:  eventconsts.ActionExecutionTriggered,
				EventTime:  time.Now().UTC(),
				Source:     eventconsts.SourceAction,
				Schema:     eventconsts.SchemaAction,
				ResourceID: execution.ID.Hex(),
				Data:       eventbus.ProtoMarshaler{Message: execution.ToProto()},
			}
			_ = s.publisher.Publish(ctx, &triggeredEvt)
		}
	}

	return nil
}

func (s *ActionServiceServer) HandleActionExecutionExecuted(ctx context.Context, evt *event.Event) error {
	var result apiv1.ActionExecutionResult
	if err := eventbus.UnmarshalPayload(evt.Data, &result); err != nil {
		return err
	}

	if result.ExecutionId == "" {
		return nil
	}

	status := int32(apiv1.ActionStatus_ACTION_STATUS_COMPLETED)
	if result.Status == apiv1.ActionExecutionStatus_ACTION_EXECUTION_STATUS_FAILED {
		status = int32(apiv1.ActionStatus_ACTION_STATUS_FAILED)
	}

	modelResult := &model.ActionExecutionResult{
		Status:      int32(result.Status),
		Error:       result.Error,
		Logs:        result.Logs,
		CompletedAt: time.Now().UTC(),
	}

	if result.CompletedAt != nil {
		modelResult.CompletedAt = result.CompletedAt.AsTime()
	}

	if len(result.Results) > 0 {
		metadata := make(map[string]interface{})
		for k, v := range result.Results {
			metadata[k] = v.AsInterface()
		}
		modelResult.Metadata = metadata
	}

	update := model.ActionExecutionUpdate{
		Status: &status,
		Result: modelResult,
	}

	if err := s.repo.UpdateExecution(ctx, result.ExecutionId, update); err != nil {
		return err
	}

	// Fetch the updated execution to publish the completed event
	execution, err := s.repo.GetExecution(ctx, result.ExecutionId)
	if err != nil {
		return err
	}

	if execution != nil {
		completeEvt := event.Event{
			EventId:    uuid.NewString(),
			EventType:  eventconsts.ActionExecutionCompleted,
			EventTime:  time.Now().UTC(),
			Source:     eventconsts.SourceAction,
			Schema:     eventconsts.SchemaAction,
			ResourceID: execution.ID.Hex(),
			Data:       eventbus.ProtoMarshaler{Message: execution.ToProto()},
		}
		_ = s.publisher.Publish(ctx, &completeEvt)
	}

	return nil
}
