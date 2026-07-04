package app

import (
	"context"
	"net"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/coder/websocket"

	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/config"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/protocol"
)

func TestPairingFlowOverWebSocket(t *testing.T) {
	port := freeTCPPort(t)
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	cfg := config.Default()
	cfg.Server.Host = "127.0.0.1"
	cfg.Server.Port = port
	cfg.Server.EnableTLS = false
	cfg.DatabasePath = filepath.Join(dir, "pulselink.db")
	cfg.DeviceName = "PulseLink-Test"
	if err := config.Save(cfgPath, cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}

	a, err := New(cfgPath)
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := a.Start(ctx); err != nil {
		t.Fatalf("start app: %v", err)
	}
	t.Cleanup(a.Stop)

	wsURL := "ws://127.0.0.1:" + strconv.Itoa(port) + "/ws"
	ui := dialAndHandshake(t, wsURL, protocol.ClientHello{
		ProtocolVersion: protocol.Version,
		DeviceID:        "desktop-ui",
		DeviceName:      "PulseLink Desktop",
		AppVersion:      "test",
		Token:           "desktop-local",
		Capabilities:    []string{"pairing", "devices", "sysinfo", "volume"},
	})
	defer ui.Close(websocket.StatusNormalClosure, "")

	infoResp := sendRequest(t, ui, "pairing", "info", nil)
	var info struct {
		Host   string `json:"host"`
		Port   int    `json:"port"`
		Scheme string `json:"scheme"`
		Token  string `json:"token"`
		URI    string `json:"uri"`
	}
	if err := infoResp.Bind(&info); err != nil {
		t.Fatalf("bind pairing info: %v", err)
	}
	if info.Host != "127.0.0.1" || info.Port != port || info.Scheme != "ws" || info.Token == "" || info.URI == "" {
		t.Fatalf("unexpected pairing info: %+v", info)
	}

	android := dialAndHandshake(t, wsURL, protocol.ClientHello{
		ProtocolVersion: protocol.Version,
		DeviceID:        "android-pixel",
		DeviceName:      "Pixel",
		AppVersion:      "test",
		Token:           info.Token,
		Capabilities:    []string{"pairing", "devices", "sysinfo", "volume"},
	})
	defer android.Close(websocket.StatusNormalClosure, "")

	evt := readUntil(t, ui, func(env protocol.Envelope) bool {
		return env.Type == protocol.TypeEvent && env.Capability == "pairing" && env.Action == "request"
	})
	var pending struct {
		ID string `json:"id"`
	}
	if err := evt.Bind(&pending); err != nil {
		t.Fatalf("bind pairing request: %v", err)
	}
	if pending.ID != "android-pixel" {
		t.Fatalf("unexpected pairing request: %+v", pending)
	}

	sendRequest(t, ui, "pairing", "accept", map[string]string{"deviceId": "android-pixel"})
	approved := readUntil(t, android, func(env protocol.Envelope) bool {
		return env.Type == protocol.TypeEvent && env.Capability == "pairing" && env.Action == "approved"
	})
	if approved.Capability != "pairing" {
		t.Fatalf("expected approved event, got %+v", approved)
	}
	android.Close(websocket.StatusNormalClosure, "")

	trusted := dialAndHandshake(t, wsURL, protocol.ClientHello{
		ProtocolVersion: protocol.Version,
		DeviceID:        "android-pixel",
		DeviceName:      "Pixel",
		AppVersion:      "test",
		Token:           info.Token,
		Capabilities:    []string{"pairing", "devices", "sysinfo", "volume"},
	})
	defer trusted.Close(websocket.StatusNormalClosure, "")

	sendRequest(t, trusted, "sysinfo", "get", nil)
}

func freeTCPPort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("allocate port: %v", err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

func dialAndHandshake(t *testing.T, url string, hello protocol.ClientHello) *websocket.Conn {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		t.Fatalf("dial %s: %v", url, err)
	}
	req, err := protocol.NewRequest("handshake-"+hello.DeviceID, protocol.CapHandshake, protocol.ActionHello, hello)
	if err != nil {
		t.Fatalf("handshake request: %v", err)
	}
	writeEnvelope(t, conn, req)
	env := readEnvelope(t, conn)
	var welcome protocol.ServerWelcome
	if err := env.Bind(&welcome); err != nil {
		t.Fatalf("bind welcome: %v", err)
	}
	if !welcome.Accepted {
		t.Fatalf("handshake rejected for %s: %s", hello.DeviceID, welcome.Reason)
	}
	return conn
}

func sendRequest(t *testing.T, conn *websocket.Conn, capability, action string, payload any) protocol.Envelope {
	t.Helper()
	req, err := protocol.NewRequest(capability+"-"+action, capability, action, payload)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	writeEnvelope(t, conn, req)
	resp := readUntil(t, conn, func(env protocol.Envelope) bool {
		return env.Type == protocol.TypeResponse && env.ID == req.ID
	})
	if resp.Error != nil {
		t.Fatalf("%s.%s failed: %v", capability, action, resp.Error)
	}
	return resp
}

func writeEnvelope(t *testing.T, conn *websocket.Conn, env protocol.Envelope) {
	t.Helper()
	data, err := protocol.Encode(env)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func readEnvelope(t *testing.T, conn *websocket.Conn) protocol.Envelope {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	typ, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if typ != websocket.MessageText {
		t.Fatalf("unexpected message type: %v", typ)
	}
	env, err := protocol.Decode(data)
	if err != nil {
		t.Fatalf("decode %s: %v", data, err)
	}
	return env
}

func readUntil(t *testing.T, conn *websocket.Conn, match func(protocol.Envelope) bool) protocol.Envelope {
	t.Helper()
	deadline := time.After(15 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatal("timed out waiting for websocket message")
		default:
		}
		env := readEnvelope(t, conn)
		if match(env) {
			return env
		}
	}
}
