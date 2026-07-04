import { useEffect, useState } from 'react'
import QRCode from 'qrcode'
import { Smartphone, Copy, Check, Shield, ShieldCheck, ShieldX } from 'lucide-react'
import { useBackend } from '../lib/backend'
import { Card, CardHeader, Field, inputCls } from '../components/ui'

export function Devices() {
  const { config, host, pairingInfo, deviceHistory, pairingRequests, acceptPairing, rejectPairing, send } = useBackend()
  const [pairHost, setPairHost] = useState(host)
  const [pairPort, setPairPort] = useState(String(config.server.port || 9843))
  const [qr, setQr] = useState('')
  const [copied, setCopied] = useState(false)

  const pairUri = pairingInfo?.uri || ''

  // Fetch a fresh token whenever this panel is opened — the one issued at
  // connect time may already be expired (10 min TTL) or already used.
  useEffect(() => {
    send('pairing', 'info')
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  useEffect(() => {
    if (!pairingInfo) return
    setPairHost(pairingInfo.host)
    setPairPort(String(pairingInfo.port))
    // Auto-refresh once this token's 10-minute TTL runs out, so a QR left
    // on screen doesn't silently go dead.
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
    QRCode.toDataURL(pairUri, { margin: 1, width: 240, errorCorrectionLevel: 'M' })
      .then(setQr)
      .catch(() => setQr(''))
  }, [pairUri])

  const copy = () => {
    navigator.clipboard?.writeText(pairUri)
    setCopied(true)
    setTimeout(() => setCopied(false), 1500)
  }

  return (
    <div className="space-y-5">
      {/* Pending pairing info banner */}
      {pairingRequests.length > 0 && (
        <div className="flex items-center gap-3 rounded-lg border border-accent/30 bg-accent-soft px-4 py-3 animate-in fade-in slide-in-from-top-2 duration-200">
          <Shield size={18} className="text-accent shrink-0" />
          <p className="text-sm text-text-secondary">
            <span className="font-semibold text-accent">{pairingRequests.length}</span>{' '}
            device{pairingRequests.length > 1 ? 's' : ''} waiting for approval.
            You can accept or reject them directly in the list below or via the pop-up dialog.
          </p>
        </div>
      )}

      <div className="grid grid-cols-1 gap-5 lg:grid-cols-2">
        <Card className="flex flex-col items-center p-6 text-center">
          <CardHeader title="Pair a device" subtitle="Scan with the PulseLink Android app" />
          <div className="rounded-xl border border-stroke bg-white p-3 shadow-[var(--shadow-card)]">
            {qr ? (
              <img src={qr} alt="Pairing QR code" width={220} height={220} />
            ) : (
              <div className="grid h-[220px] w-[220px] place-items-center text-xs text-slate-500">Connect backend first</div>
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

      <Card className="p-6">
        <CardHeader title="Device History & Management" subtitle="Manage trusted and pending companion devices" />
        {deviceHistory.length === 0 ? (
          <div className="py-8 text-center text-sm text-text-tertiary">
            No devices registered. Scan the QR code or use the pairing URI above to connect a companion device.
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
                        {/* Capabilities preview */}
                        <div className="hidden md:flex flex-wrap gap-1 mr-3 max-w-[200px] justify-end">
                          {dev.capabilities.filter(c => c !== 'pairing').map((cap) => (
                            <span
                              key={cap}
                              className="rounded bg-control px-1.5 py-0.5 text-[9px] font-medium text-text-tertiary animate-in fade-in duration-200"
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
    </div>
  )
}
