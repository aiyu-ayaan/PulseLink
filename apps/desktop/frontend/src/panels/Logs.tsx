import { useEffect, useRef } from 'react'
import { Trash2 } from 'lucide-react'
import { useBackend } from '../lib/backend'
import { Card, CardHeader } from '../components/ui'

export function Logs() {
  const { logs, clearLogs } = useBackend()
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (ref.current) ref.current.scrollTop = ref.current.scrollHeight
  }, [logs])

  return (
    <Card className="flex h-[calc(100vh-9rem)] flex-col p-5">
      <CardHeader
        title="Console"
        subtitle="Live WebSocket traffic"
        right={
          <button onClick={clearLogs} className="flex items-center gap-1.5 rounded-md border border-transparent bg-danger-soft px-2.5 py-1 text-xs font-medium text-danger transition hover:brightness-110 cursor-pointer">
            <Trash2 size={12} /> Clear
          </button>
        }
      />
      <div ref={ref} className="flex-1 overflow-y-auto rounded-md border border-stroke bg-control p-3 font-mono text-[11px] leading-relaxed text-text-secondary">
        {logs.length ? (
          logs.map((l, i) => <div key={i} className="whitespace-pre-wrap">{l}</div>)
        ) : (
          <div className="grid h-full place-items-center text-text-tertiary">Console is empty.</div>
        )}
      </div>
    </Card>
  )
}
