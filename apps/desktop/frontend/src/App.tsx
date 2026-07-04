import { useState } from 'react'
import { AnimatePresence, motion } from 'framer-motion'
import {
  LayoutDashboard, Smartphone, Volume2, Sun, ClipboardList,
  Bell, AppWindow, Terminal, Settings as SettingsIcon, Moon, SunMedium,
} from 'lucide-react'
import { BackendProvider, useBackend } from './lib/backend'
import { Sidebar, type NavItem } from './components/Sidebar'
import { PairingDialog } from './components/PairingDialog'
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
  const { host, port, setHost, setPort, connect, status, error, theme, toggleTheme } = useBackend()

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
      </main>

      {/* Global pairing dialog — renders on top of everything */}
      <PairingDialog />
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
