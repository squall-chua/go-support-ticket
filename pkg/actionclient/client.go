package actionclient

import (
	"context"
	"fmt"
	"time"

	apiv1 "github.com/squall-chua/go-support-ticket/api/v1"
)

// Dispatcher represents a client that talks to external systems to perform actions.
type Dispatcher interface {
	Dispatch(ctx context.Context, actionType string, params map[string]string) (map[string]string, error)
	GetSchemas(ctx context.Context) ([]*apiv1.ActionSchema, error)
}

// MockDispatcher provides a simulated external systems client.
type MockDispatcher struct{}

// NewMockDispatcher creates a new simulated dispatcher.
func NewMockDispatcher() *MockDispatcher {
	return &MockDispatcher{}
}

// GetSchemas returns the hardcoded action schemas.
func (c *MockDispatcher) GetSchemas(ctx context.Context) ([]*apiv1.ActionSchema, error) {
	return []*apiv1.ActionSchema{
		{
			Id:               "github_issue",
			ActionType:       "create_github_issue",
			DisplayName:      "Create GitHub Issue",
			Description:      "Creates a bug report or feature request in the target GitHub repository.",
			RequireApproval:  false,
			Parameters: []*apiv1.ActionParameter{
				{Name: "repo", Type: apiv1.FieldType_FIELD_TYPE_STRING, Required: true, Description: "Repository name (owner/repo)"},
				{Name: "title", Type: apiv1.FieldType_FIELD_TYPE_STRING, Required: true, Description: "Issue title"},
				{Name: "body", Type: apiv1.FieldType_FIELD_TYPE_STRING, Required: false, Description: "Issue description"},
			},
		},
		{
			Id:               "refund_stripe",
			ActionType:       "issue_refund",
			DisplayName:      "Issue Stripe Refund",
			Description:      "Issues a refund for a specific transaction in Stripe.",
			RequireApproval:  true,
			Parameters: []*apiv1.ActionParameter{
				{Name: "charge_id", Type: apiv1.FieldType_FIELD_TYPE_STRING, Required: true, Description: "Stripe Charge ID"},
				{Name: "amount", Type: apiv1.FieldType_FIELD_TYPE_NUMBER, Required: true, Description: "Amount to refund in cents"},
				{Name: "reason", Type: apiv1.FieldType_FIELD_TYPE_ENUM, Required: false, EnumValues: []string{"duplicate", "fraudulent", "requested_by_customer"}},
			},
		},
	}, nil
}

// Dispatch executes the action against an external system.
func (c *MockDispatcher) Dispatch(ctx context.Context, actionType string, params map[string]string) (map[string]string, error) {
	// Simulate network latency
	select {
	case <-time.After(100 * time.Millisecond):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	results := make(map[string]string)

	switch actionType {
	case "create_github_issue":
		repo := params["repo"]
		if repo == "" {
			return nil, fmt.Errorf("missing required parameter: repo")
		}
		results["issue_url"] = fmt.Sprintf("https://github.com/%s/issues/123", repo)
		results["issue_number"] = "123"

	case "issue_refund":
		chargeID := params["charge_id"]
		if chargeID == "" {
			return nil, fmt.Errorf("missing required parameter: charge_id")
		}
		results["refund_id"] = "re_" + chargeID
		results["status"] = "succeeded"

	default:
		return nil, fmt.Errorf("unknown action type: %s", actionType)
	}

	return results, nil
}
