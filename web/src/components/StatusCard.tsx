import { useState } from 'react'
import { startServer, stopServer, type StatusData } from '../lib/api'
import { Button, Card, Spinner, StatusDot } from './ui'

function formatUptime(seconds: number): string {
  const h = Math.floor(seconds / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  const s = seconds % 60
  if (h) return `${h}h ${m}m`
  if (m) return `${m}m ${s}s`
  return `${s}s`
}

export function StatusCard({
  status,
  loading,
  onChanged,
  onError,
}: {
  status: StatusData | null
  loading: boolean
  onChanged: () => void
  onError: (msg: string) => void
}) {
  const [busy, setBusy] = useState<'start' | 'stop' | null>(null)
  const running = status?.running ?? false

  async function act(which: 'start' | 'stop') {
    setBusy(which)
    try {
      await (which === 'start' ? startServer() : stopServer())
    } catch (err) {
      onError(err instanceof Error ? err.message : `failed to ${which} server`)
    } finally {
      setBusy(null)
      onChanged()
    }
  }

  return (
    <Card delay={0.05}>
      <div className="flex items-start justify-between">
        <div>
          <h2 className="text-sm font-medium uppercase tracking-wide text-text-muted">Server</h2>
          <div className="mt-2 flex items-center gap-2.5">
            <StatusDot on={running} />
            <span className="text-2xl font-semibold">
              {status ? (running ? 'Running' : 'Stopped') : '—'}
            </span>
          </div>
        </div>
        {loading && <Spinner />}
      </div>

      {running && (status?.uptime_seconds !== undefined || status?.pid !== undefined) && (
        <div className="mt-4 flex flex-wrap gap-x-6 gap-y-1 text-sm text-text-secondary">
          {status?.uptime_seconds !== undefined && (
            <span>
              uptime <b className="font-medium text-text-primary">{formatUptime(status.uptime_seconds)}</b>
            </span>
          )}
          {status?.pid !== undefined && (
            <span>
              pid <b className="font-medium text-text-primary">{status.pid}</b>
            </span>
          )}
        </div>
      )}

      <div className="mt-5 flex gap-3">
        <Button variant="primary" disabled={running || busy !== null} onClick={() => void act('start')}>
          {busy === 'start' && <Spinner />}
          Start
        </Button>
        <Button variant="danger" disabled={!running || busy !== null} onClick={() => void act('stop')}>
          {busy === 'stop' && <Spinner />}
          Stop
        </Button>
      </div>
    </Card>
  )
}
