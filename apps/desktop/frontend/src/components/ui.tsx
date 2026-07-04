import { motion } from 'framer-motion'
import type { ButtonHTMLAttributes, ReactNode } from 'react'

// Fluent card surface — translucent Mica fill with a hairline stroke.
export function Card({
  children,
  className = '',
  hover = false,
}: {
  children: ReactNode
  className?: string
  hover?: boolean
}) {
  return (
    <div
      className={`rounded-lg border border-stroke bg-card shadow-[var(--shadow-card)] backdrop-blur-xl ${
        hover ? 'transition-colors duration-200 hover:bg-card-hover hover:border-stroke-strong' : ''
      } ${className}`}
    >
      {children}
    </div>
  )
}

export function CardHeader({ title, subtitle, right }: { title: string; subtitle?: string; right?: ReactNode }) {
  return (
    <div className="flex items-start justify-between gap-4 mb-4">
      <div>
        <h3 className="text-base font-semibold text-text leading-tight">{title}</h3>
        {subtitle && <p className="text-xs text-text-tertiary mt-0.5">{subtitle}</p>}
      </div>
      {right}
    </div>
  )
}

type Variant = 'accent' | 'standard' | 'subtle' | 'danger'

const variantCls: Record<Variant, string> = {
  accent: 'bg-accent text-on-accent hover:bg-accent-hover border-transparent',
  standard: 'bg-control text-text hover:bg-control-hover border-stroke',
  subtle: 'bg-transparent text-text hover:bg-control border-transparent',
  danger: 'bg-danger-soft text-danger hover:brightness-110 border-transparent',
}

export function Button({
  children,
  variant = 'standard',
  className = '',
  ...rest
}: { children: ReactNode; variant?: Variant } & ButtonHTMLAttributes<HTMLButtonElement>) {
  return (
    <button
      {...rest}
      className={`inline-flex items-center justify-center gap-2 rounded-md border px-4 py-2 text-sm font-medium transition-colors duration-150 cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed ${variantCls[variant]} ${className}`}
    >
      {children}
    </button>
  )
}

export function IconButton({
  children,
  label,
  active = false,
  className = '',
  ...rest
}: { children: ReactNode; label: string; active?: boolean } & ButtonHTMLAttributes<HTMLButtonElement>) {
  return (
    <button
      {...rest}
      aria-label={label}
      title={label}
      className={`inline-flex items-center justify-center rounded-md border transition-colors duration-150 cursor-pointer ${
        active
          ? 'bg-accent-soft border-accent text-accent'
          : 'bg-control border-stroke text-text-secondary hover:bg-control-hover hover:text-text'
      } ${className}`}
    >
      {children}
    </button>
  )
}

export function Toggle({ checked, onChange, label }: { checked: boolean; onChange: (v: boolean) => void; label: string }) {
  return (
    <button
      role="switch"
      aria-checked={checked}
      aria-label={label}
      onClick={() => onChange(!checked)}
      className={`relative h-5 w-10 shrink-0 rounded-full border transition-colors duration-200 cursor-pointer ${
        checked ? 'bg-accent border-accent' : 'bg-control border-stroke-strong'
      }`}
    >
      <span
        className={`absolute top-1/2 h-3 w-3 -translate-y-1/2 rounded-full transition-all duration-200 ${
          checked ? 'left-[22px] bg-on-accent' : 'left-1 bg-text-tertiary'
        }`}
      />
    </button>
  )
}

export function StatTile({
  label,
  value,
  icon,
  accent = false,
}: {
  label: string
  value: ReactNode
  icon: ReactNode
  accent?: boolean
}) {
  return (
    <Card className="p-5">
      <div className="flex items-start justify-between">
        <span className="text-xs font-medium uppercase tracking-wide text-text-tertiary">{label}</span>
        <span className={`grid h-9 w-9 place-items-center rounded-lg ${accent ? 'bg-accent-soft text-accent' : 'bg-control text-text-secondary'}`}>
          {icon}
        </span>
      </div>
      <div className="mt-3 text-2xl font-semibold text-text tabular-nums">{value}</div>
    </Card>
  )
}

export function Meter({ value, label }: { value: number; label?: string }) {
  return (
    <div>
      {label && (
        <div className="mb-1 flex justify-between text-xs text-text-tertiary">
          <span>{label}</span>
          <span className="tabular-nums">{Math.round(value)}%</span>
        </div>
      )}
      <div className="h-1.5 w-full overflow-hidden rounded-full bg-stroke-strong">
        <motion.div
          className="h-full rounded-full bg-accent"
          animate={{ width: `${Math.max(0, Math.min(100, value))}%` }}
          transition={{ type: 'spring', stiffness: 200, damping: 30 }}
        />
      </div>
    </div>
  )
}

export function Badge({ children, tone = 'neutral' }: { children: ReactNode; tone?: 'neutral' | 'ok' | 'danger' }) {
  const cls = {
    neutral: 'bg-control text-text-secondary border-stroke',
    ok: 'bg-[color:var(--accent-soft)] text-accent border-accent',
    danger: 'bg-danger-soft text-danger border-transparent',
  }[tone]
  return (
    <span className={`inline-flex items-center rounded-full border px-2 py-0.5 text-[11px] font-medium ${cls}`}>
      {children}
    </span>
  )
}

export function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <label className="block space-y-1.5">
      <span className="text-xs font-medium text-text-secondary">{label}</span>
      {children}
    </label>
  )
}

export const inputCls =
  'w-full rounded-md border border-stroke bg-control px-3 py-2 text-sm text-text placeholder:text-text-tertiary transition-colors focus:border-accent focus:outline-none'
