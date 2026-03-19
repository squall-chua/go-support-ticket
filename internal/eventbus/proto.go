package eventbus

import (
	"encoding/json"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

// ProtoMarshaler wraps a protobuf message to implement json and bson interfaces.
type ProtoMarshaler struct {
	proto.Message
}

func (w ProtoMarshaler) MarshalJSON() ([]byte, error) {
	if w.Message == nil {
		return []byte("null"), nil
	}
	return protojson.Marshal(w.Message)
}

func (w *ProtoMarshaler) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		w.Message = nil
		return nil
	}
	// By default, we unmarshal into anypb.Any if no message is set.
	// This supports the frequent case of typed polymorphic fields.
	if w.Message == nil {
		w.Message = &anypb.Any{}
	}
	return protojson.Unmarshal(b, w.Message)
}

// UnmarshalNew unmarshals the encapsulated message into its concrete type,
// assuming the encapsulated message is an anypb.Any.
func (w ProtoMarshaler) UnmarshalNew() (proto.Message, error) {
	if w.Message == nil {
		return nil, nil
	}
	anyData, ok := w.Message.(*anypb.Any)
	if !ok {
		return nil, nil
	}
	return anypb.UnmarshalNew(anyData, proto.UnmarshalOptions{})
}

// UnmarshalTo unmarshals the encapsulated Any into the provided proto message.
func (w ProtoMarshaler) UnmarshalTo(msg proto.Message) error {
	if w.Message == nil {
		return nil
	}
	anyData, ok := w.Message.(*anypb.Any)
	if !ok {
		return nil
	}
	return anypb.UnmarshalTo(anyData, msg, proto.UnmarshalOptions{})
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
