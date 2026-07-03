package transport

import "github.com/aiyu-ayaan/pulselink/apps/desktop/internal/protocol"

// Authenticator validates a client's opening handshake. It is implemented by
// the auth package so transport stays free of storage/pairing dependencies.
type Authenticator interface {
	// Authenticate resolves a ClientHello into an AuthResult. A non-nil error
	// is an internal failure; a denied-but-handled auth sets Accepted=false.
	Authenticate(hello protocol.ClientHello) (AuthResult, error)
}

// AuthResult is the outcome of authenticating a handshake.
type AuthResult struct {
	Accepted   bool
	Reason     string // set when Accepted is false
	DeviceID   string
	DeviceName string
	// AllowedCapabilities restricts what the device may use. Nil means "no
	// restriction" (all server capabilities are offered).
	AllowedCapabilities []string
}

// AllowAll is an Authenticator that accepts every connection. It exists for
// local development and tests; production wiring uses the auth package.
type AllowAll struct{}

// Authenticate always accepts.
func (AllowAll) Authenticate(hello protocol.ClientHello) (AuthResult, error) {
	return AuthResult{
		Accepted:   true,
		DeviceID:   hello.DeviceID,
		DeviceName: hello.DeviceName,
	}, nil
}
