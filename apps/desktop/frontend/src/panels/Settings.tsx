import { useEffect, useState } from 'react'
import { Save, Database } from 'lucide-react'
import { useBackend, type BackendConfig } from '../lib/backend'
import { Card, CardHeader, Field, Toggle, Button, inputCls } from '../components/ui'

export function Settings() {
  const { config, send } = useBackend()
  const [draft, setDraft] = useState<BackendConfig>(config)

  // Keep the form in sync when the backend delivers the live config.
  useEffect(() => setDraft(config), [config])

  const save = (e: React.FormEvent) => {
    e.preventDefault()
    send('settings', 'set', draft)
  }

  return (
    <div className="mx-auto max-w-2xl">
      <Card className="p-6">
        <CardHeader title="Backend configuration" subtitle="Writes to config.json on save" />
        <form onSubmit={save} className="space-y-5">
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <Field label="Device name">
              <input className={inputCls} value={draft.deviceName} onChange={(e) => setDraft({ ...draft, deviceName: e.target.value })} />
            </Field>
            <Field label="Port">
              <input
                type="number"
                className={inputCls}
                value={draft.server.port}
                onChange={(e) => setDraft({ ...draft, server: { ...draft.server, port: Number(e.target.value) } })}
              />
            </Field>
          </div>

          <div className="grid grid-cols-1 gap-4 border-t border-stroke pt-4 sm:grid-cols-2">
            <Field label="Log level">
              <select
                className={inputCls}
                value={draft.logLevel}
                onChange={(e) => setDraft({ ...draft, logLevel: e.target.value })}
              >
                {['debug', 'info', 'warn', 'error'].map((l) => (
                  <option key={l} value={l}>{l}</option>
                ))}
              </select>
            </Field>
            <Field label="Database path">
              <div className="relative">
                <Database size={14} className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-text-tertiary" />
                <input className={`${inputCls} cursor-not-allowed pl-9 font-mono text-xs`} value={draft.databasePath} readOnly />
              </div>
            </Field>
          </div>

          <div className="flex items-center justify-between border-t border-stroke pt-4">
            <div>
              <div className="text-sm font-medium text-text">TLS encryption</div>
              <div className="text-xs text-text-tertiary">Serve over wss:// with a self-signed certificate</div>
            </div>
            <Toggle
              label="Enable TLS"
              checked={draft.server.enableTls}
              onChange={(v) => setDraft({ ...draft, server: { ...draft.server, enableTls: v } })}
            />
          </div>

          <Button type="submit" variant="accent" className="w-full"><Save size={15} /> Save changes</Button>
        </form>
      </Card>
    </div>
  )
}
