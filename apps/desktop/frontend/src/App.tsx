import { useState, useEffect } from 'react'
import { AnimatePresence, motion } from 'framer-motion'
import {
  LayoutDashboard, Smartphone, Volume2, Sun, ClipboardList,
  Bell, AppWindow, Terminal, Settings as SettingsIcon, Moon, SunMedium,
} from 'lucide-react'
import { BackendProvider, useBackend } from './lib/backend'
import { Sidebar, type NavItem } from './components/Sidebar'
import { Dashboard } from './panels/Dashboard'
import { MediaVolume } from './panels/MediaVolume'
import { Brightness } from './panels/Brightness'
import { Devices } from './panels/Devices'
import { Clipboard } from './panels/Clipboard'
import { Notifications } from './panels/Notifications'
import { Apps } from './panels/Apps'
import { Logs } from './panels/Logs'
import { Settings } from './panels/Settings'

const NAV: NavItem[] = [
  { id: 'dashboard', label: 'Dashboard', icon: LayoutDashboard },
  { id: 'devices', label: 'Devices', icon: Smartphone },
  { id: 'media', label: 'Media & Volume', icon: Volume2 },
  { id: 'brightness', label: 'Brightness', icon: Sun },
  { id: 'clipboard', label: 'Clipboard', icon: ClipboardList },
  { id: 'notifications', label: 'Notifications', icon: Bell },
  { id: 'apps', label: 'Apps', icon: AppWindow },
  { id: 'logs', label: 'Logs', icon: Terminal },
  { id: 'settings', label: 'Settings', icon: SettingsIcon },
]

const PANELS: Record<string, React.FC> = {
  dashboard: Dashboard,
  devices: Devices,
  media: MediaVolume,
  brightness: Brightness,
  clipboard: Clipboard,
  notifications: Notifications,
  apps: Apps,
  logs: Logs,
  settings: Settings,
}

function Shell() {
  const [active, setActive] = useState('dashboard')
  const { host, port, setHost, setPort, connect, status, error, theme, toggleTheme, pairingRequests, pairingToast, dismissPairingToast, acceptPairing } = useBackend()

  useEffect(() => {
    if (pairingToast.show) {
      const timer = setTimeout(() => dismissPairingToast(), 8000)
      return () => clearTimeout(timer)
    }
  }, [pairingToast, dismissPairingToast])
  const Panel = PANELS[active] ?? Dashboard
  const title = NAV.find((n) => n.id === active)?.label ?? ''

  return (
    <div className="flex h-screen w-screen overflow-hidden text-text">
      <Sidebar items={NAV} active={active} onSelect={setActive} />

      <main className="flex flex-1 flex-col overflow-hidden">
        <header className="flex shrink-0 items-center justify-between gap-4 border-b border-stroke px-6 py-3.5 backdrop-blur-xl">
          <h1 className="text-lg font-semibold text-text">{title}</h1>
          <div className="flex items-center gap-2">
            <input
              value={host}
              onChange={(e) => setHost(e.target.value)}
              placeholder="host"
              aria-label="Backend host"
              className="w-32 rounded-md border border-stroke bg-control px-2.5 py-1.5 text-xs text-text placeholder:text-text-tertiary focus:border-accent focus:outline-none"
            />
            <input
              value={port}
              onChange={(e) => setPort(e.target.value)}
              placeholder="port"
              aria-label="Backend port"
              className="w-16 rounded-md border border-stroke bg-control px-2.5 py-1.5 text-xs text-text placeholder:text-text-tertiary focus:border-accent focus:outline-none"
            />
            <button
              onClick={connect}
              className="rounded-md bg-accent px-3 py-1.5 text-xs font-medium text-on-accent transition-colors hover:bg-accent-hover cursor-pointer"
            >
              {status === 'ready' ? 'Reconnect' : 'Connect'}
            </button>
            <button
              onClick={toggleTheme}
              aria-label="Toggle theme"
              className="grid h-8 w-8 place-items-center rounded-md border border-stroke bg-control text-text-secondary transition-colors hover:bg-control-hover hover:text-text cursor-pointer"
            >
              {theme === 'dark' ? <SunMedium size={15} /> : <Moon size={15} />}
            </button>
          </div>
        </header>

        <div className="flex-1 overflow-y-auto p-6">
          {error && (
            <div className="mb-5 rounded-md border border-danger/40 bg-danger-soft px-4 py-3 text-sm text-danger">{error}</div>
          )}
          <AnimatePresence mode="wait">
            <motion.div
              key={active}
              initial={{ opacity: 0, y: 8 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -6 }}
              transition={{ duration: 0.18, ease: 'easeOut' }}
            >
              <Panel />
            </motion.div>
          </AnimatePresence>
        </div>

        {pairingToast.show && (
          <motion.div
            initial={{ opacity: 0, y: 40, scale: 0.95 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, y: 20, scale: 0.95 }}
            className="pointer-events-none fixed inset-0 z-50 flex items-start justify-center pt-16 sm:pt-20"
          >
            <div className="pointer-events-auto w-full max-w-sm rounded-xl border border-accent/30 bg-card p-4 shadow-2xl backdrop-blur-2xl">
              <div className="flex items-start gap-3">
                <div className="grid h-10 w-10 shrink-0 place-items-center rounded-lg bg-accent-soft text-accent">
                  <Smartphone size={20} />
                </div>
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-semibold text-text">New pairing request</p>
                  <p className="truncate text-xs text-text-secondary mt-0.5">{pairingToast.deviceName}</p>
                  <div className="mt-3 flex gap-2">
                    <button
                      onClick={() => { setActive('devices'); dismissPairingToast() }}
                      className="rounded-md bg-accent px-3 py-1.5 text-xs font-semibold text-on-accent transition-colors hover:bg-accent-hover cursor-pointer"
                    >
                      View
                    </button>
                    <button
                      onClick={() => { acceptPairing(pairingToast.deviceId); dismissPairingToast() }}
                      className="rounded-md border border-stroke bg-control px-3 py-1.5 text-xs font-semibold text-text-secondary transition-colors hover:bg-control-hover cursor-pointer"
                    >
                      Accept
                    </button>
                    <button
                      onClick={dismissPairingToast}
                      className="ml-auto rounded-md bg-transparent px-2 py-1.5 text-xs text-text-tertiary hover:text-text cursor-pointer"
                    >
                      Dismiss
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </motion.div>
        )}
      </main>
    </div>
  )
}

export default function App() {
  return (
    <BackendProvider>
      <Shell />
    </BackendProvider>
  )
}
