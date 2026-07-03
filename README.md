# PulseLink 🔗

PulseLink is a modern, modular desktop companion platform that allows Android devices (and web controllers) to securely control a Windows PC over the local network (LAN). It features isolated backend services communicating over an event bus and a gorgeous Windows 11 Fluent/Mica style dark-mode control board.

---

## 🏗️ Repository Layout

- `apps/desktop/` — Go backend core structure.
  - `cmd/pulselinkd/` — Headless backend daemon runner.
  - `internal/services/` — Platform-safe Windows companion services (media, volume, brightness, clipboard, power, sysinfo, apps, input, notification, filetransfer, settings).
  - `internal/transport/` — WebSocket server, hub, and routing logic.
- `apps/desktop/frontend/` — React + TypeScript + TailwindCSS v4 + Framer Motion user interface.
- `design-system/pulselink/` — Master design system tokens and guidelines.

---

## 🛠️ Prerequisites

To run this project, make sure you have installed:
1. **Go (v1.25 or higher)**: Check with `go version`.
2. **Node.js (v20 or higher)**: Check with `node -v` and `npm -v`.

---

## 🚀 Running in Development Mode

You can run the Go backend and React frontend concurrently during development to support hot-reloading (HMR):

### 1. Start the Go Backend Daemon
Run the Go daemon from the project root:
```bash
go run ./apps/desktop/cmd/pulselinkd
```
The server will boot, initialize the SQLite database, register all Windows controller services, and start listening on port `9843` (e.g. `ws://localhost:9843/ws`).

### 2. Start the React Frontend
Navigate to the frontend folder, install dependencies, and start the Vite dev server:
```bash
cd apps/desktop/frontend
npm install
npm run dev
```
Open `http://localhost:5173` in your web browser. The Fluent controller panel will load, establish a WebSocket handshake, and allow you to control your PC.

---

## 📦 Running in Production Mode (Pre-compiled UI)

The Go backend features a static file server that automatically detects and serves the compiled React application. 

### 1. Compile the React Frontend
Run the build pipeline:
```bash
cd apps/desktop/frontend
npm run build
```
This outputs the optimized static HTML, CSS, and JS bundle into `apps/desktop/frontend/dist/`.

### 2. Run the Unified Application
Run the backend daemon from the root directory:
```bash
go run ./apps/desktop/cmd/pulselinkd
```
Because the `dist` directory is present, the Go backend will host the static assets directly. Open your web browser and navigate to:
```
http://localhost:9843
```
You can now access and run the complete companion control dashboard from a single port without running a separate Node/Vite development server!

---

## 🔌 Supported Capabilities & Actions

PulseLink's modular architecture supports the following actions over WebSockets:

| Capability | Actions | Description |
|---|---|---|
| **`media`** | `play_pause`, `next`, `previous`, `stop` | Media playback keyboard simulations. |
| **`volume`** | `up`, `down`, `mute`, `get`, `set` | Master volume increments and precise levels via CoreAudio. |
| **`brightness`** | `get`, `set` | Laptop screens (WMI WmiMonitorBrightness) and external displays (DDC/CI). |
| **`clipboard`** | `get`, `set`, `changed` (event) | High performance Win32 clipboard sync and change event broadcasts. |
| **`power`** | `lock`, `sleep`, `restart`, `shutdown` | Asynchronous PC lock, suspend state, restart, and shutdown. |
| **`sysinfo`** | `get` | Retrieves OS, CPU, RAM metrics, monitor counts, and battery status. |
| **`apps`** | `list`, `launch` | Opens predefined tools (Notepad, Calculator, CMD, Paint). |
| **`input`** | `mouse_move`, `mouse_click`, `keypress` | Virtual mouse and keyboard simulator. |
| **`notification`**| `toast` | Displays Windows bubble toast alerts. |
| **`filetransfer`**| `upload` | Pushes base64 files to `Downloads/PulseLink/` folder. |
| **`settings`** | `get`, `set` | Updates configurations directly inside `config.json`. |