// Package protocol defines the wire format spoken between the desktop backend
// and mobile clients.
//
// The transport is JSON today. The envelope is deliberately flat and uses only
// scalar fields plus an opaque payload so it maps cleanly onto a Protocol
// Buffers message later: each field here becomes a proto field of the same
// name, and Payload (currently json.RawMessage) becomes `bytes`.
package protocol

import (
	"encoding/json"
	"errors"
)

// Version is the protocol version advertised during the handshake. Bump the
// major when the envelope changes incompatibly.
const Version = "1.0"

// Type discriminates the three message shapes.
type Type string

const (
	// TypeRequest is a client→server (or server→client) call expecting a
	// matching TypeResponse with the same ID.
	TypeRequest Type = "request"
	// TypeResponse answers a request; ID echoes the request's ID.
	TypeResponse Type = "response"
	// TypeEvent is an unsolicited, fire-and-forget notification.
	TypeEvent Type = "event"
)

// Envelope is the single message frame. Exactly one of Payload/Error is
// meaningful on a response.
type Envelope struct {
	// ID correlates a response with its request. Events may omit it.
	ID string `json:"id,omitempty"`
	// Type is request, response, or event.
	Type Type `json:"type"`
	// Capability names the target service, e.g. "media", "brightness".
	Capability string `json:"capability"`
	// Action is the operation within the capability, e.g. "play", "set".
	Action string `json:"action"`
	// Payload is the operation's opaque arguments or result.
	Payload json.RawMessage `json:"payload,omitempty"`
	// Error is set on a response when the operation failed.
	Error *Error `json:"error,omitempty"`
}

// Error is a structured failure returned in a response.
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

// Common error codes.
const (
	CodeBadRequest    = "bad_request"
	CodeUnauthorized  = "unauthorized"
	CodeNotFound      = "not_found"
	CodeUnsupported   = "unsupported"
	CodeInternal      = "internal"
	CodeUnimplemented = "unimplemented"
)

// NewRequest builds a request envelope. payload may be nil.
func NewRequest(id, capability, action string, payload any) (Envelope, error) {
	raw, err := marshal(payload)
	if err != nil {
		return Envelope{}, err
	}
	return Envelope{ID: id, Type: TypeRequest, Capability: capability, Action: action, Payload: raw}, nil
}

// NewResponse builds a success response echoing req's ID/capability/action.
func NewResponse(req Envelope, payload any) (Envelope, error) {
	raw, err := marshal(payload)
	if err != nil {
		return Envelope{}, err
	}
	return Envelope{ID: req.ID, Type: TypeResponse, Capability: req.Capability, Action: req.Action, Payload: raw}, nil
}

// NewErrorResponse builds a failure response echoing req's ID.
func NewErrorResponse(req Envelope, code, message string) Envelope {
	return Envelope{
		ID:         req.ID,
		Type:       TypeResponse,
		Capability: req.Capability,
		Action:     req.Action,
		Error:      &Error{Code: code, Message: message},
	}
}

// NewEvent builds an event envelope.
func NewEvent(capability, action string, payload any) (Envelope, error) {
	raw, err := marshal(payload)
	if err != nil {
		return Envelope{}, err
	}
	return Envelope{Type: TypeEvent, Capability: capability, Action: action, Payload: raw}, nil
}

// Decode parses one JSON frame and validates required fields.
func Decode(data []byte) (Envelope, error) {
	var e Envelope
	if err := json.Unmarshal(data, &e); err != nil {
		return Envelope{}, err
	}
	if err := e.Validate(); err != nil {
		return Envelope{}, err
	}
	return e, nil
}

// Encode serialises an envelope to JSON.
func Encode(e Envelope) ([]byte, error) { return json.Marshal(e) }

// Validate checks structural invariants independent of any capability.
func (e Envelope) Validate() error {
	switch e.Type {
	case TypeRequest, TypeResponse, TypeEvent:
	default:
		return errors.New("protocol: unknown message type")
	}
	if e.Capability == "" {
		return errors.New("protocol: missing capability")
	}
	if e.Action == "" {
		return errors.New("protocol: missing action")
	}
	if (e.Type == TypeRequest || e.Type == TypeResponse) && e.ID == "" {
		return errors.New("protocol: request/response requires id")
	}
	return nil
}

// Bind unmarshals the envelope payload into v.
func (e Envelope) Bind(v any) error {
	if len(e.Payload) == 0 {
		return nil
	}
	return json.Unmarshal(e.Payload, v)
}

func marshal(payload any) (json.RawMessage, error) {
	if payload == nil {
		return nil, nil
	}
	if raw, ok := payload.(json.RawMessage); ok {
		return raw, nil
	}
	return json.Marshal(payload)
}
