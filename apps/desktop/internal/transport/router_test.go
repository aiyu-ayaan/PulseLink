package transport

import (
	"context"
	"testing"

	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/protocol"
)

func TestRouterDispatchSuccess(t *testing.T) {
	r := NewRouter()
	r.Register("media", HandlerFunc(func(_ context.Context, req protocol.Envelope) (any, error) {
		return map[string]string{"state": "playing"}, nil
	}))

	req, _ := protocol.NewRequest("1", "media", "play", nil)
	resp := r.Dispatch(context.Background(), req)

	if resp.Type != protocol.TypeResponse || resp.Error != nil {
		t.Fatalf("unexpected: %+v", resp)
	}
	var out struct{ State string }
	_ = resp.Bind(&out)
	if out.State != "playing" {
		t.Fatalf("got %+v", out)
	}
}

func TestRouterUnknownCapability(t *testing.T) {
	r := NewRouter()
	req, _ := protocol.NewRequest("1", "ghost", "x", nil)
	resp := r.Dispatch(context.Background(), req)
	if resp.Error == nil || resp.Error.Code != protocol.CodeUnsupported {
		t.Fatalf("want unsupported, got %+v", resp.Error)
	}
}

func TestRouterHandlerError(t *testing.T) {
	r := NewRouter()
	r.Register("x", HandlerFunc(func(_ context.Context, _ protocol.Envelope) (any, error) {
		return nil, &protocol.Error{Code: protocol.CodeNotFound, Message: "nope"}
	}))
	req, _ := protocol.NewRequest("1", "x", "y", nil)
	resp := r.Dispatch(context.Background(), req)
	if resp.Error == nil || resp.Error.Code != protocol.CodeNotFound {
		t.Fatalf("want not_found, got %+v", resp.Error)
	}
}
