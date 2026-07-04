# PulseLink Development Progress 📈

Tracks implementation progress. Newest work is at the top of each section so a
fresh session can catch up fast.

---

## 🚦 Overall Status

| Stage | Scope | Status |
|-------|-------|--------|
| 1 | Desktop backend (Go services, WS server, storage) | ✅ Done |
| 2 (old) | First-pass React UI | ⚠️ Superseded by Stage A rebuild |
| **A** | **Desktop UI rebuilt as Windows 11 Fluent/Mica** | ✅ Done + verified here |
| **B** | **Real Wails v3 native window** | 🟦 Code complete — user must build/verify |
| **C** | **Android companion MVP** | ⏳ Next |

Design spec for A/B/C: `docs/superpowers/specs/2026-07-04-pulselink-desktop-android-mvp-design.md`

---

## 🟩 Stage A — Desktop UI rebuild (DONE)

Replaced the monolithic ~1300-line `App.tsx` (neon-green "AI slop") with a real
Windows 11 **Fluent / Mica** design system and componentized panels.

- **Design system** (`src/index.css`): Windows accent **blue** (not green),
  Segoe UI Variable, translucent mica surfaces, **light + dark** themes via
  `data-theme`, Fluent slider/scrollbar/focus styling.
- **`src/lib/backend.tsx`** — `BackendProvider` + `useBackend()` centralizes the
  WebSocket protocol, connection, polling, and shared state.
- **`src/components/`** — `ui.tsx` (Card, Button, Toggle, StatTile, Meter,
  Badge, Field), `Sidebar.tsx`.
- **`src/panels/`** — Dashboard, MediaVolume, Brightness, Devices, Clipboard,
  Notifications, Apps, Logs, Settings.
- **Devices panel renders a REAL scannable QR** (`qrcode` dep) encoding
  `pulselink://pair?host=&port=&token=&name=` — replaced the fake CSS-grid QR
  and the fabricated device list.
- **Verified**: `npm run build` clean (0 type errors); headless Chrome
  screenshot reviewed — genuine Fluent dashboard.

## 🟦 Stage B — Wails v3 native window (code complete, user-verified)

Turns the "web app served by a headless daemon" into a real native Windows app.

- **`apps/desktop/main.go`** (`//go:build wails`): native WebView2 window that
  embeds the built frontend and runs the backend **in-process**.
- **`apps/desktop/stub.go`** (`//go:build !wails`): keeps `go build/test ./...`
  green without the Wails dep. Real app: `go build -tags wails ./apps/desktop`.
- Loopback UI runs **plain ws** (WebView2 serves `http://`), so no cert trust
  needed; `main.go` sets `EnableTLS=false`. Self-signed TLS code stays for later.
- **`docs/desktop-app.md`** — toolchain + build steps.
- ⚠️ **Blocker for the user**: dev machine has 32-bit Go (`windows/386`); Wails
  needs **64-bit Go + gcc + WebView2**. After installing: `npm run build` →
  `go get github.com/wailsapp/wails/v3@latest && go mod tidy` →
  `go build -tags wails -o pulselink.exe ./apps/desktop`.
- ⚠️ The exact Wails v3 window API may need minor adjustment for the pinned
  release (couldn't be compiled here without the toolchain).

## ⏳ Stage C — Android companion MVP (NEXT)

Kotlin + Compose + Material 3, MVVM, under `apps/android/` (currently just
`.gitkeep`s). Planned:
- Gradle/Compose skeleton (version catalog; Ktor, Room, CameraX + ML Kit).
- **Connect screen**: manual `host:port` **and** QR scan (decodes the
  `pulselink://pair` URI the desktop shows).
- **Ktor WebSocket client** speaking the protocol below (ClientHello →
  ServerWelcome → request/response/event).
- **Control screen**: media transport, volume slider + mute, power
  (lock/sleep/restart/shutdown), live sysinfo tiles.
- **Room** persists the last paired PC; basic reconnect.
- MVP uses **plain ws** (matches Stage B); TLS + token enforcement are follow-ups.

---

## 🔌 Protocol reference (backend is done — clients just speak this)

JSON WebSocket at `/ws`. Envelope:
`{ id, type(request|response|event), capability, action, payload?, error? }`

- Handshake: `handshake/hello` (ClientHello) → `handshake/welcome` (ServerWelcome).
- `media`: play_pause · next · previous · stop
- `volume`: up · down · mute · get · set `{level}`
- `power`: lock · sleep · restart · shutdown
- `sysinfo`: get → `{hostname, os, cpuUsage, ramTotal, ramFree, batteryPct, isCharging, monitorCount}`
- `brightness`: get · set `{type: internal|external, level}`
- `clipboard`: get · set `{text}` · `changed` (event)
- `apps`: launch `{name}` · `notification`: toast `{title, message}`
- `settings`: get · set (config.json)
- Auth is **AllowAll** (dev) — token carried but not enforced yet.

---

## 📈 Verification

- Backend: `go build ./...` ✅ · `go test ./...` ✅ (vet has pre-existing
  `unsafe.Pointer` notes in `clipboard_windows.go`).
- Frontend: `cd apps/desktop/frontend && npm run build` ✅ (0 type errors).
- Wails native app: **user builds** with `-tags wails` after toolchain setup.
- Android: **user builds** in Android Studio (Stage C).
