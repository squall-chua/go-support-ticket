package repository

import (
	"context"
	"database/sql"
	"encoding/json"

	apiv1 "github.com/squall-chua/go-support-ticket/api/v1"
)

type ActionSchemaRepository interface {
	CreateSchema(ctx context.Context, schema *apiv1.ActionSchema) error
	GetSchema(ctx context.Context, id string) (*apiv1.ActionSchema, error)
	ListSchemas(ctx context.Context) ([]*apiv1.ActionSchema, error)
}

type sqlActionSchemaRepo struct {
	db *sql.DB
}

func NewActionSchemaRepository(connector DBConnector) ActionSchemaRepository {
	return &sqlActionSchemaRepo{
		db: connector.SQL(),
	}
}

func (r *sqlActionSchemaRepo) CreateSchema(ctx context.Context, schema *apiv1.ActionSchema) error {
	paramsJSON, err := json.Marshal(schema.Parameters)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO action_schemas (id, action_type, display_name, description, require_approval, parameters)
		VALUES ($1, $2, $3, $4, $5, $6)`
	_, err = r.db.ExecContext(ctx, query,
		schema.Id,
		schema.ActionType,
		schema.DisplayName,
		schema.Description,
		schema.RequireApproval,
		paramsJSON,
	)
	return err
}

func (r *sqlActionSchemaRepo) GetSchema(ctx context.Context, id string) (*apiv1.ActionSchema, error) {
	query := `SELECT id, action_type, display_name, description, require_approval, parameters FROM action_schemas WHERE id = $1`
	row := r.db.QueryRowContext(ctx, query, id)

	var schema apiv1.ActionSchema
	var paramsJSON []byte
	err := row.Scan(&schema.Id, &schema.ActionType, &schema.DisplayName, &schema.Description, &schema.RequireApproval, &paramsJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Or return a specific error
		}
		return nil, err
	}

	if len(paramsJSON) > 0 {
		if err := json.Unmarshal(paramsJSON, &schema.Parameters); err != nil {
			return nil, err
		}
	}

	return &schema, nil
}

func (r *sqlActionSchemaRepo) ListSchemas(ctx context.Context) ([]*apiv1.ActionSchema, error) {
	query := `SELECT id, action_type, display_name, description, require_approval, parameters FROM action_schemas`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schemas []*apiv1.ActionSchema
	for rows.Next() {
		var schema apiv1.ActionSchema
		var paramsJSON []byte
		if err := rows.Scan(&schema.Id, &schema.ActionType, &schema.DisplayName, &schema.Description, &schema.RequireApproval, &paramsJSON); err != nil {
			return nil, err
		}
		if len(paramsJSON) > 0 {
			if err := json.Unmarshal(paramsJSON, &schema.Parameters); err != nil {
				return nil, err
			}
		}
		schemas = append(schemas, &schema)
	}

	return schemas, nil
}
