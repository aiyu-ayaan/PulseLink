import { SkipBack, SkipForward, Play, Square, Volume2, VolumeX } from 'lucide-react'
import { useBackend } from '../lib/backend'
import { Card, CardHeader, IconButton } from '../components/ui'

export function MediaVolume() {
  const { volume, send } = useBackend()

  return (
    <div className="mx-auto max-w-2xl space-y-5">
      <Card className="p-6">
        <CardHeader title="Media playback" subtitle="Controls the active media session on the PC" />
        <div className="flex items-center justify-center gap-4 py-4">
          <IconButton label="Previous" onClick={() => send('media', 'previous')} className="h-12 w-12">
            <SkipBack size={20} />
          </IconButton>
          <button
            onClick={() => send('media', 'play_pause')}
            aria-label="Play or pause"
            className="grid h-16 w-16 place-items-center rounded-full bg-accent text-on-accent transition-colors hover:bg-accent-hover cursor-pointer"
          >
            <Play size={26} fill="currentColor" />
          </button>
          <IconButton label="Next" onClick={() => send('media', 'next')} className="h-12 w-12">
            <SkipForward size={20} />
          </IconButton>
          <IconButton label="Stop" onClick={() => send('media', 'stop')} className="h-12 w-12">
            <Square size={18} fill="currentColor" />
          </IconButton>
        </div>
      </Card>

      <Card className="p-6">
        <CardHeader title="Master volume" right={<span className="tabular-nums text-sm text-text-secondary">{volume.level}%</span>} />
        <div className="flex items-center gap-4">
          <IconButton label={volume.muted ? 'Unmute' : 'Mute'} active={volume.muted} onClick={() => send('volume', 'mute')} className="h-10 w-10">
            {volume.muted ? <VolumeX size={18} /> : <Volume2 size={18} />}
          </IconButton>
          <input
            type="range"
            min={0}
            max={100}
            value={volume.level}
            onChange={(e) => send('volume', 'set', { level: Number(e.target.value) })}
            className="w-full"
            aria-label="Master volume"
          />
        </div>
        <div className="mt-4 flex gap-2">
          <button onClick={() => send('volume', 'down')} className="flex-1 rounded-md border border-stroke bg-control py-2 text-sm text-text-secondary transition-colors hover:bg-control-hover cursor-pointer">Volume −</button>
          <button onClick={() => send('volume', 'up')} className="flex-1 rounded-md border border-stroke bg-control py-2 text-sm text-text-secondary transition-colors hover:bg-control-hover cursor-pointer">Volume +</button>
        </div>
      </Card>
    </div>
  )
}
