# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**PulseLink** is a modular desktop companion platform that allows Android devices to securely control a Windows PC over the local network. The project prioritizes simplicity, security, and extensibility with a modular architecture where features are isolated modules communicating through interfaces and events.

- **Desktop**: Go + Wails v3 + React + TypeScript + TailwindCSS + shadcn/ui
- **Android**: Kotlin + Jetpack Compose + Material 3 + Room + Ktor Client
- **Communication**: WebSocket + JSON (initially) → Protocol Buffers (future), with TLS encryption

**Target Platforms**: Windows 11/10 (desktop), Android 10+ (mobile)

---

## High-Level Architecture

### Overall Flow

```
Android Client
    ↓
[Pairing & Authentication]
    ↓
Encrypted WebSocket Connection
    ↓
[Capability Exchange]
    ↓
Module Request → Desktop Handler → Module Response
```

### Desktop Architecture (`apps/desktop/`)

```
apps/desktop/
├── cmd/               # Application entry points
├── frontend/          # React + TypeScript UI (compiled by Wails)
├── internal/
│   ├── api/           # HTTP/WebSocket endpoints
│   ├── websocket/     # Connection management
│   ├── auth/          # Pairing & authentication
│   ├── pairing/       # Device pairing logic
│   ├── storage/       # SQLite data layer
│   ├── plugins/       # Module/plugin system
│   └── services/      # Individual feature modules
│       ├── media/
│       ├── brightness/
│       ├── clipboard/
│       ├── notifications/
│       └── [more modules]
└── assets/            # Static resources
```

**Key Principle**: Each feature exists as an isolated service module under `internal/services/`. Modules communicate through interfaces and events, not direct dependencies.

### Android Architecture (`apps/android/`)

```
apps/android/
├── app/               # Application module
├── core/              # Shared utilities, network, encryption
├── feature/           # Feature-specific modules (media, brightness, etc.)
├── service/           # Background services
├── widgets/           # Reusable Compose components
└── buildSrc/          # Build configuration & plugins
```

**Pattern**: MVVM with Jetpack Compose. Each feature module contains its own ViewModels and UI. Shared logic lives in `core`.

### Shared/Protocol (`packages/`)

```
packages/
├── protocol/          # Protocol specifications & schemas
│   ├── protobuf/      # Protocol Buffer definitions (future)
│   ├── schema/        # JSON schema (current)
│   └── docs/          # Protocol documentation
├── api/               # Shared API models & interfaces
├── icons/             # Icon assets
└── branding/          # Brand guidelines & colors
```

---

## Build Commands

### Desktop (Go + Wails)

**Development**:
```bash
# Install Wails (one-time)
go install github.com/wailsapp/wails/v3/cmd/wails@latest

# Run in dev mode (hot reload)
wails dev

# Run specific dev configuration
wails dev -debug
```

**Build**:
```bash
# Build production binary
wails build

# Build for specific platform
wails build -platform windows/amd64
wails build -platform windows/arm64

# Build with custom output
wails build -o pulselink.exe
```

**Frontend** (React + TypeScript in `apps/desktop/frontend/`):
```bash
cd apps/desktop/frontend

# Install dependencies
npm install

# Development server (usually run via `wails dev`)
npm run dev

# Build for production
npm run build

# Type checking
npm run type-check

# Linting
npm run lint
```

### Android (Gradle + Kotlin)

**Development**:
```bash
cd apps/android

# Install dependencies & sync
./gradlew build

# Build debug APK
./gradlew assembleDebug

# Build release APK (requires signing config)
./gradlew assembleRelease

# Run on connected device/emulator
./gradlew installDebug

# Run tests
./gradlew test
./gradlew connectedAndroidTest  # Instrumented tests
```

**Linting**:
```bash
# ktlint (Kotlin style)
./gradlew ktlintCheck

# Fix formatting
./gradlew ktlintFormat
```

### Protocol & Shared

```bash
# Protocol validation (when protobuf is introduced)
# See packages/protocol/docs/ for schema validation

# Build shared packages
cd packages/api
npm install
npm run build
```

---

## Testing

### Desktop (Go)

```bash
# All tests
go test ./...

# Single package
go test ./internal/auth

# Single test
go test ./internal/auth -run TestSpecificName

# With coverage
go test -cover ./...

# Verbose
go test -v ./...
```

### Frontend (React)

```bash
cd apps/desktop/frontend

# Run test suite
npm test

# Watch mode
npm test -- --watch

# Coverage
npm test -- --coverage
```

