package event

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type Event struct {
	Type    string                 `json:"type"`
	Source  string                 `json:"source"`
	Payload map[string]interface{} `json:"payload"`
}

type Publisher interface {
	Publish(ctx context.Context, eventType, source string, payload map[string]interface{}) error
}

type NatsPublisher struct {
	js jetstream.JetStream
}

func NewNatsPublisher(nc *nats.Conn) (*NatsPublisher, error) {
	js, err := jetstream.New(nc)
	if err != nil {
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}
	return &NatsPublisher{js: js}, nil
}

func (p *NatsPublisher) Publish(ctx context.Context, eventType, source string, payload map[string]interface{}) error {
	evt := Event{
		Type:    eventType,
		Source:  source,
		Payload: payload,
	}

	data, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Topic structure: events.<source>.<type> -> events.system.ticket.created
	subject := fmt.Sprintf("events.%s.%s", source, eventType)
	_, err = p.js.Publish(ctx, subject, data)
	return err
}
