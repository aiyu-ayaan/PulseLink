package protocol

// Handshake capability/action names. The first exchange after a socket opens is
// a ClientHello request answered by a ServerWelcome.
const (
	CapHandshake  = "handshake"
	ActionHello   = "hello"
	ActionWelcome = "welcome"
)

// ClientHello is the payload a client sends to open a session.
type ClientHello struct {
	ProtocolVersion string   `json:"protocolVersion"`
	DeviceID        string   `json:"deviceId"`
	DeviceName      string   `json:"deviceName"`
	AppVersion      string   `json:"appVersion"`
	Token           string   `json:"token"`        // pairing or session token
	Capabilities    []string `json:"capabilities"` // what the client can do
}

// ServerWelcome is the backend's reply, negotiating the shared capability set.
type ServerWelcome struct {
	ProtocolVersion string   `json:"protocolVersion"`
	ServerName      string   `json:"serverName"`
	ServerVersion   string   `json:"serverVersion"`
	// Capabilities is the intersection of what the server offers and, when the
	// device is known, what it is permitted to use.
	Capabilities []string `json:"capabilities"`
	// Accepted is false when auth failed; Reason explains why.
	Accepted bool   `json:"accepted"`
	Reason   string `json:"reason,omitempty"`
}

// Negotiate returns the capabilities present in both offered and requested,
// preserving offered's order. Passing a nil requested means "all offered".
func Negotiate(offered, requested []string) []string {
	if requested == nil {
		out := make([]string, len(offered))
		copy(out, offered)
		return out
	}
	want := make(map[string]bool, len(requested))
	for _, c := range requested {
		want[c] = true
	}
	var out []string
	for _, c := range offered {
		if want[c] {
			out = append(out, c)
		}
	}
	return out
}
