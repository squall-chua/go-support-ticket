# System Design: Go Support Ticket

## Overview

`go-support-ticket` is a robust and scalable support ticket and workflow management system built in Go. It handles the lifecycle of support tickets alongside executing predefined actions that may require an approval state machine. The system adopts modern, cloud-native patterns with structured event-driven communication and multiplexed gRPC and HTTP access.

## Architecture

The system leans on hexagonal/clean architecture principles, cleanly separating core business contexts into domain-focused services: **Ticket**, **Action**, **Approval**, and **Audit**. 

Persistence is achieved using MongoDB as a document-based store for flexible schemas. Asynchronous decoupling across bounded contexts logic runs over an event broker.

### High-Level Architecture Diagram

```mermaid
graph TD
    Client[Client Apps / Users] -->|HTTP REST / gRPC| API[Grpc-Gateway & gRPC Multiplexer]
    
    subgraph Services Layer
        API --> TS[Ticket Service]
        API --> AS[Action Service]
        API --> AppS[Approval Service]
        API --> AudS[Audit Service]
    end
    
    subgraph Infrastructure
        TS -.-> DB[(MongoDB)]
        AS -.-> DB
        AppS -.-> DB
        AudS -.-> DB
    end
    
    subgraph Event Driven Architecture
        TS -- Publishes/Subscribes --> EB((Message Broker / Kafka))
        AS -- Publishes/Subscribes --> EB
        AppS -- Publishes/Subscribes --> EB
    end
```

## Core Components

The application divides its domain into 4 main bounded contexts implemented as distinct gRPC services exposed on the same API server:

1. **Ticket Service (`TicketServiceServer`)**
   Manages the primary workflows and lifecycles of support tickets. Provides operations like creating/updating tickets, managing comments, merging, and distributing tickets. It delegates sensitive updates to an approval channel by broadcasting specific events (`ticket.update.pending_approval`, `ticket.merge.pending_approval`).
2. **Action Service (`ActionServiceServer`)**
   Executes deterministic automated actions defined by `ActionSchema`. It interacts with the rest of the application ecosystem. Executions that need peer review submit an `action.execution.pending_approval` event. 
3. **Approval Service (`ApprovalServiceServer`)**
   Maintains the state machine for an approval flow. Driven by configurations (`ApprovalConfigs`), it orchestrates reviewers answering "Approve" or "Reject". Subscribes to events raised by other services, and publishes back definitive `approval.decided` decisions.
4. **Audit Service (`AuditServiceServer`)**
   Tracks and reads the log trail of events happening across the platform for compliance. Read-only API allowing tracking who did what, and what triggered it.

## Communication & Process Flow

Communication leverages synchronous operations (gRPC + REST via HTTP multiplexing) for requests that can be completed immediately. For multi-step sequences like "Pending Approvals", the system decouples producers and consumers using a Pub/Sub mechanism (e.g., Kafka).

### Example Workflow: Approval Process State Machine

When a sensitive action or ticket update is requested, it starts an asynchronous approval process instead of being finalized instantly.

```mermaid
sequenceDiagram
    participant C as Client
    participant TS as Service (Ticket/Action)
    participant EB as Event Broker
    participant AppS as Approval Service
    participant R as Reviewer

    C->>TS: Update Ticket / Execute Action
    TS->>EB: Publish pending_approval Event (e.g., ticket.update.pending_approval)
    TS-->>C: Return 202 (Pending Status)
    
    EB->>AppS: Consume pending_approval Event
    AppS->>AppS: Initiate Request & Evaluate Policies
    
    R->>AppS: Review & Submit (DecideApproval)
    AppS->>EB: Publish approval.decided
    
    EB->>TS: Consume approval.decided
    TS->>TS: Finalize/Abort Original Operation
    TS->>EB: Publish execution completed event
```

### Supported Event Types

- **Approval triggers**: `ticket.update.pending_approval`, `ticket.merge.pending_approval`, `action.execution.pending_approval`
- **Approval decisions**: `approval.decided`
- **Action execution tracking**: `action.execution.triggered`, `action.execution.executed`, `action.execution.completed`, `action.execution.status.updated`
- **Ticket lifecycle events**: `ticket.created`, `ticket.updated`, `ticket.assigned`, `ticket.merged`, `ticket.comment.added`, `ticket.deleted`
- **Config & Meta state**: `ticket_type.*`, `action.schema.*`, `approval_config.*`

## Domain Model

```mermaid
erDiagram
    TICKET ||--o{ TICKET_TYPE : "categorized by"
    TICKET ||--o{ COMMENT : "contains"
    ACTION_SCHEMA ||--o{ ACTION_EXECUTION : "defines"
    APPROVAL_CONFIG ||--o{ APPROVAL : "rules for"
    
    TICKET {
        string ID
        string Title
        string Status
        string AssignedTo
    }
    
    ACTION_SCHEMA {
        string SchemaID
        string Description
    }

    ACTION_EXECUTION {
        string ID
        string SchemaID
        string Status
    }
    
    APPROVAL {
        string ID
        string Status
        string RequestedBy
        string DecidedBy
    }
```

The system is explicitly designed for horizontal scaling across nodes, as asynchronous jobs and web server requests act statelessly and use MongoDB to lock state context.
