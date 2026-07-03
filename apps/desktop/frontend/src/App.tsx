import React, { useState, useEffect, useRef } from 'react'
import {
  Activity,
  Volume2,
  VolumeX,
  Sun,
  Clipboard,
  Smartphone,
  Settings,
  Tv,
  Send,
  Terminal,
  Info,
  RefreshCw,
  Play,
  SkipForward,
  SkipBack,
  Square,
  Lock,
  Moon,
  Laptop,
  Trash2,
  Key,
  Database,
  Grid
} from 'lucide-react'

// Protocol envelope structure
interface Envelope {
  id: string
  type: string
  capability: string
  action: string
  payload?: any
  error?: {
    code: string
    message: string
  }
}

interface SysInfoState {
  hostname: string
  os: string
  cpuUsage: number
  ramTotal: number
  ramFree: number
  batteryPct: number
  isCharging: boolean
  monitorCount: number
}

interface AppItem {
  name: string
  path: string
}

interface ClipboardLog {
  timestamp: string
  text: string
  source: 'pc' | 'android'
}

interface ToastLog {
  timestamp: string
  title: string
  message: string
}

interface BackendConfig {
  server: {
    host: string
    port: number
    enableTls: boolean
    certFile: string
    keyFile: string
  }
  databasePath: string
  logLevel: string
  deviceName: string
}

