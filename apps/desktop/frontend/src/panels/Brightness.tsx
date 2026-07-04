import { Sun, Monitor } from 'lucide-react'
import { useBackend } from '../lib/backend'
import { Card, CardHeader } from '../components/ui'

export function Brightness() {
  const { brightness, send } = useBackend()

  const rows = [
    { key: 'internal' as const, label: 'Integrated display', hint: 'Laptop backlight via WMI', value: brightness.internal },
    { key: 'external' as const, label: 'External display', hint: 'DDC/CI over the display cable', value: brightness.external },
  ]

  return (
    <div className="mx-auto max-w-2xl">
      <Card className="p-6">
        <CardHeader title="Display brightness" subtitle="Adjust backlight for each connected monitor" />
        <div className="space-y-6">
          {rows.map((r) => (
            <div key={r.key}>
              <div className="mb-2 flex items-center justify-between">
                <span className="flex items-center gap-2 text-sm font-medium text-text">
                  {r.key === 'internal' ? <Monitor size={16} /> : <Sun size={16} />}
                  {r.label}
                </span>
                <span className="tabular-nums text-sm text-accent">{r.value}%</span>
              </div>
              <input
                type="range"
                min={0}
                max={100}
                value={r.value}
                onChange={(e) => send('brightness', 'set', { type: r.key, level: Number(e.target.value) })}
                className="w-full"
                aria-label={r.label}
              />
              <p className="mt-1.5 text-xs text-text-tertiary">{r.hint}</p>
            </div>
          ))}
        </div>
      </Card>
    </div>
  )
}
