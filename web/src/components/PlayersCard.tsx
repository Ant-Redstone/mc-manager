import { motion } from 'framer-motion'
import type { Player } from '../lib/api'
import { Card, Spinner, StatusDot } from './ui'

function Tag({ children, className }: { children: string; className: string }) {
  return <span className={`rounded-md px-2 py-0.5 text-[0.68rem] font-medium ${className}`}>{children}</span>
}

export function PlayersCard({
  players,
  loading,
  error,
  onRefresh,
}: {
  players: Player[]
  loading: boolean
  error: string | null
  onRefresh: () => void
}) {
  const sorted = (Array.isArray(players) ? players : []).slice().sort(
    (a, b) => Number(b.online) - Number(a.online) || a.name.localeCompare(b.name),
  )

  return (
    <Card delay={0.15} className="lg:col-span-3">
      <div className="mb-3 flex items-center justify-between">
        <h2 className="text-sm font-medium uppercase tracking-wide text-text-muted">
          Players <span className="text-text-secondary">· {players.length}</span>
        </h2>
        <div className="flex items-center gap-2">
          {loading && <Spinner />}
          <button
            type="button"
            onClick={onRefresh}
            className="text-xs text-brand-light transition-colors hover:text-brand"
          >
            refresh
          </button>
        </div>
      </div>

      {error && <p className="text-sm text-status-error">{error}</p>}
      {!error && sorted.length === 0 && <p className="text-sm text-text-muted">No players known yet.</p>}

      <ul className="grid grid-cols-1 gap-1.5 sm:grid-cols-2">
        {sorted.map((p, i) => (
          <motion.li
            key={p.uuid || p.name}
            initial={{ opacity: 0, x: -8 }}
            animate={{ opacity: 1, x: 0 }}
            transition={{ delay: Math.min(i * 0.025, 0.25) }}
            className="flex items-center justify-between rounded-lg bg-white/[0.03] px-3 py-2"
          >
            <span className="flex items-center gap-2.5 text-sm">
              <StatusDot on={p.online} />
              {p.name}
            </span>
            <span className="flex gap-1.5">
              {p.is_op && <Tag className="bg-brand/15 text-brand-light">op</Tag>}
              {p.is_whitelisted && <Tag className="bg-status-success/15 text-status-success">wl</Tag>}
              {p.is_banned && <Tag className="bg-status-error/15 text-status-error">ban</Tag>}
            </span>
          </motion.li>
        ))}
      </ul>
    </Card>
  )
}
