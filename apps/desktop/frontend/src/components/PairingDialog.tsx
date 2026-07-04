import { AnimatePresence, motion } from 'framer-motion'
import { Smartphone, ShieldCheck, ShieldX } from 'lucide-react'
import { useBackend, type DeviceInfo } from '../lib/backend'

/**
 * PairingDialog — a global floating overlay that appears when a new device
 * requests to pair. Shows a Fluent-style glass modal with device info and
 * Accept / Reject actions. Renders on top of everything regardless of which
 * panel the user is viewing.
 */
export function PairingDialog() {
  const { pairingRequests, acceptPairing, rejectPairing } = useBackend()

  return (
    <AnimatePresence>
      {pairingRequests.length > 0 && (
        <motion.div
          key="pairing-overlay"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          transition={{ duration: 0.22 }}
          className="fixed inset-0 z-[100] flex items-center justify-center"
        >
          {/* Backdrop */}
          <div className="absolute inset-0 bg-black/40 backdrop-blur-sm" />

          {/* Dialog stack — shows all pending requests */}
          <motion.div
            initial={{ opacity: 0, scale: 0.92, y: 20 }}
            animate={{ opacity: 1, scale: 1, y: 0 }}
            exit={{ opacity: 0, scale: 0.92, y: 20 }}
            transition={{ type: 'spring', damping: 28, stiffness: 340 }}
            className="relative z-10 flex w-full max-w-md flex-col gap-3 px-4"
          >
            {pairingRequests.map((req, i) => (
              <PairingCard
                key={req.id}
                device={req}
                index={i}
                total={pairingRequests.length}
                onAccept={() => acceptPairing(req.id)}
                onReject={() => rejectPairing(req.id)}
              />
            ))}
          </motion.div>
        </motion.div>
      )}
    </AnimatePresence>
  )
}

function PairingCard({
  device,
  index,
  total,
  onAccept,
  onReject,
}: {
  device: DeviceInfo
  index: number
  total: number
  onAccept: () => void
  onReject: () => void
}) {
  return (
    <motion.div
      layout
      initial={{ opacity: 0, y: 16, scale: 0.96 }}
      animate={{ opacity: 1, y: 0, scale: 1 }}
      exit={{ opacity: 0, y: -12, scale: 0.96 }}
      transition={{ delay: index * 0.06, type: 'spring', damping: 26, stiffness: 300 }}
      className="rounded-2xl border border-stroke-strong bg-[var(--card)] p-5 shadow-[var(--shadow-flyout)] backdrop-blur-2xl"
    >
      {/* Header */}
      <div className="flex items-start gap-4">
        {/* Icon */}
        <div className="grid h-12 w-12 shrink-0 place-items-center rounded-xl bg-accent-soft text-accent">
          <Smartphone size={24} />
        </div>

        {/* Info */}
        <div className="flex-1 min-w-0">
          <p className="text-[13px] font-semibold text-accent uppercase tracking-wider">
            Pairing Request {total > 1 ? `(${index + 1}/${total})` : ''}
          </p>
          <h3 className="mt-1 text-lg font-bold text-text truncate">{device.name}</h3>
          <p className="mt-0.5 text-xs text-text-tertiary font-mono break-all">{device.id}</p>
        </div>
      </div>

      {/* Description */}
      <p className="mt-4 text-sm text-text-secondary leading-relaxed">
        This device wants to connect and control your PC. Only accept if you recognize
        this device.
      </p>

      {/* Capabilities preview */}
      {device.capabilities && device.capabilities.length > 0 && (
        <div className="mt-3 flex flex-wrap gap-1.5">
          {device.capabilities.filter(c => c !== 'pairing').map((cap) => (
            <span
              key={cap}
              className="rounded-md bg-control px-2 py-0.5 text-[10px] font-medium text-text-tertiary"
            >
              {cap}
            </span>
          ))}
        </div>
      )}

      {/* Actions */}
      <div className="mt-5 flex items-center gap-2.5">
        <button
          onClick={onAccept}
          className="flex flex-1 items-center justify-center gap-2 rounded-lg bg-accent px-4 py-2.5 text-sm font-semibold text-on-accent transition-all hover:bg-accent-hover hover:shadow-lg active:scale-[0.97] cursor-pointer"
        >
          <ShieldCheck size={16} />
          Accept
        </button>
        <button
          onClick={onReject}
          className="flex flex-1 items-center justify-center gap-2 rounded-lg border border-stroke bg-control px-4 py-2.5 text-sm font-semibold text-text-secondary transition-all hover:bg-danger hover:text-white hover:border-danger active:scale-[0.97] cursor-pointer"
        >
          <ShieldX size={16} />
          Reject
        </button>
      </div>
    </motion.div>
  )
}
