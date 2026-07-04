# PulseLink — Real Desktop App + Android MVP (Design)

Date: 2026-07-04
Status: Approved for planning

## Problem

Today "PulseLink desktop" is a headless Go daemon (`pulselinkd`) that serves the
React `dist/` over the LAN — there is no native window, `wails.json` is a 14-byte
placeholder, and `apps/android/` is empty (`.gitkeep` only). The current React UI
is a single ~800-line `App.tsx` that looks unfinished. The user wants: a real
desktop application, a rebuilt UI, and an Android companion app that connects and
controls the PC.

## What already exists and is reused as-is

The Go backend and its JSON WebSocket protocol are done and stay untouched:

- **Handshake**: client sends `handshake/hello` (`ClientHello`), server replies
  `handshake/welcome` (`ServerWelcome`) with negotiated capabilities.
- **Envelope**: `{ id, type(request|response|event), capability, action, payload?, error? }`.
- **Live capabilities/actions**:
  - `media`: `play_pause`, `next`, `previous`, `stop`
  - `volume`: `up`, `down`, `mute`, `get`, `set`
  - `power`: `lock`, `sleep`, `restart`, `shutdown`
  - `sysinfo`: status (CPU/RAM/battery)
  - (also present: brightness, clipboard, apps, input, notification, filetransfer, settings)
- **Auth**: `AllowAll` (dev) — no token is enforced yet. "Pairing" in this MVP is
  therefore *discover the PC and connect*; a token is carried end-to-end but not
  yet validated server-side. Enforcing it is out of scope here (follow-up).

Design principle: **the UI and Android are both just clients of this server.** No
protocol or service changes in this work.

## Scope (three stages, each independently committed)

### Stage A — Rebuild desktop UI (`/ui-ux-pro-max`)
- Replace monolithic `App.tsx` with componentized panels (dashboard, media/volume,
  power, devices/QR, sysinfo, logs, settings) under `frontend/src/`.
- Visual direction: **Windows 11 Fluent / Mica** — acrylic/mica surfaces, subtle
  depth, system-consistent, dark+light aware.
- Comms unchanged: keeps connecting to the local WS server exactly as now.
- **Verifiable in this session**: `npm run build` clean + Vite dev server screenshot.

### Stage B — Real Wails v3 native window
- Add a Wails v3 application rooted at `apps/desktop` with its own `main.go` and a
  real `wails.json`.
- On startup it **starts the existing backend in-process** (`internal/app`: services,
  hub, router, TLS WebSocket server) and opens a native window hosting the embedded
  Stage-A frontend.
- The desktop UI continues to reach the backend over `localhost` WebSocket (no comms
  rewrite). Android reaches the same server over the LAN.
- `main.go` at repo root stays a signpost or is updated to point at the Wails app.
- **Verified by the user**: requires 64-bit Go + gcc (cgo) + WebView2 runtime +
  `wails` CLI. Claude cannot install these or run the build here; Claude provides the
  code and exact setup/build commands.

### Stage C — Android MVP (`/ui-ux-pro-max` for mobile UI)
Kotlin + Jetpack Compose + Material 3, MVVM, under `apps/android/`.
- **Gradle/Compose project skeleton** (was only `.gitkeep`s): app module + minimal
  modules, version catalog, Kotlin/Compose/Ktor/Room/CameraX+ML Kit deps.
- **Connect screen**: manual `host:port` entry **and** QR scanning (CameraX + ML Kit
  Barcode). Desktop already shows a QR card; it encodes the `wss://host:port/ws`
  (+ token) URL.
- **WebSocket client** (Ktor): opens the socket, sends `ClientHello`, parses
  `ServerWelcome`, then the request/response/event envelope. TLS: the desktop uses a
  self-signed cert, so the client trusts it (dev: permissive trust for LAN cert;
  documented as a known dev shortcut).
- **Control screen**: media transport (play_pause/next/previous/stop), volume slider
  + mute (`volume/set` + `up`/`down`/`mute`), power buttons (lock/sleep/restart/
  shutdown with confirm), live sysinfo tiles.
- **Room**: persist last paired PC (name, host, port, token) for auto-reconnect.
- **Reconnect**: basic retry on drop.
- **Verified by the user**: Android Studio + device/emulator. Claude provides code;
  cannot run an emulator here.

## Out of scope (explicit)
- Real pairing/token enforcement server-side (stays `AllowAll`).
- Full feature parity on Android (clipboard/brightness/notifications/file transfer/
  input) — MVP is media/volume/power/sysinfo.
- Protobuf migration, screen mirroring, remote input on mobile.

## Verification per stage
- A: `cd apps/desktop/frontend && npm run build` clean; dev-server screenshot reviewed.
- B: user runs `wails build` / `wails dev` after installing toolchain; native window
  opens and UI connects. (`go build ./...` still passes for the non-Wails packages.)
- C: user runs the app from Android Studio; connects to desktop over LAN; a media/
  volume/power command visibly affects the PC.

## Risks
- **Toolchain (B)**: installed Go is `windows/386`; Wails needs 64-bit + cgo. User
  must upgrade — hard blocker Claude cannot clear.
- **Self-signed TLS (C)**: Android must trust the LAN cert; permissive trust is a
  documented dev-only shortcut, not shipped as-is.
- **Cannot verify B/C in-session**: code is written to compile/run logically but the
  user is the verification loop for the native window and the phone.

## Commit plan
1. Stage A UI rebuild (may be a few commits: scaffold components, wire, polish).
2. Stage B Wails shell.
3. Stage C Android MVP (skeleton commit, then features).
CodeGraph run before/after each stage to scan changes.
