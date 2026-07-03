package protocol

import (
	"reflect"
	"testing"
)

func TestRequestResponseRoundTrip(t *testing.T) {
	req, err := NewRequest("42", "media", "seek", map[string]int{"position": 30})
	if err != nil {
		t.Fatal(err)
	}
	data, err := Encode(req)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "42" || got.Capability != "media" || got.Action != "seek" {
		t.Fatalf("bad decode: %+v", got)
	}
	var payload struct{ Position int }
	if err := got.Bind(&payload); err != nil || payload.Position != 30 {
		t.Fatalf("payload %+v err %v", payload, err)
	}

	resp := NewErrorResponse(req, CodeUnsupported, "no media session")
	if resp.ID != "42" || resp.Error.Code != CodeUnsupported {
		t.Fatalf("bad error response: %+v", resp)
	}
}

func TestValidate(t *testing.T) {
	cases := []Envelope{
		{Type: "bogus", Capability: "x", Action: "y", ID: "1"},
		{Type: TypeRequest, Action: "y", ID: "1"},         // no capability
		{Type: TypeRequest, Capability: "x", ID: "1"},     // no action
		{Type: TypeRequest, Capability: "x", Action: "y"}, // no id
	}
	for i, e := range cases {
		if err := e.Validate(); err == nil {
			t.Errorf("case %d: expected validation error", i)
		}
	}
	// events need no ID
	ev := Envelope{Type: TypeEvent, Capability: "battery", Action: "changed"}
	if err := ev.Validate(); err != nil {
		t.Errorf("event should validate: %v", err)
	}
}

func TestNegotiate(t *testing.T) {
	offered := []string{"media", "brightness", "clipboard"}
	got := Negotiate(offered, []string{"clipboard", "media", "unknown"})
	want := []string{"media", "clipboard"} // offered order preserved
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
	if all := Negotiate(offered, nil); !reflect.DeepEqual(all, offered) {
		t.Fatalf("nil requested should return all offered, got %v", all)
	}
}
