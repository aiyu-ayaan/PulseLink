import { Cpu, MemoryStick, BatteryCharging, Battery, Monitor, Lock, Moon, Volume2, VolumeX, ClipboardCopy } from 'lucide-react'
import { useBackend } from '../lib/backend'
import { Card, CardHeader, StatTile, Meter, IconButton } from '../components/ui'

export function Dashboard() {
  const { sysInfo, volume, brightness, send } = useBackend()
  const ramUsedPct = sysInfo ? Math.round(((sysInfo.ramTotal - sysInfo.ramFree) / sysInfo.ramTotal) * 100) : 0

  const actions = [
    { label: 'Lock PC', icon: <Lock size={18} />, onClick: () => send('power', 'lock') },
    { label: 'Sleep', icon: <Moon size={18} />, onClick: () => send('power', 'sleep') },
    {
      label: volume.muted ? 'Unmute' : 'Mute',
      icon: volume.muted ? <VolumeX size={18} /> : <Volume2 size={18} />,
      onClick: () => send('volume', 'mute'),
      active: volume.muted,
    },
    { label: 'Sync Clipboard', icon: <ClipboardCopy size={18} />, onClick: () => send('clipboard', 'get') },
  ]

  return (
    <div className="space-y-5">
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <StatTile label="Host" value={<span className="text-lg">{sysInfo?.hostname || '—'}</span>} icon={<Monitor size={18} />} accent />
        <StatTile label="CPU" value={`${sysInfo?.cpuUsage ?? 0}%`} icon={<Cpu size={18} />} />
        <StatTile label="Memory" value={`${ramUsedPct}%`} icon={<MemoryStick size={18} />} />
        <StatTile
          label="Battery"
          value={`${sysInfo?.batteryPct ?? 100}%`}
          icon={sysInfo?.isCharging ? <BatteryCharging size={18} /> : <Battery size={18} />}
          accent={sysInfo?.isCharging}
        />
      </div>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
        <Card className="p-5 lg:col-span-2">
          <CardHeader title="Quick actions" subtitle="One-tap controls for the connected PC" />
          <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
            {actions.map((a) => (
              <button
                key={a.label}
                onClick={a.onClick}
                className={`flex flex-col items-center gap-2.5 rounded-lg border p-4 text-center transition-colors duration-150 cursor-pointer ${
                  a.active
                    ? 'border-accent bg-accent-soft'
                    : 'border-stroke bg-control hover:bg-control-hover'
                }`}
              >
                <span className={`grid h-10 w-10 place-items-center rounded-lg ${a.active ? 'text-accent' : 'text-text-secondary'} bg-card`}>
                  {a.icon}
                </span>
                <span className="text-xs font-medium text-text">{a.label}</span>
              </button>
            ))}
          </div>
        </Card>

        <Card className="p-5">
          <CardHeader title="Live resources" />
          <div className="space-y-4">
            <Meter label="CPU usage" value={sysInfo?.cpuUsage ?? 0} />
            <Meter label="Memory used" value={ramUsedPct} />
            <div className="flex justify-between border-t border-stroke pt-3 text-xs text-text-tertiary">
              <span>Displays</span>
              <span className="tabular-nums text-text-secondary">{sysInfo?.monitorCount ?? 1}</span>
            </div>
          </div>
        </Card>
      </div>

      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        <Card className="p-5">
          <CardHeader title="Volume" right={<span className="tabular-nums text-sm text-text-secondary">{volume.level}%</span>} />
          <div className="flex items-center gap-3">
            <IconButton label="Mute" active={volume.muted} onClick={() => send('volume', 'mute')} className="h-9 w-9">
              {volume.muted ? <VolumeX size={16} /> : <Volume2 size={16} />}
            </IconButton>
            <input
              type="range"
              min={0}
              max={100}
              value={volume.level}
              onChange={(e) => send('volume', 'set', { level: Number(e.target.value) })}
              className="w-full"
              aria-label="System volume"
            />
          </div>
        </Card>

        <Card className="p-5">
          <CardHeader title="Brightness" right={<span className="tabular-nums text-sm text-text-secondary">{brightness.internal}%</span>} />
          <div className="flex items-center gap-3">
            <span className="grid h-9 w-9 place-items-center rounded-md bg-control text-text-secondary"><Monitor size={16} /></span>
            <input
              type="range"
              min={0}
              max={100}
              value={brightness.internal}
              onChange={(e) => send('brightness', 'set', { type: 'internal', level: Number(e.target.value) })}
              className="w-full"
              aria-label="Screen brightness"
            />
          </div>
        </Card>
      </div>
    </div>
  )
}
