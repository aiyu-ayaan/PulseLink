import {
  createContext,
  useContext,
  useCallback,
  useEffect,
  useRef,
  useState,
  type ReactNode,
} from 'react'

// Wire format spoken with the Go backend (mirrors internal/protocol).
export interface Envelope {
  id: string
  type: string
  capability: string
  action: string
  payload?: any
  error?: { code: string; message: string }
}

export type Status = 'disconnected' | 'connecting' | 'connected' | 'ready'

export interface SysInfo {
  hostname: string
  os: string
  cpuUsage: number
  ramTotal: number
  ramFree: number
  batteryPct: number
  isCharging: boolean
  monitorCount: number
}

export interface DeviceInfo {
  id: string
  name: string
  capabilities: string[]
}

export interface DeviceHistoryItem {
  id: string
  name: string
  trusted: boolean
  online: boolean
  pairedAt: number
  lastSeen: number
  capabilities: string[]
}

export interface BackendConfig {
  server: { host: string; port: number; enableTls: boolean; certFile: string; keyFile: string }
  databasePath: string
  logLevel: string
  deviceName: string
}

export interface ClipEvent {
  ts: string
  text: string
  source: 'pc' | 'you'
}

interface BackendState {
  status: Status
  error: string | null
  sysInfo: SysInfo | null
  volume: { level: number; muted: boolean }
  brightness: { internal: number; external: number }
  config: BackendConfig
  pcClipboard: string
  clipEvents: ClipEvent[]
  logs: string[]
  host: string
  port: string
  theme: 'dark' | 'light'
  devices: DeviceInfo[]
  deviceHistory: DeviceHistoryItem[]
  pairingRequests: DeviceInfo[]
  setHost: (v: string) => void
  setPort: (v: string) => void
  connect: () => void
  send: (capability: string, action: string, payload?: any) => void
  pushClipEvent: (e: ClipEvent) => void
  clearLogs: () => void
  toggleTheme: () => void
  acceptPairing: (deviceId: string) => void
  rejectPairing: (deviceId: string) => void
}

const defaultConfig: BackendConfig = {
  server: { host: '0.0.0.0', port: 9843, enableTls: true, certFile: '', keyFile: '' },
  databasePath: '',
  logLevel: 'info',
  deviceName: 'PulseLink-PC',
}

const Ctx = createContext<BackendState | null>(null)

// The desktop UI identifies itself with a fixed dev token; auth is AllowAll
// server-side today, so this is informational until pairing is enforced.
const CLIENT_CAPS = [
  'media', 'volume', 'brightness', 'clipboard', 'power', 'sysinfo',
  'apps', 'input', 'notification', 'filetransfer', 'settings',
  'devices', 'pairing',
]