function App() {
  const [activeTab, setActiveTab] = useState<string>('dashboard')
  const [wsStatus, setWsStatus] = useState<'disconnected' | 'connecting' | 'connected' | 'ready'>('disconnected')
  const [wsError, setWsError] = useState<string | null>(null)
  
  // States fetched from backend
  const [sysInfo, setSysInfo] = useState<SysInfoState | null>(null)
  const [volumeLevel, setVolumeLevel] = useState<number>(50)
  const [isMuted, setIsMuted] = useState<boolean>(false)
  const [internalBrightness, setInternalBrightness] = useState<number>(50)
  const [externalBrightness, setExternalBrightness] = useState<number>(80)
  const [clipboardText, setClipboardText] = useState<string>('')
  const [pcClipboardText, setPcClipboardText] = useState<string>('')
  const [predefinedAppsList, setPredefinedAppsList] = useState<AppItem[]>([
    { name: "Notepad", path: "notepad.exe" },
    { name: "Calculator", path: "calc.exe" },
    { name: "Task Manager", path: "taskmgr.exe" },
    { name: "Command Prompt", path: "cmd.exe" },
    { name: "Paint", path: "mspaint.exe" }
  ])
  
  // UI inputs
  const [toastTitle, setToastTitle] = useState<string>('PulseLink Message')
  const [toastMessage, setToastMessage] = useState<string>('Hello from the desktop panel!')
  const [customWsHost, setCustomWsHost] = useState<string>('localhost')
  const [customWsPort, setCustomWsPort] = useState<string>('9843')
  
  // Settings config state
  const [backendConfig, setBackendConfig] = useState<BackendConfig>({
    server: { host: '0.0.0.0', port: 9843, enableTls: true, certFile: '', keyFile: '' },
    databasePath: '',
    logLevel: 'info',
    deviceName: 'PulseLink-PC'
  })

  // Logs state
  const [clipboardLogs, setClipboardLogs] = useState<ClipboardLog[]>([])
  const [toastLogs, setToastLogs] = useState<ToastLog[]>([])
  const [consoleLogs, setConsoleLogs] = useState<string[]>([])
  
  // Refs
  const wsRef = useRef<WebSocket | null>(null)
  const pollIntervalRef = useRef<any>(null)
  const logTerminalRef = useRef<HTMLDivElement>(null)

  const logToConsole = (msg: string) => {
    const timestamp = new Date().toLocaleTimeString()
    setConsoleLogs(prev => [...prev.slice(-99), `[${timestamp}] ${msg}`])
  }

  // Auto-scroll logs
  useEffect(() => {
    if (logTerminalRef.current) {
      logTerminalRef.current.scrollTop = logTerminalRef.current.scrollHeight
    }
  }, [consoleLogs])

  // WebSocket connection handler
  const connectWebSocket = () => {
    if (wsRef.current) {
      wsRef.current.close()
    }
    
    setWsStatus('connecting')
    setWsError(null)
    const secure = window.location.protocol === 'https:'
    const protocolStr = secure ? 'wss' : 'ws'
    const wsUrl = `${protocolStr}://${customWsHost}:${customWsPort}/ws`
    
    logToConsole(`Connecting to WebSocket: ${wsUrl}`)
    
    try {
      const ws = new WebSocket(wsUrl)
      wsRef.current = ws

      ws.onopen = () => {
        setWsStatus('connected')
        logToConsole("WebSocket socket opened. Performing hello handshake...")
        
        // Send ClientHello immediately
        const helloEnvelope: Envelope = {
          id: "handshake_init",
          type: "request",
          capability: "handshake",
          action: "hello",
          payload: {
            protocolVersion: "1.0",
            deviceId: "desktop-web-ui",
            deviceName: "Web Controller Dashboard",
            appVersion: "1.0.0",
            token: "web-dev-session",
            capabilities: ["media", "volume", "brightness", "clipboard", "power", "sysinfo", "apps", "input", "notification", "filetransfer", "settings"]
          }
        }
        ws.send(JSON.stringify(helloEnvelope))
      }

      ws.onmessage = (e) => {
        try {
          const envelope: Envelope = JSON.parse(e.data)
          handleServerMessage(envelope)
        } catch (err: any) {
          logToConsole(`Error parsing server message: ${err.message}`)
        }
      }

      ws.onerror = () => {
        logToConsole("WebSocket encountered an error.")
        setWsError("Connection error occurred. Ensure the Go daemon is running on port 9843.")
      }

      ws.onclose = () => {
        setWsStatus('disconnected')
        logToConsole("WebSocket connection closed.")
        clearInterval(pollIntervalRef.current)
      }
    } catch (err: any) {
      setWsStatus('disconnected')
      setWsError(err.message)
      logToConsole(`Connection exception: ${err.message}`)
    }
  }

  // Handle incoming envelopes
  const handleServerMessage = (envelope: Envelope) => {
    // 1. Handshake response
    if (envelope.capability === "handshake" && envelope.action === "welcome") {
      if (envelope.payload?.accepted) {
        setWsStatus('ready')
        logToConsole(`Handshake accepted by ${envelope.payload.serverName} (v${envelope.payload.serverVersion})`)
        
        // Fetch initial details
        sendRequest("sysinfo", "get")
        sendRequest("volume", "get")
        sendRequest("brightness", "get")
        sendRequest("settings", "get")
        
        // Start polling sysinfo, volume, brightness
        clearInterval(pollIntervalRef.current)
        pollIntervalRef.current = setInterval(() => {
          sendRequest("sysinfo", "get")
        }, 4000)
      } else {
        setWsStatus('disconnected')
        setWsError(`Handshake rejected: ${envelope.payload?.reason || 'unauthorized'}`)
        logToConsole(`Handshake rejected: ${envelope.payload?.reason}`)
      }
      return
    }

    // Handle responses or events
    if (envelope.error) {
      logToConsole(`Error response from [${envelope.capability}.${envelope.action}]: ${envelope.error.message}`)
      return
    }

    switch (envelope.capability) {
      case "sysinfo":
        if (envelope.action === "get" && envelope.payload) {
          setSysInfo(envelope.payload)
        }
        break
      case "volume":
        if (envelope.action === "get" && envelope.payload) {
          setVolumeLevel(envelope.payload.level)
          setIsMuted(envelope.payload.muted)
        }
        break
      case "brightness":
        if (envelope.action === "get" && envelope.payload) {
          setInternalBrightness(envelope.payload.internal)
          setExternalBrightness(envelope.payload.external)
        }
        break
      case "settings":
        if (envelope.action === "get" && envelope.payload) {
          setBackendConfig(envelope.payload)
        }
        break
      case "apps":
        if (envelope.action === "list" && Array.isArray(envelope.payload)) {
          setPredefinedAppsList(envelope.payload)
        }
        break
      case "clipboard":
        if (envelope.action === "get" && envelope.payload) {
          setPcClipboardText(envelope.payload.text)
          logToConsole(`Retrieved PC clipboard: "${envelope.payload.text}"`)
        } else if (envelope.action === "changed" && envelope.payload) {
          // Event broadcasted from server
          const newText = envelope.payload.text
          setClipboardLogs(prev => [
            {
              timestamp: new Date().toLocaleTimeString(),
              text: newText,
              source: 'pc'
            },
            ...prev.slice(0, 19)
          ])
          logToConsole(`Event: PC clipboard changed: "${newText}"`)
        }
        break
      default:
        logToConsole(`Received payload for capability [${envelope.capability}.${envelope.action}]: ${JSON.stringify(envelope.payload)}`)
    }
  }

  // Send request helper
  const sendRequest = (capability: string, action: string, payload: any = null) => {
    if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) {
      logToConsole(`Cannot send request: WebSocket not open (state: ${wsStatus})`)
      return
    }
    const id = `${capability}_${action}_${Math.random().toString(36).substr(2, 9)}`
    const envelope: Envelope = {
      id,
      type: "request",
      capability,
      action,
      payload
    }
    wsRef.current.send(JSON.stringify(envelope))
    logToConsole(`Sent request [${capability}.${action}]`)
  }

  // Auto connect on mount
  useEffect(() => {
    connectWebSocket()
    return () => {
      if (wsRef.current) {
        wsRef.current.close()
      }
      clearInterval(pollIntervalRef.current)
    }
  }, [])

  // Clipboard sync triggers
  const handleSetClipboard = (e: React.FormEvent) => {
    e.preventDefault()
    if (!clipboardText) return
    sendRequest("clipboard", "set", { text: clipboardText })
    setClipboardLogs(prev => [
      {
        timestamp: new Date().toLocaleTimeString(),
        text: clipboardText,
        source: 'android' // Simulates external device push
      },
      ...prev.slice(0, 19)
    ])
    setClipboardText('')
  }

  const handleSendToast = (e: React.FormEvent) => {
    e.preventDefault()
    sendRequest("notification", "toast", { title: toastTitle, message: toastMessage })
    setToastLogs(prev => [
      {
        timestamp: new Date().toLocaleTimeString(),
        title: toastTitle,
        message: toastMessage
      },
      ...prev.slice(0, 19)
    ])
    logToConsole(`Sent toast trigger: "${toastTitle}" - "${toastMessage}"`)
  }

  // Volume operations
  const setVolumePct = (val: number) => {
    setVolumeLevel(val)
    sendRequest("volume", "set", { level: val })
  }

  const setBrightnessPct = (type: string, val: number) => {
    if (type === 'internal') {
      setInternalBrightness(val)
    } else {
      setExternalBrightness(val)
    }
    sendRequest("brightness", "set", { type, level: val })
  }

  const toggleMute = () => {
    setIsMuted(!isMuted)
    sendRequest("volume", "mute")
  }

  // Predefined settings updates
  const saveSettings = (e: React.FormEvent) => {
    e.preventDefault()
    sendRequest("settings", "set", backendConfig)
  }

  return (
    <div className="flex h-screen bg-[#020617] text-[#f8fafc] font-sans selection:bg-green-500/30 selection:text-green-300">
      
      {/* Side Navigation panel (Fluent Acrylic sidebar) */}
      <aside className="w-64 border-r border-slate-800/80 bg-slate-950/40 backdrop-blur-md flex flex-col">
        <div className="p-6 border-b border-slate-800/50 flex items-center gap-3">
          <div className="w-8 h-8 rounded-lg bg-green-500/20 flex items-center justify-center border border-green-500/30 text-green-400">
            <Activity size={18} className="animate-pulse" />
          </div>
          <div>
            <h1 className="font-bold text-lg leading-tight tracking-tight text-white">PulseLink</h1>
            <span className="text-[10px] text-slate-500 font-mono">v0.1.0 (Stage 2)</span>
          </div>
        </div>

        {/* Navigation list */}
        <nav className="flex-1 px-3 py-4 space-y-1 overflow-y-auto">
          {[
            { id: 'dashboard', label: 'Dashboard', icon: Grid },
            { id: 'devices', label: 'Devices Manager', icon: Smartphone },
            { id: 'media', label: 'Media & Volume', icon: Volume2 },
            { id: 'brightness', label: 'Display Brightness', icon: Sun },
            { id: 'clipboard', label: 'Clipboard Sync', icon: Clipboard },
            { id: 'notifications', label: 'Notification Bridge', icon: Send },
            { id: 'apps', label: 'Predefined Apps', icon: Laptop },
            { id: 'logs', label: 'Terminal Logs', icon: Terminal },
            { id: 'settings', label: 'Settings', icon: Settings },
            { id: 'about', label: 'About App', icon: Info },
          ].map((tab) => {
            const Icon = tab.icon
            const isActive = activeTab === tab.id
            return (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                className={`w-full flex items-center gap-3 px-3 py-2 text-sm font-medium rounded-lg transition-colors cursor-pointer ${
                  isActive
                    ? 'bg-slate-800 text-green-400 border-l-2 border-green-500'
                    : 'text-slate-400 hover:text-white hover:bg-slate-900/60'
                }`}
              >
                <Icon size={16} />
                <span>{tab.label}</span>
              </button>
            )
          })}
        </nav>

        {/* Footer connections status widget */}
        <div className="p-4 border-t border-slate-800 bg-slate-950/60">
          <div className="flex items-center justify-between mb-2">
            <span className="text-xs text-slate-500">Service Status</span>
            <div className="flex items-center gap-1.5">
              <span className={`w-2 h-2 rounded-full ${
                wsStatus === 'ready' ? 'bg-green-500 pulse-green' :
                wsStatus === 'connected' ? 'bg-blue-400' :
                wsStatus === 'connecting' ? 'bg-amber-400 animate-pulse' :
                'bg-red-500'
              }`} />
              <span className="text-xs font-semibold uppercase tracking-wider text-slate-300">
                {wsStatus}
              </span>
            </div>
          </div>
          <button 
            onClick={connectWebSocket}
            className="w-full py-1.5 px-3 rounded bg-slate-800 hover:bg-slate-700 active:bg-slate-900 text-xs font-semibold transition-colors flex items-center justify-center gap-2 cursor-pointer border border-slate-700/50"
          >
            <RefreshCw size={12} className={wsStatus === 'connecting' ? 'animate-spin' : ''} />
            <span>Reconnect</span>
          </button>
        </div>
      </aside>

      {/* Main Panel */}
      <main className="flex-1 flex flex-col bg-[#020617] overflow-y-auto">
        {/* Header */}
        <header className="px-8 py-4 border-b border-slate-800/80 bg-slate-950/20 backdrop-blur flex justify-between items-center shrink-0">
          <div>
            <h2 className="text-xl font-bold tracking-tight text-white capitalize">{activeTab.replace('-', ' ')}</h2>
            <p className="text-xs text-slate-400">Windows Companion Backend Control Board</p>
          </div>

          {/* Quick host connection configure */}
          <div className="flex items-center gap-2">
            <input 
              type="text" 
              placeholder="Host"
              value={customWsHost}
              onChange={(e) => setCustomWsHost(e.target.value)}
              className="bg-slate-900/60 border border-slate-800 rounded px-2.5 py-1 text-xs text-slate-300 focus:outline-none focus:border-green-500/50 w-28"
            />
            <input 
              type="text" 
              placeholder="Port"
              value={customWsPort}
              onChange={(e) => setCustomWsPort(e.target.value)}
              className="bg-slate-900/60 border border-slate-800 rounded px-2.5 py-1 text-xs text-slate-300 focus:outline-none focus:border-green-500/50 w-16"
            />
            {wsStatus !== 'ready' && (
              <button 
                onClick={connectWebSocket}
                className="bg-green-600 hover:bg-green-500 active:bg-green-700 text-white rounded px-3 py-1 text-xs font-semibold transition-colors cursor-pointer"
              >
                Connect
              </button>
            )}
          </div>
        </header>

        {/* Content Container */}
        <div className="p-8 flex-1">
          {wsError && (
            <div className="mb-6 p-4 rounded-lg bg-red-500/10 border border-red-500/20 text-red-300 text-sm">
              <p className="font-semibold">Connection Alert</p>
              <p className="text-xs mt-0.5">{wsError}</p>
            </div>
          )}

          {/* TAB 1: DASHBOARD */}
          {activeTab === 'dashboard' && (
            <div className="space-y-6">
              
              {/* Top Summary Cards */}
              <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                
                {/* Host System Info Card */}
                <div className="glass-panel p-6 rounded-2xl flex flex-col justify-between">
                  <div className="flex justify-between items-start">
                    <div>
                      <span className="text-xs font-medium text-slate-400 uppercase tracking-wider">Host Machine</span>
                      <h3 className="text-2xl font-bold text-white mt-1">{sysInfo?.hostname || 'Unknown Host'}</h3>
                    </div>
                    <div className="w-10 h-10 rounded-xl bg-slate-800 flex items-center justify-center border border-slate-700/50 text-green-400">
                      <Tv size={20} />
                    </div>
                  </div>
                  <div className="mt-6 flex flex-col gap-2 border-t border-slate-800/40 pt-4 text-xs font-mono text-slate-400">
                    <div className="flex justify-between">
                      <span>OS:</span>
                      <span className="text-slate-200">{sysInfo?.os || 'Windows'}</span>
                    </div>
                    <div className="flex justify-between">
                      <span>Monitors:</span>
                      <span className="text-slate-200">{sysInfo?.monitorCount || 1} display(s)</span>
                    </div>
                  </div>
                </div>

                {/* Battery Status Card */}
                <div className="glass-panel p-6 rounded-2xl flex flex-col justify-between">
                  <div className="flex justify-between items-start">
                    <div>
                      <span className="text-xs font-medium text-slate-400 uppercase tracking-wider">Device Battery</span>
                      <h3 className="text-3xl font-bold text-white mt-1">
                        {sysInfo ? `${sysInfo.batteryPct}%` : '100%'}
                      </h3>
                    </div>
                    <div className={`w-10 h-10 rounded-xl flex items-center justify-center border ${
                      sysInfo?.isCharging 
                        ? 'bg-green-500/10 border-green-500/20 text-green-400' 
                        : 'bg-slate-800 border-slate-700/50 text-slate-300'
                    }`}>
                      <Activity size={20} className={sysInfo?.isCharging ? 'animate-bounce' : ''} />
                    </div>
                  </div>
                  <div className="mt-6 flex flex-col gap-2 border-t border-slate-800/40 pt-4 text-xs font-mono text-slate-400">
                    <div className="flex justify-between">
                      <span>Charging Status:</span>
                      <span className={sysInfo?.isCharging ? 'text-green-400' : 'text-slate-300'}>
                        {sysInfo?.isCharging ? 'AC Connected' : 'On Battery'}
                      </span>
                    </div>
                    <div className="flex justify-between">
                      <span>Battery Health:</span>
                      <span className="text-slate-200">Good</span>
                    </div>
                  </div>
                </div>

                {/* Resource Stats Card */}
                <div className="glass-panel p-6 rounded-2xl flex flex-col justify-between">
                  <div className="flex justify-between items-start">
                    <div>
                      <span className="text-xs font-medium text-slate-400 uppercase tracking-wider">PC Resources</span>
                      <div className="flex items-baseline gap-2 mt-1">
                        <span className="text-2xl font-bold text-white">CPU: {sysInfo?.cpuUsage ? `${sysInfo.cpuUsage}%` : '12%'}</span>
                      </div>
                    </div>
                    <div className="w-10 h-10 rounded-xl bg-slate-800 flex items-center justify-center border border-slate-700/50 text-blue-400">
                      <Activity size={20} />
                    </div>
                  </div>
                  <div className="mt-6 flex flex-col gap-2 border-t border-slate-800/40 pt-4 text-xs font-mono text-slate-400">
                    <div className="flex justify-between">
                      <span>RAM Utilization:</span>
                      <span className="text-slate-200">
                        {sysInfo ? `${Math.round(((sysInfo.ramTotal - sysInfo.ramFree) / sysInfo.ramTotal) * 100)}%` : '50%'}
                      </span>
                    </div>
                    <div className="flex justify-between">
                      <span>RAM Free / Total:</span>
                      <span className="text-slate-200">
                        {sysInfo ? `${sysInfo.ramFree}MB / ${sysInfo.ramTotal}MB` : '8192MB / 16384MB'}
                      </span>
                    </div>
                  </div>
                </div>

              </div>

              {/* Quick Actions Panel */}
              <div className="glass-panel p-6 rounded-2xl">
                <h4 className="text-sm font-semibold text-white mb-4">Quick Desktop Actions</h4>
                <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
                  <button 
                    onClick={() => sendRequest("power", "lock")}
                    className="flex flex-col items-center gap-3 p-4 rounded-xl border border-slate-800 bg-slate-900/40 hover:bg-slate-850 hover:border-slate-700 transition-all cursor-pointer group"
                  >
                    <div className="w-10 h-10 rounded-lg bg-red-500/10 text-red-400 flex items-center justify-center group-hover:scale-105 transition-transform">
                      <Lock size={18} />
                    </div>
                    <span className="text-xs font-medium text-slate-300">Lock PC</span>
                  </button>
                  <button 
                    onClick={() => sendRequest("power", "sleep")}
                    className="flex flex-col items-center gap-3 p-4 rounded-xl border border-slate-800 bg-slate-900/40 hover:bg-slate-850 hover:border-slate-700 transition-all cursor-pointer group"
                  >
                    <div className="w-10 h-10 rounded-lg bg-amber-500/10 text-amber-400 flex items-center justify-center group-hover:scale-105 transition-transform">
                      <Moon size={18} />
                    </div>
                    <span className="text-xs font-medium text-slate-300">Sleep</span>
                  </button>
                  <button 
                    onClick={toggleMute}
                    className="flex flex-col items-center gap-3 p-4 rounded-xl border border-slate-800 bg-slate-900/40 hover:bg-slate-850 hover:border-slate-700 transition-all cursor-pointer group"
                  >
                    <div className={`w-10 h-10 rounded-lg flex items-center justify-center group-hover:scale-105 transition-transform ${
                      isMuted ? 'bg-red-500/10 text-red-400' : 'bg-green-500/10 text-green-400'
                    }`}>
                      {isMuted ? <VolumeX size={18} /> : <Volume2 size={18} />}
                    </div>
                    <span className="text-xs font-medium text-slate-300">{isMuted ? 'Unmute' : 'Mute Master'}</span>
                  </button>
                  <button 
                    onClick={() => sendRequest("clipboard", "get")}
                    className="flex flex-col items-center gap-3 p-4 rounded-xl border border-slate-800 bg-slate-900/40 hover:bg-slate-850 hover:border-slate-700 transition-all cursor-pointer group"
                  >
                    <div className="w-10 h-10 rounded-lg bg-blue-500/10 text-blue-400 flex items-center justify-center group-hover:scale-105 transition-transform">
                      <Clipboard size={18} />
                    </div>
                    <span className="text-xs font-medium text-slate-300">Sync Clip</span>
                  </button>
                </div>
              </div>

              {/* Split Volume and Brightness quick control */}
              <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                <div className="glass-panel p-6 rounded-2xl">
                  <div className="flex justify-between items-center mb-4">
                    <h4 className="text-sm font-semibold text-white">System Volume</h4>
                    <span className="text-xs font-mono text-slate-400">{volumeLevel}%</span>
                  </div>
                  <div className="flex items-center gap-4">
                    <button onClick={toggleMute} className="text-slate-400 hover:text-white transition-colors cursor-pointer">
                      {isMuted ? <VolumeX size={20} className="text-red-400" /> : <Volume2 size={20} />}
                    </button>
                    <input 
                      type="range" 
                      min="0" 
                      max="100" 
                      value={volumeLevel}
                      onChange={(e) => setVolumePct(Number(e.target.value))}
                      className="w-full accent-green-500 cursor-pointer h-1.5 bg-slate-800 rounded-lg appearance-none"
                    />
                  </div>
                </div>

                <div className="glass-panel p-6 rounded-2xl">
                  <div className="flex justify-between items-center mb-4">
                    <h4 className="text-sm font-semibold text-white">Screen Brightness</h4>
                    <span className="text-xs font-mono text-slate-400">{internalBrightness}%</span>
                  </div>
                  <div className="flex items-center gap-4">
                    <Sun size={20} className="text-slate-400" />
                    <input 
                      type="range" 
                      min="0" 
                      max="100" 
                      value={internalBrightness}
                      onChange={(e) => setBrightnessPct('internal', Number(e.target.value))}
                      className="w-full accent-green-500 cursor-pointer h-1.5 bg-slate-800 rounded-lg appearance-none"
                    />
                  </div>
                </div>
              </div>

            </div>
          )}

          {/* TAB 2: DEVICES */}
          {activeTab === 'devices' && (
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
              
              {/* Paired devices list */}
              <div className="glass-panel p-6 rounded-2xl">
                <h3 className="font-bold text-lg text-white mb-1">Paired Companion Devices</h3>
                <p className="text-xs text-slate-400 mb-6">Manage Android devices authorized to control this PC.</p>
                
                <div className="space-y-4">
                  <div className="p-4 rounded-xl border border-slate-800 bg-slate-900/30 flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <div className="w-10 h-10 rounded-lg bg-green-500/10 flex items-center justify-center text-green-400">
                        <Smartphone size={20} />
                      </div>
                      <div>
                        <h4 className="font-semibold text-sm text-white">Pixel 8 Pro</h4>
                        <p className="text-[10px] font-mono text-slate-500">ID: dev_p8_38ef29 | Last seen: 2 mins ago</p>
                      </div>
                    </div>
                    <div className="flex items-center gap-2">
                      <span className="px-2 py-0.5 rounded bg-green-500/20 border border-green-500/30 text-green-400 text-[10px] font-semibold uppercase">Trusted</span>
                      <button className="p-1.5 text-slate-500 hover:text-red-400 hover:bg-red-500/10 rounded transition-colors cursor-pointer">
                        <Trash2 size={14} />
                      </button>
                    </div>
                  </div>

                  <div className="p-4 rounded-xl border border-slate-800 bg-slate-900/30 flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <div className="w-10 h-10 rounded-lg bg-slate-800 flex items-center justify-center text-slate-400">
                        <Smartphone size={20} />
                      </div>
                      <div>
                        <h4 className="font-semibold text-sm text-slate-300">Galaxy Tab S9</h4>
                        <p className="text-[10px] font-mono text-slate-500">ID: dev_gts9_8f0a1b | Last seen: 1 day ago</p>
                      </div>
                    </div>
                    <div className="flex items-center gap-2">
                      <span className="px-2 py-0.5 rounded bg-slate-800 text-slate-400 text-[10px] font-semibold uppercase">Regular</span>
                      <button className="p-1.5 text-slate-500 hover:text-red-400 hover:bg-red-500/10 rounded transition-colors cursor-pointer">
                        <Trash2 size={14} />
                      </button>
                    </div>
                  </div>
                </div>
              </div>

              {/* QR Pairing Code Panel */}
              <div className="glass-panel p-6 rounded-2xl flex flex-col items-center text-center">
                <h3 className="font-bold text-lg text-white mb-1">Pair New Mobile Companion</h3>
                <p className="text-xs text-slate-400 mb-6">Scan the QR code with the PulseLink Android companion app to pair.</p>
                
                {/* Styled CSS QR Code Mockup */}
                <div className="w-48 h-48 bg-white p-3 rounded-2xl border-4 border-slate-800 shadow-xl flex items-center justify-center relative overflow-hidden group">
                  <div className="grid grid-cols-8 grid-rows-8 gap-1 w-full h-full">
                    {Array.from({ length: 64 }).map((_, idx) => {
                      const isFilled = (idx % 3 === 0 || idx % 7 === 0 || idx < 12 || idx > 52 || (idx % 8 === 0 && idx < 32))
                      const isCorner = (idx < 3 && idx % 8 < 3) || (idx > 5 && idx < 8) || (idx > 13 && idx < 16) || (idx > 47 && idx % 8 < 3)
                      return (
                        <div 
                          key={idx} 
                          className={`rounded-sm transition-colors duration-300 ${
                            isCorner ? 'bg-slate-900' :
                            isFilled ? 'bg-slate-800 group-hover:bg-green-600' : 'bg-transparent'
                          }`}
                        />
                      )
                    })}
                  </div>
                  <div className="absolute inset-0 bg-slate-950/80 backdrop-blur-xs flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity">
                    <Key size={32} className="text-green-400 animate-pulse" />
                  </div>
                </div>
                
                <div className="mt-6 space-y-2 w-full text-xs font-mono">
                  <div className="bg-slate-900/60 p-2.5 rounded-lg border border-slate-800 text-slate-300 flex justify-between">
                    <span>Pairing Token:</span>
                    <span className="text-green-400 font-bold select-all">PLS-894-329</span>
                  </div>
                  <p className="text-[10px] text-slate-500 pt-2">The pairing key expires in 10 minutes.</p>
                </div>
              </div>

            </div>
          )}

          {/* TAB 3: MEDIA */}
          {activeTab === 'media' && (
            <div className="space-y-6">
              
              <div className="glass-panel p-8 rounded-2xl max-w-xl mx-auto flex flex-col items-center">
                <div className="w-24 h-24 rounded-full bg-slate-900/80 border border-slate-800/80 flex items-center justify-center text-green-400 mb-6 shadow-inner">
                  <Activity size={40} />
                </div>
                <h3 className="font-bold text-xl text-white mb-1">PC Media Player Controls</h3>
                <p className="text-xs text-slate-400 mb-8">Control media playback and hardware system volume.</p>
                
                {/* Media Playback Keys */}
                <div className="flex items-center gap-6 mb-8">
                  <button 
                    onClick={() => sendRequest("media", "previous")}
                    className="w-12 h-12 rounded-full border border-slate-800 bg-slate-900/60 hover:bg-slate-800 text-slate-300 flex items-center justify-center transition-colors cursor-pointer"
                  >
                    <SkipBack size={18} />
                  </button>
                  <button 
                    onClick={() => sendRequest("media", "play_pause")}
                    className="w-16 h-16 rounded-full bg-green-600 hover:bg-green-500 active:bg-green-700 text-white flex items-center justify-center transition-colors cursor-pointer shadow-lg shadow-green-950/20"
                  >
                    <Play size={24} fill="currentColor" />
                  </button>
                  <button 
                    onClick={() => sendRequest("media", "next")}
                    className="w-12 h-12 rounded-full border border-slate-800 bg-slate-900/60 hover:bg-slate-800 text-slate-300 flex items-center justify-center transition-colors cursor-pointer"
                  >
                    <SkipForward size={18} />
                  </button>
                  <button 
                    onClick={() => sendRequest("media", "stop")}
                    className="w-12 h-12 rounded-full border border-slate-800 bg-slate-900/60 hover:bg-slate-800 text-slate-300 flex items-center justify-center transition-colors cursor-pointer"
                  >
                    <Square size={16} fill="currentColor" />
                  </button>
                </div>

                {/* Detailed Volume Adjustment Slider */}
                <div className="w-full border-t border-slate-800/80 pt-6">
                  <div className="flex justify-between items-center mb-3 text-sm">
                    <span className="text-slate-400 font-medium">Master System Volume</span>
                    <span className="font-mono text-slate-200 font-bold">{volumeLevel}%</span>
                  </div>
                  <div className="flex items-center gap-4">
                    <button 
                      onClick={toggleMute} 
                      className={`p-2 rounded-lg transition-colors cursor-pointer ${
                        isMuted ? 'bg-red-500/10 text-red-400 border border-red-500/20' : 'text-slate-400 hover:text-white'
                      }`}
                    >
                      {isMuted ? <VolumeX size={20} /> : <Volume2 size={20} />}
                    </button>
                    <input 
                      type="range" 
                      min="0" 
                      max="100" 
                      value={volumeLevel}
                      onChange={(e) => setVolumePct(Number(e.target.value))}
                      className="w-full accent-green-500 cursor-pointer h-2 bg-slate-800 rounded-lg appearance-none"
                    />
                  </div>
                </div>

              </div>

            </div>
          )}

          {/* TAB 4: BRIGHTNESS */}
          {activeTab === 'brightness' && (
            <div className="space-y-6 max-w-xl mx-auto">
              
              <div className="glass-panel p-6 rounded-2xl space-y-6">
                <div className="flex items-center gap-3 mb-2">
                  <Sun size={24} className="text-green-400" />
                  <div>
                    <h3 className="font-bold text-lg text-white leading-none">PC Display Brightness</h3>
                    <span className="text-xs text-slate-500">Configure screen backlighting levels.</span>
                  </div>
                </div>

                {/* Laptop Internal Monitor */}
                <div className="space-y-3">
                  <div className="flex justify-between text-sm">
                    <span className="font-medium text-slate-300">Integrated Laptop Display</span>
                    <span className="font-mono text-green-400 font-semibold">{internalBrightness}%</span>
                  </div>
                  <input 
                    type="range" 
                    min="0" 
                    max="100" 
                    value={internalBrightness}
                    onChange={(e) => setBrightnessPct('internal', Number(e.target.value))}
                    className="w-full accent-green-500 cursor-pointer h-2 bg-slate-800 rounded-lg appearance-none"
                  />
                  <p className="text-[10px] text-slate-500">Adjusts the backlight of built-in screens via WMI.</p>
                </div>

                {/* External Display DDC/CI */}
                <div className="space-y-3 border-t border-slate-800/80 pt-6">
                  <div className="flex justify-between text-sm">
                    <span className="font-medium text-slate-300">External Display (DDC/CI)</span>
                    <span className="font-mono text-green-400 font-semibold">{externalBrightness}%</span>
                  </div>
                  <input 
                    type="range" 
                    min="0" 
                    max="100" 
                    value={externalBrightness}
                    onChange={(e) => setBrightnessPct('external', Number(e.target.value))}
                    className="w-full accent-green-500 cursor-pointer h-2 bg-slate-800 rounded-lg appearance-none"
                  />
                  <p className="text-[10px] text-slate-500">Communicates over display cords using I2C/VCP code commands.</p>
                </div>
              </div>

            </div>
          )}

          {/* TAB 5: CLIPBOARD */}
          {activeTab === 'clipboard' && (
            <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
              
              {/* Get & Set Controls */}
              <div className="lg:col-span-2 space-y-6">
                <div className="glass-panel p-6 rounded-2xl">
                  <h3 className="font-bold text-lg text-white mb-1">Set Host Clipboard</h3>
                  <p className="text-xs text-slate-400 mb-4">Paste or type text to immediately copy it onto the Windows PC clipboard.</p>
                  
                  <form onSubmit={handleSetClipboard} className="space-y-3">
                    <textarea 
                      placeholder="Type something to copy to host..."
                      value={clipboardText}
                      onChange={(e) => setClipboardText(e.target.value)}
                      rows={4}
                      className="w-full bg-slate-900 border border-slate-800 rounded-xl p-3.5 text-sm text-slate-300 focus:outline-none focus:border-green-500/50"
                    />
                    <button 
                      type="submit"
                      className="w-full py-2.5 rounded-xl bg-green-600 hover:bg-green-500 active:bg-green-700 text-white font-semibold text-sm transition-colors cursor-pointer flex items-center justify-center gap-2"
                    >
                      <Send size={14} />
                      <span>Push to PC Clipboard</span>
                    </button>
                  </form>
                </div>

                <div className="glass-panel p-6 rounded-2xl">
                  <div className="flex justify-between items-center mb-3">
                    <h3 className="font-bold text-lg text-white">Retrieve PC Clipboard</h3>
                    <button 
                      onClick={() => sendRequest("clipboard", "get")}
                      className="py-1 px-3 rounded-lg bg-slate-800 hover:bg-slate-750 text-xs font-semibold text-slate-300 transition-colors flex items-center gap-1.5 cursor-pointer"
                    >
                      <RefreshCw size={12} />
                      <span>Sync Clipboard</span>
                    </button>
                  </div>
                  
                  {pcClipboardText ? (
                    <div className="bg-slate-900 border border-slate-800 rounded-xl p-4 text-sm font-mono text-slate-300 break-words select-all max-h-48 overflow-y-auto">
                      {pcClipboardText}
                    </div>
                  ) : (
                    <div className="p-8 text-center border border-dashed border-slate-800 rounded-xl text-slate-500 text-xs font-medium">
                      No text currently retrieved. Click 'Sync Clipboard' to query PC clipboard.
                    </div>
                  )}
                </div>
              </div>

              {/* Live clipboard logs */}
              <div className="glass-panel p-6 rounded-2xl flex flex-col h-[400px]">
                <h3 className="font-bold text-sm text-white mb-1">Clipboard Activity Log</h3>
                <span className="text-[10px] text-slate-500 mb-4 block">Tracks real-time copy transactions.</span>
                
                <div className="flex-1 space-y-3 overflow-y-auto pr-1">
                  {clipboardLogs.length > 0 ? (
                    clipboardLogs.map((log, idx) => (
                      <div key={idx} className="p-2.5 rounded-lg bg-slate-900/60 border border-slate-800/80 text-xs font-mono">
                        <div className="flex justify-between text-[9px] text-slate-500 mb-1.5">
                          <span>{log.timestamp}</span>
                          <span className={log.source === 'pc' ? 'text-blue-400' : 'text-green-400'}>
                            {log.source === 'pc' ? 'Copied on PC' : 'Pushed to PC'}
                          </span>
                        </div>
                        <p className="text-slate-300 line-clamp-2 select-all">{log.text}</p>
                      </div>
                    ))
                  ) : (
                    <div className="h-full flex items-center justify-center text-slate-500 text-xs font-medium">
                      No clipboard changes captured.
                    </div>
                  )}
                </div>
              </div>

            </div>
          )}

          {/* TAB 6: NOTIFICATIONS */}
          {activeTab === 'notifications' && (
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
              
              {/* Send Notification Toast */}
              <div className="glass-panel p-6 rounded-2xl">
                <h3 className="font-bold text-lg text-white mb-1">Trigger Toast Alert</h3>
                <p className="text-xs text-slate-400 mb-6">Test the notification bridge by sending a bubble toast to the Windows host.</p>
                
                <form onSubmit={handleSendToast} className="space-y-4">
                  <div className="space-y-1.5">
                    <label className="text-xs font-medium text-slate-400">Notification Title</label>
                    <input 
                      type="text" 
                      value={toastTitle}
                      onChange={(e) => setToastTitle(e.target.value)}
                      className="w-full bg-slate-900 border border-slate-800 rounded-xl px-3 py-2 text-sm text-slate-300 focus:outline-none focus:border-green-500/50"
                    />
                  </div>
                  <div className="space-y-1.5">
                    <label className="text-xs font-medium text-slate-400">Message Description</label>
                    <textarea 
                      value={toastMessage}
                      onChange={(e) => setToastMessage(e.target.value)}
                      rows={3}
                      className="w-full bg-slate-900 border border-slate-800 rounded-xl p-3 text-sm text-slate-300 focus:outline-none focus:border-green-500/50"
                    />
                  </div>
                  <button 
                    type="submit"
                    className="w-full py-2.5 rounded-xl bg-green-600 hover:bg-green-500 active:bg-green-700 text-white font-semibold text-sm transition-colors cursor-pointer flex items-center justify-center gap-2"
                  >
                    <Send size={14} />
                    <span>Send Windows Toast</span>
                  </button>
                </form>
              </div>

              {/* Notification logs history */}
              <div className="glass-panel p-6 rounded-2xl flex flex-col h-[350px]">
                <h3 className="font-bold text-sm text-white mb-1">Toast History</h3>
                <p className="text-[10px] text-slate-500 mb-4">Lists previously pushed alerts.</p>
                
                <div className="flex-1 space-y-3 overflow-y-auto">
                  {toastLogs.length > 0 ? (
                    toastLogs.map((log, idx) => (
                      <div key={idx} className="p-3 rounded-xl bg-slate-900/60 border border-slate-850 flex flex-col gap-1 text-xs">
                        <div className="flex justify-between items-center text-[9px] text-slate-500">
                          <span className="font-mono">{log.timestamp}</span>
                          <span className="text-green-400">Delivered</span>
                        </div>
                        <h4 className="font-bold text-white">{log.title}</h4>
                        <p className="text-slate-400">{log.message}</p>
                      </div>
                    ))
                  ) : (
                    <div className="h-full flex items-center justify-center text-slate-500 text-xs font-medium">
                      No toast history.
                    </div>
                  )}
                </div>
              </div>

            </div>
          )}

          {/* TAB 7: APPS LAUNCHER */}
          {activeTab === 'apps' && (
            <div className="space-y-6">
              
              <div className="glass-panel p-6 rounded-2xl">
                <div className="mb-6">
                  <h3 className="font-bold text-lg text-white leading-tight">Launch Predefined Applications</h3>
                  <p className="text-xs text-slate-400">Trigger standard utilities directly on the Windows host desktop.</p>
                </div>

                <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-5 gap-4">
                  {predefinedAppsList.map((app, idx) => (
                    <button
                      key={idx}
                      onClick={() => sendRequest("apps", "launch", { name: app.name })}
                      className="p-4 border border-slate-800 bg-slate-900/40 hover:bg-slate-850 hover:border-slate-700 hover:shadow-lg transition-all rounded-2xl cursor-pointer flex flex-col items-center gap-3 group text-center"
                    >
                      <div className="w-12 h-12 rounded-xl bg-slate-800 text-green-400 flex items-center justify-center group-hover:scale-105 transition-transform border border-slate-700/50 shadow-inner">
                        <Laptop size={22} />
                      </div>
                      <div>
                        <h4 className="font-bold text-xs text-white leading-none mb-1">{app.name}</h4>
                        <span className="text-[9px] text-slate-500 font-mono truncate max-w-full block">{app.path}</span>
                      </div>
                    </button>
                  ))}
                </div>
              </div>

            </div>
          )}

          {/* TAB 8: TERMINAL LOGS */}
          {activeTab === 'logs' && (
            <div className="space-y-6">
              
              <div className="glass-panel p-6 rounded-2xl flex flex-col h-[500px]">
                <div className="flex justify-between items-center mb-4 shrink-0">
                  <div>
                    <h3 className="font-bold text-lg text-white">Live Transaction Console</h3>
                    <p className="text-xs text-slate-400">Monitors in-app WebSocket packets and API updates.</p>
                  </div>
                  <button 
                    onClick={() => setConsoleLogs([])}
                    className="px-2.5 py-1 rounded bg-red-500/10 hover:bg-red-500/20 text-red-400 text-xs font-semibold transition-colors cursor-pointer border border-red-500/20"
                  >
                    Clear Logs
                  </button>
                </div>

                {/* Log terminal */}
                <div 
                  ref={logTerminalRef}
                  className="flex-1 bg-black/60 border border-slate-900 rounded-xl p-4 font-mono text-[11px] text-green-400 overflow-y-auto leading-relaxed shadow-inner"
                >
                  {consoleLogs.length > 0 ? (
                    consoleLogs.map((log, idx) => (
                      <div key={idx} className="border-b border-slate-900/30 pb-1 last:border-0">{log}</div>
                    ))
                  ) : (
                    <div className="text-slate-600 text-center py-12">Console empty. Ready for transactions.</div>
                  )}
                </div>
              </div>

            </div>
          )}

          {/* TAB 9: SETTINGS */}
          {activeTab === 'settings' && (
            <div className="max-w-2xl mx-auto">
              
              <div className="glass-panel p-6 rounded-2xl">
                <h3 className="font-bold text-lg text-white mb-1 font-sans">Daemon Configuration Settings</h3>
                <p className="text-xs text-slate-400 mb-6">Modify default host properties of the PulseLink backend.</p>
                
                <form onSubmit={saveSettings} className="space-y-5">
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <div className="space-y-1.5">
                      <label className="text-xs font-medium text-slate-400">Device Advertise Name</label>
                      <input 
                        type="text" 
                        value={backendConfig.deviceName}
                        onChange={(e) => setBackendConfig({ ...backendConfig, deviceName: e.target.value })}
                        className="w-full bg-slate-900 border border-slate-800 rounded-xl px-3 py-2 text-sm text-slate-300 focus:outline-none focus:border-green-500/50 font-mono"
                      />
                    </div>
                    <div className="space-y-1.5">
                      <label className="text-xs font-medium text-slate-400">Port Number</label>
                      <input 
                        type="number" 
                        value={backendConfig.server.port}
                        onChange={(e) => setBackendConfig({ 
                          ...backendConfig, 
                          server: { ...backendConfig.server, port: Number(e.target.value) } 
                        })}
                        className="w-full bg-slate-900 border border-slate-800 rounded-xl px-3 py-2 text-sm text-slate-300 focus:outline-none focus:border-green-500/50 font-mono"
                      />
                    </div>
                  </div>

                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4 border-t border-slate-800/80 pt-4">
                    <div className="space-y-1.5">
                      <label className="text-xs font-medium text-slate-400">Log level threshold</label>
                      <select 
                        value={backendConfig.logLevel}
                        onChange={(e) => setBackendConfig({ ...backendConfig, logLevel: e.target.value })}
                        className="w-full bg-slate-900 border border-slate-800 rounded-xl px-3 py-2 text-sm text-slate-300 focus:outline-none focus:border-green-500/50 font-mono"
                      >
                        <option value="debug">debug</option>
                        <option value="info">info</option>
                        <option value="warn">warn</option>
                        <option value="error">error</option>
                      </select>
                    </div>
                    <div className="space-y-1.5">
                      <label className="text-xs font-medium text-slate-400">Database Storage Path</label>
                      <div className="relative">
                        <input 
                          type="text" 
                          readOnly
                          value={backendConfig.databasePath}
                          className="w-full bg-slate-900/60 border border-slate-800 rounded-xl pl-9 pr-3 py-2 text-xs text-slate-400 cursor-not-allowed font-mono"
                        />
                        <Database size={14} className="absolute left-3 top-3 text-slate-600" />
                      </div>
                    </div>
                  </div>

                  <div className="flex items-center gap-3 border-t border-slate-800/80 pt-4">
                    <input 
                      type="checkbox"
                      id="tls-checkbox"
                      checked={backendConfig.server.enableTls}
                      onChange={(e) => setBackendConfig({
                        ...backendConfig,
                        server: { ...backendConfig.server, enableTls: e.target.checked }
                      })}
                      className="w-4 h-4 text-green-600 accent-green-600 border-slate-800 rounded focus:ring-green-500 bg-slate-900"
                    />
                    <label htmlFor="tls-checkbox" className="text-xs text-slate-400 font-medium cursor-pointer selection:bg-transparent">
                      Enable WebSocket TLS encryption (wss:// protocol)
                    </label>
                  </div>

                  <button 
                    type="submit"
                    className="w-full py-2.5 rounded-xl bg-green-600 hover:bg-green-500 active:bg-green-700 text-white font-semibold text-sm transition-colors cursor-pointer"
                  >
                    Save Changes to config.json
                  </button>
                </form>
              </div>

            </div>
          )}

          {/* TAB 10: ABOUT */}
          {activeTab === 'about' && (
            <div className="max-w-xl mx-auto text-center space-y-6">
              
              <div className="glass-panel p-8 rounded-2xl flex flex-col items-center">
                <div className="w-16 h-16 rounded-2xl bg-green-500/10 border border-green-500/30 text-green-400 flex items-center justify-center mb-6">
                  <Activity size={32} />
                </div>
                <h3 className="font-bold text-xl text-white">PulseLink PC Companion Panel</h3>
                <p className="text-xs text-slate-400 mt-1">Cross-device automation and control system.</p>
                
                <div className="mt-8 space-y-3 w-full border-t border-slate-800/80 pt-6 text-xs text-slate-400">
                  <div className="flex justify-between border-b border-slate-900/60 pb-2">
                    <span>Protocol Version</span>
                    <span className="font-mono text-slate-200">1.0</span>
                  </div>
                  <div className="flex justify-between border-b border-slate-900/60 pb-2">
                    <span>UI Build Version</span>
                    <span className="font-mono text-slate-200">0.1.0-dev (React + Tailwind v4)</span>
                  </div>
                  <div className="flex justify-between border-b border-slate-900/60 pb-2">
                    <span>License</span>
                    <span className="font-mono text-slate-200">MIT</span>
                  </div>
                  <div className="flex justify-between">
                    <span>OS Platform Compatibility</span>
                    <span className="font-mono text-slate-200">Windows 10/11</span>
                  </div>
                </div>

                <p className="text-[10px] text-slate-500 mt-8 leading-normal">
                  PulseLink is an advanced agentic coding experiment designed to test secure, headless background services coupled with micro-app interfaces.
                </p>
              </div>

            </div>
          )}

        </div>
      </main>
    </div>
  )
}

export default App
