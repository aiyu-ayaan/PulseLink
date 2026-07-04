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
  const { config, host } = useBackend()
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
          <p className="text-xs text-text-tertiary">
            Live device management (trusted list, revoke) arrives with server-side pairing enforcement — tracked as a follow-up.
          </p>
        </div>
      </Card>
    </div>
  )
}