export function BackendProvider({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<Status>('disconnected')
  const [error, setError] = useState<string | null>(null)
  const [sysInfo, setSysInfo] = useState<SysInfo | null>(null)
  const [volume, setVolume] = useState({ level: 50, muted: false })
  const [brightness, setBrightness] = useState({ internal: 50, external: 80 })
  const [config, setConfig] = useState<BackendConfig>(defaultConfig)
  const [pcClipboard, setPcClipboard] = useState('')
  const [clipEvents, setClipEvents] = useState<ClipEvent[]>([])
  const [logs, setLogs] = useState<string[]>([])
  const [host, setHost] = useState(window.location.hostname || 'localhost')
  const [port, setPort] = useState('9843')
  const [theme, setTheme] = useState<'dark' | 'light'>('dark')
  const [devices, setDevices] = useState<DeviceInfo[]>([])
  const [deviceHistory, setDeviceHistory] = useState<DeviceHistoryItem[]>([])
  const [pairingRequests, setPairingRequests] = useState<DeviceInfo[]>([])

  const wsRef = useRef<WebSocket | null>(null)
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const pairingPollRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  // Use refs for the message handler so WebSocket callbacks always see the
  // latest state setters without recreating the WebSocket connection.
  const handleMessageRef = useRef<(env: Envelope) => void>(() => {})
  const sendRef = useRef<(capability: string, action: string, payload?: any) => void>(() => {})

  const log = useCallback((msg: string) => {
    const ts = new Date().toLocaleTimeString()
    setLogs((prev) => [...prev.slice(-199), `[${ts}] ${msg}`])
  }, [])

  useEffect(() => {
    document.documentElement.setAttribute('data-theme', theme)
  }, [theme])

  // Stable send function that closes over the ref
  const send = useCallback(
    (capability: string, action: string, payload: any = null) => {
      const ws = wsRef.current
      if (!ws || ws.readyState !== WebSocket.OPEN) {
        log(`cannot send ${capability}.${action}: socket not open`)
        return
      }
      const env: Envelope = {
        id: `${capability}_${action}_${Math.random().toString(36).slice(2, 9)}`,
        type: 'request',
        capability,
        action,
        payload,
      }
      ws.send(JSON.stringify(env))
      log(`→ ${capability}.${action}`)
    },
    [log],
  )

  // Keep sendRef up-to-date
  useEffect(() => { sendRef.current = send }, [send])

  // Update the message handler ref whenever deps change — this avoids
  // recreating the WebSocket every time handleMessage's identity changes.
  useEffect(() => {
    handleMessageRef.current = (env: Envelope) => {
      if (env.capability === 'handshake' && env.action === 'welcome') {
        if (env.payload?.accepted) {
          setStatus('ready')
          setError(null)
          log(`handshake accepted by ${env.payload.serverName} v${env.payload.serverVersion}`)
          send('sysinfo', 'get')
          send('volume', 'get')
          send('brightness', 'get')
          send('settings', 'get')
          send('devices', 'get')
          send('devices', 'history')
          send('pairing', 'pending')
          if (pollRef.current) clearInterval(pollRef.current)
          pollRef.current = setInterval(() => {
            send('sysinfo', 'get')
            send('devices', 'history')
          }, 4000)
          if (pairingPollRef.current) clearInterval(pairingPollRef.current)
          pairingPollRef.current = setInterval(() => send('pairing', 'pending'), 5000)
        } else {
          setStatus('disconnected')
          setError(`Handshake rejected: ${env.payload?.reason || 'unauthorized'}`)
        }
        return
      }
      if (env.error) {
        log(`✕ ${env.capability}.${env.action}: ${env.error.message}`)
        return
      }

      switch (env.capability) {
        case 'sysinfo':
          if (env.payload) setSysInfo(env.payload)
          break
        case 'volume':
          if (env.payload) setVolume({ level: env.payload.level, muted: env.payload.muted })
          break
        case 'brightness':
          if (env.payload)
            setBrightness({ internal: env.payload.internal, external: env.payload.external })
          break
        case 'settings':
          if (env.payload) setConfig(env.payload)
          break
        case 'devices':
          if (env.action === 'history') {
            if (env.payload) setDeviceHistory(env.payload)
          } else {
            if (env.payload) setDevices(env.payload)
          }
          break
        case 'pairing':
          if (env.action === 'request') {
            if (env.payload) {
              setPairingRequests((prev) => {
                if (prev.some((r) => r.id === env.payload.id)) return prev
                return [...prev, env.payload]
              })
              send('devices', 'history')
            }
          } else if (env.action === 'pending') {
            if (env.payload?.devices) {
              setPairingRequests(env.payload.devices)
            }
          } else if (env.action === 'list') {
            if (env.payload) setPairingRequests(env.payload)
          } else if (env.action === 'approved' || env.action === 'rejected') {
            const devId = env.payload?.deviceId || env.payload
            if (devId) {
              setPairingRequests((prev) => prev.filter((r) => r.id !== devId))
            }
            send('devices', 'history')
          }
          break
        case 'clipboard':
          if (env.payload?.text != null) {
            setPcClipboard(env.payload.text)
            if (env.action === 'changed') {
              setClipEvents((prev) => [
                { ts: new Date().toLocaleTimeString(), text: env.payload.text, source: 'pc' },
                ...prev.slice(0, 24),
              ])
            }
          }
          break
      }
    }
  }, [log, send])

  // Stable connect that never changes identity — avoids re-creating the
  // WebSocket on every render. Reads host/port from refs.
  const hostRef = useRef(host)
  const portRef = useRef(port)
  useEffect(() => { hostRef.current = host }, [host])
  useEffect(() => { portRef.current = port }, [port])

  const connectRef = useRef<(() => void) | null>(null)

  const scheduleReconnect = useCallback(() => {
    if (reconnectTimerRef.current) clearTimeout(reconnectTimerRef.current)
    reconnectTimerRef.current = setTimeout(() => {
      log('attempting automatic reconnection...')
      connectRef.current?.()
    }, 3000)
  }, [log])

  const connect = useCallback(() => {
    if (reconnectTimerRef.current) clearTimeout(reconnectTimerRef.current)
    if (wsRef.current) {
      wsRef.current.onclose = null
      wsRef.current.onerror = null
      wsRef.current.close()
    }
    setStatus('connecting')
    setError(null)
    const h = hostRef.current
    const p = portRef.current
    const scheme = window.location.protocol === 'https:' ? 'wss' : 'ws'
    const url = `${scheme}://${h}:${p}/ws`
    log(`connecting ${url}`)
    try {
      const ws = new WebSocket(url)
      wsRef.current = ws
      ws.onopen = () => {
        setStatus('connected')
        if (reconnectTimerRef.current) {
          clearTimeout(reconnectTimerRef.current)
          reconnectTimerRef.current = null
        }
        ws.send(
          JSON.stringify({
            id: 'handshake_init',
            type: 'request',
            capability: 'handshake',
            action: 'hello',
            payload: {
              protocolVersion: '1.0',
              deviceId: 'desktop-ui',
              deviceName: 'PulseLink Desktop',
              appVersion: '1.0.0',
              token: 'desktop-local',
              capabilities: CLIENT_CAPS,
            },
          }),
        )
      }
      ws.onmessage = (e) => {
        try {
          handleMessageRef.current(JSON.parse(e.data))
        } catch (err: any) {
          log(`parse error: ${err.message}`)
        }
      }
      ws.onerror = () => {
        setError(`Connection error. Is the backend running on ${h}:${p}?`)
        if (h === 'localhost') {
          log(`localhost connection failed, trying fallback 127.0.0.1`)
          hostRef.current = '127.0.0.1'
          setHost('127.0.0.1')
          setTimeout(connect, 200)
        } else {
          scheduleReconnect()
        }
      }
      ws.onclose = () => {
        setStatus('disconnected')
        if (pollRef.current) clearInterval(pollRef.current)
        if (pairingPollRef.current) clearInterval(pairingPollRef.current)
        scheduleReconnect()
      }
    } catch (err: any) {
      setStatus('disconnected')
      setError(err.message)
      scheduleReconnect()
    }
  }, [log, scheduleReconnect])

  useEffect(() => {
    connectRef.current = connect
  }, [connect])

  useEffect(() => {
    connect()
    return () => {
      wsRef.current?.close()
      if (pollRef.current) clearInterval(pollRef.current)
      if (pairingPollRef.current) clearInterval(pairingPollRef.current)
      if (reconnectTimerRef.current) clearTimeout(reconnectTimerRef.current)
    }
    // connect() is now stable; auto-connect once on mount.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const acceptPairing = useCallback((deviceId: string) => {
    send('pairing', 'accept', { deviceId })
    setPairingRequests((prev) => prev.filter((r) => r.id !== deviceId))
  }, [send])

  const rejectPairing = useCallback((deviceId: string) => {
    send('pairing', 'reject', { deviceId })
    setPairingRequests((prev) => prev.filter((r) => r.id !== deviceId))
  }, [send])

  const value: BackendState = {
    status,
    error,
    sysInfo,
    volume,
    brightness,
    config,
    pcClipboard,
    clipEvents,
    logs,
    host,
    port,
    theme,
    devices,
    deviceHistory,
    pairingRequests,
    setHost,
    setPort,
    connect,
    send,
    pushClipEvent: (e) => setClipEvents((prev) => [e, ...prev.slice(0, 24)]),
    clearLogs: () => setLogs([]),
    toggleTheme: () => setTheme((t) => (t === 'dark' ? 'light' : 'dark')),
    acceptPairing,
    rejectPairing,
  }

  return <Ctx.Provider value={value}>{children}</Ctx.Provider>
}

export function useBackend() {
  const ctx = useContext(Ctx)
  if (!ctx) throw new Error('useBackend must be used within BackendProvider')
  return ctx
}
