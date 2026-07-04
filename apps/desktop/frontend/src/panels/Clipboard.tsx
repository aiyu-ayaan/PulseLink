import { useState } from 'react'
import { Send, RefreshCw } from 'lucide-react'
import { useBackend } from '../lib/backend'
import { Card, CardHeader, Button } from '../components/ui'

export function Clipboard() {
  const { pcClipboard, clipEvents, send, pushClipEvent } = useBackend()
  const [text, setText] = useState('')

  const push = (e: React.FormEvent) => {
    e.preventDefault()
    if (!text.trim()) return
    send('clipboard', 'set', { text })
    pushClipEvent({ ts: new Date().toLocaleTimeString(), text, source: 'you' })
    setText('')
  }

  return (
    <div className="grid grid-cols-1 gap-5 lg:grid-cols-3">
      <div className="space-y-5 lg:col-span-2">
        <Card className="p-6">
          <CardHeader title="Push to PC clipboard" subtitle="Copies this text onto the connected Windows clipboard" />
          <form onSubmit={push} className="space-y-3">
            <textarea
              rows={4}
              value={text}
              onChange={(e) => setText(e.target.value)}
              placeholder="Type or paste text to send…"
              className="w-full resize-none rounded-md border border-stroke bg-control p-3 text-sm text-text placeholder:text-text-tertiary focus:border-accent focus:outline-none"
            />
            <Button type="submit" variant="accent" className="w-full"><Send size={15} /> Push to PC</Button>
          </form>
        </Card>

        <Card className="p-6">
          <CardHeader
            title="PC clipboard"
            right={
              <button onClick={() => send('clipboard', 'get')} className="flex items-center gap-1.5 rounded-md border border-stroke bg-control px-2.5 py-1 text-xs text-text-secondary transition-colors hover:bg-control-hover cursor-pointer">
                <RefreshCw size={12} /> Sync
              </button>
            }
          />
          {pcClipboard ? (
            <div className="max-h-48 overflow-y-auto rounded-md border border-stroke bg-control p-3 font-mono text-sm text-text break-words">{pcClipboard}</div>
          ) : (
            <div className="rounded-md border border-dashed border-stroke p-8 text-center text-xs text-text-tertiary">Press Sync to read the PC clipboard.</div>
          )}
        </Card>
      </div>

      <Card className="flex h-[440px] flex-col p-5">
        <CardHeader title="Activity" subtitle="Recent clipboard changes" />
        <div className="flex-1 space-y-2 overflow-y-auto pr-1">
          {clipEvents.length ? (
            clipEvents.map((e, i) => (
              <div key={i} className="rounded-md border border-stroke bg-control p-2.5 text-xs">
                <div className="mb-1 flex justify-between text-[10px] text-text-tertiary">
                  <span>{e.ts}</span>
                  <span className={e.source === 'pc' ? 'text-accent' : 'text-ok'}>{e.source === 'pc' ? 'from PC' : 'sent'}</span>
                </div>
                <p className="line-clamp-2 text-text-secondary">{e.text}</p>
              </div>
            ))
          ) : (
            <div className="grid h-full place-items-center text-xs text-text-tertiary">No activity yet.</div>
          )}
        </div>
      </Card>
    </div>
  )
}
