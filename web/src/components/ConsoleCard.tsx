import { useEffect, useRef, useState, type FormEvent } from 'react'
import type { ConsoleLine } from '../lib/hooks'
import { Badge, Button, Card } from './ui'

export function ConsoleCard({
  lines,
  connected,
  running,
  onSend,
  onClear,
}: {
  lines: ConsoleLine[]
  connected: boolean
  running: boolean
  onSend: (cmd: string) => boolean
  onClear: () => void
}) {
  const [cmd, setCmd] = useState('')
  const scrollRef = useRef<HTMLDivElement>(null)
  const atBottom = useRef(true)

  useEffect(() => {
    const el = scrollRef.current
    if (el && atBottom.current) el.scrollTop = el.scrollHeight
  }, [lines])

  function handleScroll() {
    const el = scrollRef.current
    if (el) atBottom.current = el.scrollHeight - el.scrollTop - el.clientHeight < 48
  }

  function submit(e: FormEvent) {
    e.preventDefault()
    const c = cmd.trim()
    if (c && onSend(c)) setCmd('')
  }

  return (
    <Card delay={0.1} className="flex flex-col lg:col-span-2">
      <div className="mb-3 flex items-center justify-between">
        <h2 className="text-sm font-medium uppercase tracking-wide text-text-muted">Console</h2>
        <div className="flex items-center gap-3">
          <Badge tone={connected ? 'success' : 'neutral'}>{connected ? 'live' : 'offline'}</Badge>
          <button
            type="button"
            onClick={onClear}
            className="text-xs text-text-muted transition-colors hover:text-text-secondary"
          >
            clear
          </button>
        </div>
      </div>

      <div
        ref={scrollRef}
        onScroll={handleScroll}
        className="h-72 grow overflow-y-auto rounded-xl border border-white/5 bg-black/40 p-3 font-mono text-xs leading-relaxed sm:h-80"
      >
        {lines.length === 0 ? (
          <p className="text-text-muted">{running ? 'Waiting for output…' : 'Server is offline.'}</p>
        ) : (
          lines.map((l) => (
            <div key={l.id} className={l.text.startsWith('> ') ? 'text-brand-light' : 'text-text-secondary'}>
              {l.text}
            </div>
          ))
        )}
      </div>

      <form onSubmit={submit} className="mt-3 flex gap-2">
        <input
          value={cmd}
          onChange={(e) => setCmd(e.target.value)}
          disabled={!connected}
          placeholder={connected ? 'Send a command, e.g. say hello' : 'Console offline'}
          className="glass-input min-w-0 flex-1 px-3.5 py-2.5 font-mono text-sm placeholder:text-text-muted disabled:opacity-50"
        />
        <Button type="submit" variant="primary" disabled={!connected || !cmd.trim()}>
          Send
        </Button>
      </form>
    </Card>
  )
}
