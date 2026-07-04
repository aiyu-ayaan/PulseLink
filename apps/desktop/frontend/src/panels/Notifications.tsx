import { useState } from 'react'
import { Send, Bell } from 'lucide-react'
import { useBackend } from '../lib/backend'
import { Card, CardHeader, Field, Button, inputCls } from '../components/ui'

export function Notifications() {
  const { send } = useBackend()
  const [title, setTitle] = useState('PulseLink')
  const [message, setMessage] = useState('Hello from the desktop panel!')
  const [sent, setSent] = useState<{ ts: string; title: string; message: string }[]>([])

  const fire = (e: React.FormEvent) => {
    e.preventDefault()
    send('notification', 'toast', { title, message })
    setSent((p) => [{ ts: new Date().toLocaleTimeString(), title, message }, ...p.slice(0, 19)])
  }

  return (
    <div className="grid grid-cols-1 gap-5 lg:grid-cols-2">
      <Card className="p-6">
        <CardHeader title="Send a toast" subtitle="Posts a Windows notification on the PC" />
        <form onSubmit={fire} className="space-y-4">
          <Field label="Title">
            <input className={inputCls} value={title} onChange={(e) => setTitle(e.target.value)} />
          </Field>
          <Field label="Message">
            <textarea rows={3} className={`${inputCls} resize-none`} value={message} onChange={(e) => setMessage(e.target.value)} />
          </Field>
          <Button type="submit" variant="accent" className="w-full"><Send size={15} /> Send toast</Button>
        </form>
      </Card>

      <Card className="flex h-[360px] flex-col p-5">
        <CardHeader title="Sent history" />
        <div className="flex-1 space-y-2 overflow-y-auto pr-1">
          {sent.length ? (
            sent.map((n, i) => (
              <div key={i} className="rounded-md border border-stroke bg-control p-3 text-xs">
                <div className="mb-1 flex items-center justify-between text-[10px] text-text-tertiary">
                  <span className="flex items-center gap-1"><Bell size={11} /> {n.ts}</span>
                  <span className="text-ok">delivered</span>
                </div>
                <div className="font-medium text-text">{n.title}</div>
                <p className="text-text-secondary">{n.message}</p>
              </div>
            ))
          ) : (
            <div className="grid h-full place-items-center text-xs text-text-tertiary">No toasts sent yet.</div>
          )}
        </div>
      </Card>
    </div>
  )
}
