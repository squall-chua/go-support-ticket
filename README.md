# Go Support Ticket

`go-support-ticket` is a robust, scalable support ticket and workflow management system built in Go. It empowers organizations to manage the lifecycle of user support tickets, automate predefined actions, mandate peer approvals, and maintain a compliance audit trail.

This project uses a layered, domain-driven design running over a unified gRPC and HTTP multiplexer. Communication across bounded contexts relies on an event-driven architecture using Apache Kafka and MongoDB.

For more deep-dive documentation on the system architecture, please see [DESIGN.md](./DESIGN.md).

## Features

- **Ticket Management:** Create, assign, comment, merge, and organize support tickets.
- **Workflow / Actions Engine:** Define and execute pre-configured workflow actions (e.g., granting a user a role, escalating a case).
- **Approval State Machine:** Flexible control to enforce multi-user approvals before sensitive updates or actions are committed.
- **Audit Trails:** Automatic event journaling of all sensitive operations.
- **Multiplexed API Server:** Single-port service simultaneously serving gRPC and HTTP REST (via `grpc-gateway`).
- **Telemetry & Health:** Built-in Prometheus metrics and gRPC health checks.

## Prerequisites

Before setting up the project, assure you have the following installed on your local machine:
- [Go](https://golang.org/doc/install) 1.25.1 or later
- [MongoDB](https://www.mongodb.com/) (running instance)
- [Apache Kafka](https://kafka.apache.org/quickstart) (running instance)
- [Buf CLI](https://buf.build/docs/installation) (for protobuf generation)

## Configuration / Environment Variables

The server behaves depending on the given command line flags or environment variables:

| Environment Variable | CLI Flag       | Default                                | Description |
|----------------------|----------------|----------------------------------------|-------------|
| `MONGO_URI`          | `--mongo-uri`  | `mongodb://localhost:27017/support_ticket` | MongoDB connection string. |
| `KAFKA_URL`          | `--kafka-url`  | `localhost:9092`                       | Comma-separated list of Kafka broker URLs. |
| `PORT`               | `--port`       | `8080`                                 | Port to launch the unified HTTP+gRPC server on. |
| `CORS_ORIGINS`       | `--cors-origins`| `*`                                    | List of domains allowed for CORS requests. |
| `JWT_SECRET`         | `--jwt-secret` | `secret`                               | Verification secret for authorization bearer tokens. |
| `SCHEMA_CONFIG`      | `--schema-config`| `configs/schemas.yaml`               | Path to the Kafka event schemas registry mapping file. |

## Quick Start Guide

### 1. Start Support Infrastructure
Ensure your MongoDB and Kafka servers are running locally.
```bash
# Example starting Kafka and Mongo using docker (Assuming you have a docker-compose in place):
# docker-compose up -d mongo kafka zookeeper
```

### 2. Generate Protoc Stubs (Optional)
If you modify any file inside `api/proto/v1/`, you will need to regenerate the Go bindings and OpenAPI specs. From the root directory, simply run:
```bash
./scripts/generate.sh
```

### 3. Run the Server
Use the standard go tooling to run the main server process:
```bash
go run cmd/server/main.go
```
The server will boot up and bind to `localhost:8080` by default.

## API Examples

### Authentication
Most endpoints require an authenticated JWT Token passed in the `Authorization` header.
```bash
export JWT_TOKEN="your_jwt_signing_string"
```

### 1. Creating a Support Ticket
**REST:**
```bash
curl -X POST http://localhost:8080/api/v1/tickets \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "ticket_type": "type_123",
    "title": "Cannot sync billing info",
    "description": "The billing info in the dashboard refuses to sync.",
    "priority": "TICKET_PRIORITY_HIGH",
    "customer_id": "cust_88x912"
  }'
```

### 2. Executing an Automated Action
**REST:**
```bash
curl -X POST http://localhost:8080/api/v1/actions \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "ticket_id": "ticket_abc123",
    "action_type": "restart_worker",
    "parameters": {
      "force": true
    }
  }'
```
*Depending on the configuration in the Approval Config, this will either return a completed action, or a `Status=PENDING` signaling that the action was sent for approval.*

### 3. Fetching the Audit Trail
**REST:**
```bash
curl -X GET "http://localhost:8080/api/v1/tickets/ticket_abc123/audit-trail?pagination.page_size=10" \
  -H "Authorization: Bearer $JWT_TOKEN"
```

## Architecture
The application is neatly decoupled using event streams (`squall-chua/go-event-pubsub`) to guarantee loose coupling across contexts.

## License
Refer to the `LICENSE` file for more details.
