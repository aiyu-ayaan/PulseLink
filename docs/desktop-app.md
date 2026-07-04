# PulseLink Desktop App (Wails v3)

The desktop app is a native Windows window (WebView2) that hosts the React UI
and runs the Go backend **in-process** — one binary, no separate daemon.

The Wails entrypoint lives at `apps/desktop/main.go` behind a `wails` build tag,
so the rest of the module builds and tests without the Wails dependency. A stub
(`apps/desktop/stub.go`) stands in for the default build.

## Prerequisites (one-time)

The dev machine currently has 32-bit Go (`windows/386`); Wails needs 64-bit.

1. **64-bit Go** — install `go1.25+ windows/amd64` from https://go.dev/dl/ and
   confirm `go env GOARCH` prints `amd64`.
2. **WebView2 Runtime** — preinstalled on Windows 11. If missing, get the
   Evergreen runtime from Microsoft.
3. **C compiler** — install TDM-GCC or MSYS2 gcc and ensure `gcc` is on `PATH`
   (`gcc --version`). Set `CGO_ENABLED=1` if not already.
4. **Wails v3 CLI** (optional, for `wails3 dev` hot-reload & packaging):
   ```bash
   go install github.com/wailsapp/wails/v3/cmd/wails3@latest
   wails3 doctor   # verifies your toolchain
   ```

## Build & run

```bash
# 1. Build the frontend (embedded into the Go binary)
cd apps/desktop/frontend
npm install
npm run build

# 2. Add the Wails dependency (first time only)
cd ../../..
go get github.com/wailsapp/wails/v3@latest
go mod tidy

# 3. Build the native app
go build -tags wails -o pulselink.exe ./apps/desktop

# 4. Run
./pulselink.exe
```

For hot-reload during UI work, `wails3 dev` (needs the CLI) or run the headless
backend + Vite dev server separately:

```bash
go run ./apps/desktop/cmd/pulselinkd          # backend
cd apps/desktop/frontend && npm run dev        # UI at localhost:5173
```

## How it connects

- The window loads the embedded UI over `http://` (WebView2), so the frontend
  connects to the backend with `ws://localhost:9843/ws`.
- `main.go` forces `EnableTLS=false` in `config.json` for this reason — the
  loopback UI needs a scheme match. The self-signed TLS code
  (`internal/security`) stays for a later LAN/Android-facing secure mode.
- The Android app connects to the same server over the LAN (see the Devices
  panel QR / manual host:port).

## Notes / caveats

- The exact Wails v3 window API (`application.New`, `NewWebviewWindowWithOptions`,
  `NewRGB`) may need minor adjustment to match the pinned Wails release — run
  `wails3 doctor` and check the v3 docs if `go build -tags wails` reports an API
  mismatch.
- Mica/acrylic window backdrop is applied by the UI's faux-Mica background; a
  native Mica backdrop can be requested via Wails window options later.