### Android

```bash
cd apps/android

# Unit tests
./gradlew test

# Instrumented tests (requires device/emulator)
./gradlew connectedAndroidTest
```

---

## Development Workflow

### Adding a New Feature Module

1. **Desktop**: Create `apps/desktop/internal/services/[feature]/`
   - Define interfaces for the module contract
   - Implement handlers for WebSocket messages
   - No direct imports from other service modules
   - Expose functionality through event-based communication

2. **Android**: Create `apps/android/feature/[feature]/`
   - Implement ViewModel (StateFlow-based)
   - Create Compose UI components
   - Import shared logic from `core`

3. **Protocol**: Define API contract in `packages/protocol/schema/[feature].json`
   - Document request/response structure
   - Include examples

4. **Update**: Both platforms must negotiate capabilities during connection handshake

### Modular Design Rules

- **No circular dependencies**: Modules should not depend on sibling modules
- **Communication through events/interfaces**: Use message passing, not direct calls
- **Platform-specific code isolated**: Don't share platform code (e.g., Go-specific utils with Android)
- **Testable in isolation**: Each module should be testable without others

---

## Code Standards

### Go

- Run `gofmt` before committing
- Keep packages focused and small
- Use interfaces only where meaningful (composition)
- Error handling: return errors, don't panic
- No global state (use dependency injection)

### Kotlin

- Follow [official Kotlin style guide](https://kotlinlang.org/docs/coding-conventions.html)
- Immutable state (prefer `val` over `var`)
- Use `StateFlow` for reactive state
- Compose components should be composable and predictable
- No mutable global state

### React/TypeScript

- Functional components + Hooks
- TypeScript strict mode enabled
- Feature-based folder structure
- Prefer composition over inheritance

---

## Repository Layout Reference

- `apps/desktop/` — Go backend + React frontend
- `apps/android/` — Kotlin Android app
- `packages/` — Shared protocol, API models, branding
- `docs/` — Architecture docs, API specs, roadmap
- `tools/` — Build scripts, code generators, release tools
- `.github/workflows/` — CI/CD pipelines (when added)

---

## Key Files to Understand Architecture

- `main.go` — Desktop app entry point (Wails setup)
- `apps/desktop/frontend/App.tsx` — React root component
- `apps/android/app/build.gradle.kts` — Android build config
- `wails.json` — Wails project configuration (frontend/backend integration)
- `packages/protocol/schema/` — API contracts between platforms

---

## Security Considerations

- Every device must be **paired** before communication (stored in SQLite)
- Connections are **encrypted with TLS** (Wails handles this)
- No anonymous WebSocket connections
- Pairing uses secure token exchange
- Future: QR code pairing, public/private key authentication

---

## Design Principles

1. **Simplicity first** — Choose obvious solutions over clever ones
2. **Small packages** — Each module should have a single responsibility
3. **Testable code** — Write code that can be tested in isolation
4. **Composition over inheritance** — Favor interfaces and composition
5. **Explicit dependencies** — No hidden global state
6. **No premature optimization** — Write readable code first

---

## Common Tasks

### Running the App Locally

```bash
# Desktop
wails dev          # Runs Go backend + React frontend with hot reload

# Android
# Use Android Studio or: ./gradlew installDebug && adb shell am start ...
```

### Making a Protocol Change

1. Update schema in `packages/protocol/schema/`
2. Implement handler on desktop (`internal/services/`)
3. Implement UI/ViewModel on Android (`feature/`)
4. Test locally before committing

### Debugging

**Desktop**:
- Wails dev server shows frontend console in browser
- Use `log.Printf()` for backend logging
- Check `wails dev -debug` for verbose output

**Android**:
- Use Android Studio Logcat
- Add `Log.d("TAG", "message")` for debugging

### Commits

Keep commits small and meaningful. Example format:
```
[desktop] Add brightness control API handler
[android] Implement brightness UI with StateFlow

- Add HTTP POST endpoint for brightness change
- Validate range 0-100
- Update brightness module tests
```

---

## Future Considerations

- **Protocol Buffers**: Replace JSON with protobuf for performance
- **Plugin System**: Allow community plugins for new features
- **Screen Mirroring**: Planned for v2
- **Remote Input**: Keyboard/mouse control
- **OBS Integration**: Streaming control
- **AI Voice Commands**: Natural language support

When implementing these, maintain the modular architecture — each feature should be self-contained.
