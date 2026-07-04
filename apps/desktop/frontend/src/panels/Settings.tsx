import { useEffect, useState } from 'react'
import QRCode from 'qrcode'
import {
  Save, Database, Smartphone, Copy, Check, Shield, ShieldCheck, ShieldX,
  Sliders, Radio, Users, CheckSquare
} from 'lucide-react'
import { useBackend, type BackendConfig } from '../lib/backend'
import { Card, CardHeader, Field, Toggle, Button, inputCls } from '../components/ui'

export function Settings() {
  const {
    config,
    host,
    pairingInfo,
    deviceHistory,
    pairingRequests,
    acceptPairing,
    rejectPairing,
    send,
  } = useBackend()

  const [activeTab, setActiveTab] = useState<'general' | 'permissions' | 'pairing' | 'devices'>('general')
  const [draft, setDraft] = useState<BackendConfig>(config)
  const [pairHost, setPairHost] = useState(host)
  const [pairPort, setPairPort] = useState(String(config.server.port || 9843))
  const [qr, setQr] = useState('')
  const [copied, setCopied] = useState(false)

  const pairUri = pairingInfo?.uri || ''

  // Sync draft configuration with backend when it changes
  useEffect(() => {
    setDraft(config)
  }, [config])

  // Fetch fresh pairing details on load
  useEffect(() => {
    send('pairing', 'info')
    send('pairing', 'pending')
    send('devices', 'history')
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  useEffect(() => {
    if (!pairingInfo) return
    setPairHost(pairingInfo.host)
    setPairPort(String(pairingInfo.port))
    const msLeft = pairingInfo.expiresAt * 1000 - Date.now()
    if (msLeft <= 0) return
    const t = setTimeout(() => send('pairing', 'info'), msLeft)
    return () => clearTimeout(t)
  }, [pairingInfo, send])

  useEffect(() => {
    if (!pairUri) {
      setQr('')
      return
    }
    QRCode.toDataURL(pairUri, { margin: 1, width: 200, errorCorrectionLevel: 'M' })
      .then(setQr)
      .catch(() => setQr(''))
  }, [pairUri])

  const copy = () => {
    navigator.clipboard?.writeText(pairUri)
    setCopied(true)
    setTimeout(() => setCopied(false), 1500)
  }

  const save = (e: React.FormEvent) => {
    e.preventDefault()
    send('settings', 'set', draft)
  }

  const togglePermission = (key: keyof BackendConfig['permissions']) => {
    const nextPerms = { ...draft.permissions, [key]: !draft.permissions[key] }
    setDraft({ ...draft, permissions: nextPerms })
  }

  const tabs = [
    { id: 'general' as const, label: 'General', icon: Sliders },
    { id: 'permissions' as const, label: 'Permissions', icon: CheckSquare },
    { id: 'pairing' as const, label: 'Pair Device', icon: Radio },
    { id: 'devices' as const, label: 'Devices History', icon: Users, badge: pairingRequests.length },
  ]

  return (
    <div className="mx-auto max-w-4xl space-y-6">
      {/* Sub tabs nav */}
      <div className="flex gap-1.5 border-b border-stroke pb-1">
        {tabs.map((t) => {
          const Icon = t.icon
          const isActive = activeTab === t.id
          return (
            <button
              key={t.id}
              onClick={() => setActiveTab(t.id)}
              className={`flex items-center gap-2 rounded-t-lg border-b-2 px-4 py-2.5 text-sm font-medium transition-all duration-150 cursor-pointer ${
                isActive
                  ? 'border-accent text-accent bg-accent-soft/20'
                  : 'border-transparent text-text-secondary hover:text-text hover:bg-control/30'
              }`}
            >
              <Icon size={16} />
              <span>{t.label}</span>
              {t.badge && t.badge > 0 ? (
                <span className="ml-1 rounded-full bg-accent px-1.5 py-0.5 text-[10px] font-bold text-on-accent animate-pulse">
                  {t.badge}
                </span>
              ) : null}
            </button>
          )
        })}
      </div>

      <div className="mt-4">
        {/* GENERAL SETTINGS */}
        {activeTab === 'general' && (
          <Card className="p-6">
            <CardHeader title="General Configuration" subtitle="Primary backend port, device name, and logging parameters" />
            <form onSubmit={save} className="space-y-5">
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                <Field label="Device name">
                  <input
                    className={inputCls}
                    value={draft.deviceName}
                    onChange={(e) => setDraft({ ...draft, deviceName: e.target.value })}
                  />
                </Field>
                <Field label="Port">
                  <input
                    type="number"
                    className={inputCls}
                    value={draft.server.port}
                    onChange={(e) =>
                      setDraft({ ...draft, server: { ...draft.server, port: Number(e.target.value) } })
                    }
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
                      <option key={l} value={l}>
                        {l}
                      </option>
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
                  <div className="text-sm font-medium text-text">TLS Encryption</div>
                  <div className="text-xs text-text-tertiary">Serve over wss:// with a self-signed certificate</div>
                </div>
                <Toggle
                  label="Enable TLS"
                  checked={draft.server.enableTls}
                  onChange={(v) => setDraft({ ...draft, server: { ...draft.server, enableTls: v } })}
                />
              </div>

              <div className="flex items-center justify-between border-t border-stroke pt-4">
                <div>
                  <div className="text-sm font-medium text-text">Maximum Compatibility Mode</div>
                  <div className="text-xs text-text-tertiary">
                    Use software color Gamma Ramps instead of hardware DDC/CI commands for display brightness
                  </div>
                </div>
                <Toggle
                  label="Enable Compatibility Mode"
                  checked={draft.maxCompatibilityMode}
                  onChange={(v) => setDraft({ ...draft, maxCompatibilityMode: v })}
                />
              </div>

              <Button type="submit" variant="accent" className="w-full">
                <Save size={15} /> Save changes
              </Button>
            </form>
          </Card>
        )}

        {/* FEATURE PERMISSIONS */}
        {activeTab === 'permissions' && (
          <Card className="p-6">
            <CardHeader
              title="Device Feature Permissions"
              subtitle="Control which companion features mobile devices are allowed to request/control"
            />
            <form onSubmit={save} className="space-y-4">
              <div className="divide-y divide-stroke">
                {[
                  { key: 'media' as const, title: 'Media Session Control', desc: 'Allows controlling media playback (play, pause, skip)' },
                  { key: 'volume' as const, title: 'Volume Control', desc: 'Allows modifying system volume levels and muting audio' },
                  { key: 'brightness' as const, title: 'Brightness Control', desc: 'Allows adjusting display brightness (integrated and external)' },
                  { key: 'clipboard' as const, title: 'Clipboard Sync', desc: 'Allows reading/writing desktop clipboard contents' },
                  { key: 'notifications' as const, title: 'Notifications Forwarding', desc: 'Allows client connections to receive system notifications' },
                  { key: 'apps' as const, title: 'Application Management', desc: 'Allows launching configured applications' },
                  { key: 'power' as const, title: 'Power Options', desc: 'Allows putting PC to sleep or locking the user session' },
                  { key: 'sysinfo' as const, title: 'System Resource Monitoring', desc: 'Allows reading CPU, RAM, battery, and screen configurations' },
                  { key: 'input' as const, title: 'Remote Input Control', desc: 'Allows simulated mouse and keyboard input' },
                  { key: 'filetransfer' as const, title: 'File Transfers', desc: 'Allows sending and receiving files' },
                ].map((item) => (
                  <div key={item.key} className="flex items-center justify-between py-3.5 first:pt-0 last:pb-0">
                    <div>
                      <div className="text-sm font-medium text-text">{item.title}</div>
                      <div className="text-xs text-text-tertiary">{item.desc}</div>
                    </div>
                    <Toggle
                      label={`Allow ${item.title}`}
                      checked={!!draft.permissions?.[item.key]}
                      onChange={() => togglePermission(item.key)}
                    />
                  </div>
                ))}
              </div>

              <Button type="submit" variant="accent" className="w-full border-t border-stroke pt-4">
                <Save size={15} /> Save changes
              </Button>
            </form>
          </Card>
        )}

        {/* DEVICE PAIRING */}
        {activeTab === 'pairing' && (
          <div className="grid grid-cols-1 gap-5 lg:grid-cols-2">
            <Card className="flex flex-col items-center p-6 text-center">
              <CardHeader title="Pair a Device" subtitle="Scan this QR code with the PulseLink Android companion app" />
              <div className="rounded-xl border border-stroke bg-white p-3 shadow-[var(--shadow-card)]">
                {qr ? (
                  <img src={qr} alt="Pairing QR code" width={200} height={200} />
                ) : (
                  <div className="grid h-[200px] w-[200px] place-items-center text-xs text-slate-500">
                    Establishing connection...
                  </div>
                )}
              </div>
              <button
                onClick={copy}
                className="mt-4 flex items-center gap-2 rounded-md border border-stroke bg-control px-3 py-1.5 text-xs font-medium text-text-secondary transition-colors hover:bg-control-hover cursor-pointer"
              >
                {copied ? <Check size={13} /> : <Copy size={13} />}
                {copied ? 'Copied' : 'Copy pairing link'}
              </button>
            </Card>

            <Card className="p-6">
              <CardHeader title="Connection details" subtitle="Enter the PC's LAN address, or type it manually on the phone" />
              <div className="space-y-4">
                <div className="grid grid-cols-3 gap-3">
                  <div className="col-span-2">
                    <Field label="Host / IP">
                      <input className={inputCls} value={pairHost} readOnly />
                    </Field>
                  </div>
                  <Field label="Port">
                    <input className={inputCls} value={pairPort} readOnly />
                  </Field>
                </div>
                <Field label="Pairing token">
                  <input className={`${inputCls} font-mono`} value={pairingInfo?.token || ''} readOnly />
                </Field>
                <div className="rounded-md border border-stroke bg-control/60 p-3">
                  <div className="mb-1 flex items-center gap-2 text-xs font-medium text-text-secondary">
                    <Smartphone size={14} /> Pairing link
                  </div>
                  <code className="block break-all text-[11px] text-text-tertiary">{pairUri}</code>
                </div>
              </div>
            </Card>
          </div>
        )}

        {/* DEVICE HISTORY & MANAGEMENT */}
        {activeTab === 'devices' && (
          <Card className="p-6">
            <CardHeader title="Device History & Management" subtitle="Manage trusted and pending companion devices" />
            {deviceHistory.length === 0 ? (
              <div className="py-8 text-center text-sm text-text-tertiary">
                No devices registered. Scan the QR code or use the pairing URI in the "Pair Device" tab to connect a companion device.
              </div>
            ) : (
              <div className="divide-y divide-stroke">
                {deviceHistory.map((dev) => {
                  const isPending = !dev.trusted
                  return (
                    <div
                      key={dev.id}
                      className={`flex flex-col sm:flex-row sm:items-center justify-between gap-3 py-3.5 first:pt-0 last:pb-0 ${
                        isPending ? 'bg-accent-soft/10 -mx-6 px-6 py-4 rounded-lg my-1' : ''
                      }`}
                    >
                      <div className="flex items-start gap-3">
                        <div className={`grid h-10 w-10 place-items-center rounded-lg ${
                          isPending ? 'bg-warn-soft text-warn' : 'bg-control text-ok'
                        }`}>
                          {isPending ? <Shield size={20} className="animate-pulse" /> : <ShieldCheck size={20} />}
                        </div>
                        <div>
                          <div className="flex flex-wrap items-center gap-2">
                            <h4 className="text-sm font-semibold text-text">{dev.name}</h4>
                            <span className="flex items-center gap-1 text-[10px] font-medium">
                              <span
                                className={`h-1.5 w-1.5 rounded-full ${
                                  dev.online ? 'bg-ok animate-pulse' : 'bg-text-tertiary'
                                }`}
                              />
                              <span className={dev.online ? 'text-ok font-semibold' : 'text-text-tertiary'}>
                                {dev.online ? 'Online' : 'Offline'}
                              </span>
                            </span>
                            {isPending && (
                              <span className="rounded bg-warn-soft/30 px-1.5 py-0.5 text-[9px] font-semibold text-warn">
                                Pending approval
                              </span>
                            )}
                          </div>
                          <p className="text-xs text-text-tertiary font-mono break-all">{dev.id}</p>
                          {dev.lastSeen && (
                            <p className="text-[10px] text-text-tertiary mt-0.5">
                              Last seen: {new Date(dev.lastSeen * 1000).toLocaleString()}
                            </p>
                          )}
                        </div>
                      </div>

                      <div className="flex items-center gap-2 self-end sm:self-center">
                        {isPending ? (
                          <>
                            <button
                              onClick={() => acceptPairing(dev.id)}
                              className="inline-flex items-center gap-1.5 rounded-md bg-accent px-3 py-1.5 text-xs font-semibold text-on-accent transition-colors hover:bg-accent-hover active:scale-[0.97] cursor-pointer"
                            >
                              <ShieldCheck size={13} />
                              Accept
                            </button>
                            <button
                              onClick={() => rejectPairing(dev.id)}
                              className="inline-flex items-center gap-1.5 rounded-md border border-stroke bg-control px-3 py-1.5 text-xs font-semibold text-text-secondary transition-colors hover:bg-danger hover:text-white hover:border-danger active:scale-[0.97] cursor-pointer"
                            >
                              <ShieldX size={13} />
                              Reject
                            </button>
                          </>
                        ) : (
                          <>
                            <div className="hidden md:flex flex-wrap gap-1 mr-3 max-w-[200px] justify-end">
                              {dev.capabilities.filter(c => c !== 'pairing').map((cap) => (
                                <span
                                  key={cap}
                                  className="rounded bg-control px-1.5 py-0.5 text-[9px] font-medium text-text-tertiary"
                                >
                                  {cap}
                                </span>
                              ))}
                            </div>
                            <button
                              onClick={() => rejectPairing(dev.id)}
                              className="inline-flex items-center gap-1.5 rounded-md border border-stroke bg-control px-3 py-1.5 text-xs font-medium text-text-secondary transition-all hover:bg-danger hover:text-white hover:border-danger active:scale-[0.97] cursor-pointer"
                            >
                              Unpair
                            </button>
                          </>
                        )}
                      </div>
                    </div>
                  )
                })}
              </div>
            )}
          </Card>
        )}
      </div>
    </div>
  )
}
