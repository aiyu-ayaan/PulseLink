import { motion } from 'framer-motion'
import { RefreshCw } from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import { useBackend } from '../lib/backend'

export interface NavItem {
  id: string
  label: string
  icon: LucideIcon
}

const statusMeta: Record<string, { label: string; color: string }> = {
  ready: { label: 'Connected', color: 'var(--ok)' },
  connected: { label: 'Handshaking', color: 'var(--accent)' },
  connecting: { label: 'Connecting', color: 'var(--warn)' },
  disconnected: { label: 'Offline', color: 'var(--danger)' },
}

export function Sidebar({
  items,
  active,
  onSelect,
}: {
  items: NavItem[]
  active: string
  onSelect: (id: string) => void
}) {
  const { status, connect } = useBackend()
  const meta = statusMeta[status]

  return (
    <aside className="flex w-60 shrink-0 flex-col border-r border-stroke bg-sidebar backdrop-blur-2xl">
      <div className="flex items-center gap-3 px-5 py-5">
        <span className="grid h-9 w-9 place-items-center rounded-lg bg-accent text-on-accent font-bold">P</span>
        <div>
          <div className="text-sm font-semibold leading-tight text-text">PulseLink</div>
          <div className="text-[11px] text-text-tertiary">Desktop Companion</div>
        </div>
      </div>

      <nav className="flex-1 space-y-0.5 overflow-y-auto px-2.5 py-2">
        {items.map((item) => {
          const Icon = item.icon
          const isActive = active === item.id
          return (
            <button
              key={item.id}
              onClick={() => onSelect(item.id)}
              className={`relative flex w-full items-center gap-3 rounded-md px-3 py-2 text-sm transition-colors duration-150 cursor-pointer ${
                isActive ? 'bg-control text-text font-medium' : 'text-text-secondary hover:bg-control/60 hover:text-text'
              }`}
            >
              {isActive && (
                <motion.span
                  layoutId="nav-indicator"
                  className="absolute left-0 top-1/2 h-4 w-1 -translate-y-1/2 rounded-full bg-accent"
                />
              )}
              <Icon size={18} strokeWidth={isActive ? 2.2 : 1.8} />
              <span>{item.label}</span>
            </button>
          )
        })}
      </nav>

      <div className="border-t border-stroke p-3">
        <div className="mb-2 flex items-center justify-between px-1">
          <span className="text-[11px] text-text-tertiary">Backend</span>
          <span className="flex items-center gap-1.5 text-[11px] font-medium text-text-secondary">
            <span className="h-2 w-2 rounded-full" style={{ backgroundColor: meta.color }} />
            {meta.label}
          </span>
        </div>
        <button
          onClick={connect}
          className="flex w-full items-center justify-center gap-2 rounded-md border border-stroke bg-control px-3 py-1.5 text-xs font-medium text-text-secondary transition-colors hover:bg-control-hover hover:text-text cursor-pointer"
        >
          <RefreshCw size={13} className={status === 'connecting' ? 'animate-spin' : ''} />
          Reconnect
        </button>
      </div>
    </aside>
  )
}
