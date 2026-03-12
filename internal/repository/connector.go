package repository

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// DBConnector encapsulates connections to both relational and document stores.
type DBConnector interface {
	SQL() *sql.DB
	Mongo() *mongo.Database
	Close() error
}

type defaultConnector struct {
	sqlDb   *sql.DB
	mongoDb *mongo.Database
	mongoC  *mongo.Client
}

func (c *defaultConnector) SQL() *sql.DB {
	return c.sqlDb
}

func (c *defaultConnector) Mongo() *mongo.Database {
	return c.mongoDb
}

func (c *defaultConnector) Close() error {
	var errs []error
	if c.sqlDb != nil {
		if err := c.sqlDb.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if c.mongoC != nil {
		if err := c.mongoC.Disconnect(context.Background()); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("failed to close connections: %v", errs)
	}
	return nil
}

// Config holds connection parameters.
type Config struct {
	PostgresDSN string
	MongoURI    string
	MongoDBName string
}

// NewConnector creates a new DBConnector instance.
func NewConnector(ctx context.Context, cfg Config) (DBConnector, error) {
	connector := &defaultConnector{}

	if cfg.PostgresDSN != "" {
		sqlDb, err := sql.Open("pgx", cfg.PostgresDSN)
		if err != nil {
			return nil, fmt.Errorf("failed to open postgres: %w", err)
		}
		if err := sqlDb.PingContext(ctx); err != nil {
			return nil, fmt.Errorf("failed to ping postgres: %w", err)
		}
		connector.sqlDb = sqlDb
	}

	if cfg.MongoURI != "" && cfg.MongoDBName != "" {
		clientOpt := options.Client().ApplyURI(cfg.MongoURI)
		mongoC, err := mongo.Connect(ctx, clientOpt)
		if err != nil {
			return nil, fmt.Errorf("failed to connect mongo: %w", err)
		}
		if err := mongoC.Ping(ctx, nil); err != nil {
			return nil, fmt.Errorf("failed to ping mongo: %w", err)
		}
		connector.mongoC = mongoC
		connector.mongoDb = mongoC.Database(cfg.MongoDBName)
	}

	return connector, nil
}
