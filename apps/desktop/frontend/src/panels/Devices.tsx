import { useEffect, useMemo, useState } from 'react'
import QRCode from 'qrcode'
import { Smartphone, Copy, Check } from 'lucide-react'
import { useBackend } from '../lib/backend'
import { Card, CardHeader, Field, inputCls } from '../components/ui'

// Dev pairing token. Auth is AllowAll server-side today, so this is carried
// end-to-end but not yet enforced — it exists so the QR payload is already the
// real shape the Android client parses.
const DEV_TOKEN = 'desktop-local'

export function Devices() {
  const { config, host, devices, pairingRequests, acceptPairing, rejectPairing } = useBackend()
  const [pairHost, setPairHost] = useState(host)
  const [pairPort, setPairPort] = useState(String(config.server.port || 9843))
  const [qr, setQr] = useState('')
  const [copied, setCopied] = useState(false)

  // pulselink://pair?... — the exact URI the Android scanner decodes.
  const pairUri = useMemo(() => {
    const p = new URLSearchParams({
      host: pairHost,
      port: pairPort,
      token: DEV_TOKEN,
      name: config.deviceName || 'PulseLink-PC',
    })
    return `pulselink://pair?${p.toString()}`
  }, [pairHost, pairPort, config.deviceName])

  useEffect(() => {
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
      {pairingRequests.length > 0 && (
        <Card className="border-accent bg-accent-soft p-5 animate-in fade-in slide-in-from-top-4 duration-250">
          <CardHeader title="Pairing requests" subtitle="New devices requesting to control this PC" />
          <div className="mt-3 divide-y divide-stroke/30">
            {pairingRequests.map((req) => (
              <div key={req.id} className="flex items-center justify-between py-3 first:pt-0 last:pb-0">
                <div className="flex items-center gap-3">
                  <div className="grid h-10 w-10 place-items-center rounded-lg bg-card text-accent">
                    <Smartphone size={20} />
                  </div>
                  <div>
                    <h4 className="text-sm font-semibold text-text">{req.name}</h4>
                    <p className="text-xs text-text-tertiary font-mono break-all">{req.id}</p>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <button
                    onClick={() => acceptPairing(req.id)}
                    className="rounded bg-accent px-3 py-1.5 text-xs font-semibold text-on-accent transition-colors hover:bg-accent-hover cursor-pointer"
                  >
                    Accept
                  </button>
                  <button
                    onClick={() => rejectPairing(req.id)}
                    className="rounded border border-stroke bg-control px-3 py-1.5 text-xs font-semibold text-text-secondary transition-colors hover:bg-control-hover cursor-pointer"
                  >
                    Reject
                  </button>
                </div>
              </div>
            ))}
          </div>
        </Card>
      )}

      <div className="grid grid-cols-1 gap-5 lg:grid-cols-2">
        <Card className="flex flex-col items-center p-6 text-center">
          <CardHeader title="Pair a device" subtitle="Scan with the PulseLink Android app" />
          <div className="rounded-xl border border-stroke bg-white p-3 shadow-[var(--shadow-card)]">
            {qr ? (
              <img src={qr} alt="Pairing QR code" width={220} height={220} />
            ) : (
              <div className="grid h-[220px] w-[220px] place-items-center text-xs text-slate-500">Generating…</div>
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
                  <input className={inputCls} value={pairHost} onChange={(e) => setPairHost(e.target.value)} />
                </Field>
              </div>
              <Field label="Port">
                <input className={inputCls} value={pairPort} onChange={(e) => setPairPort(e.target.value)} />
              </Field>
            </div>
            <Field label="Pairing token">
              <input className={`${inputCls} font-mono`} value={DEV_TOKEN} readOnly />
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
        <CardHeader title="Connected Devices" subtitle="Companion devices currently connected and active" />
        {devices.length === 0 ? (
          <div className="py-8 text-center text-sm text-text-tertiary">
            No devices connected. Scan the QR code or use the pairing URI above to connect.
          </div>
        ) : (
          <div className="divide-y divide-stroke">
            {devices.map((dev) => (
              <div key={dev.id} className="flex items-center justify-between py-3 first:pt-0 last:pb-0">
                <div className="flex items-center gap-3">
                  <div className="grid h-10 w-10 place-items-center rounded-lg bg-control text-accent">
                    <Smartphone size={20} />
                  </div>
                  <div>
                    <h4 className="text-sm font-semibold text-text">{dev.name}</h4>
                    <p className="text-xs text-text-tertiary font-mono break-all">{dev.id}</p>
                  </div>
                </div>
                <div className="flex flex-wrap gap-1 max-w-[55%] justify-end">
                  {dev.capabilities.map((cap) => (
                    <span key={cap} className="rounded bg-control px-1.5 py-0.5 text-[10px] font-medium text-text-secondary">
                      {cap}
                    </span>
                  ))}
                </div>
              </div>
            ))}
          </div>
        )}
      </Card>
    </div>
  )
}
