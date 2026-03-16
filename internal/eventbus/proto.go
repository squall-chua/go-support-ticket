package eventbus

import (
	"encoding/json"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// ProtoMarshaler wraps a protobuf message to implement json.Marshaler.
// This ensures that when the publisher marshals the event data, it uses
// protojson to correctly handle protobuf fields.
type ProtoMarshaler struct {
	proto.Message
}

func (w ProtoMarshaler) MarshalJSON() ([]byte, error) {
	return protojson.Marshal(w.Message)
}

// UnmarshalPayload unmarshals event data (which might be a map[string]interface{})
// into a protobuf message using protojson.
func UnmarshalPayload(data interface{}, msg proto.Message) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return protojson.Unmarshal(b, msg)
}
