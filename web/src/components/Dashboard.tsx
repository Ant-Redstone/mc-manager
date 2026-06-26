import { useCallback, useState } from 'react'
import { motion } from 'framer-motion'
import { useConsole, usePlayers, useServerStatus } from '../lib/hooks'
import { Button, Logo, Toast } from './ui'
import { StatusCard } from './StatusCard'
import { ConsoleCard } from './ConsoleCard'
import { PlayersCard } from './PlayersCard'

export function Dashboard({ onDisconnect }: { onDisconnect: () => void }) {
  const [toast, setToast] = useState<string | null>(null)
  const showError = useCallback((msg: string) => setToast(msg), [])

  const { status, loading, refresh: refreshStatus } = useServerStatus(true, onDisconnect)
  const running = status?.running ?? false
  const { players, loading: playersLoading, error: playersError, refresh: refreshPlayers } = usePlayers(true)
  const { lines, connected, send, clear } = useConsole(running)

  const refreshAll = useCallback(() => {
    void refreshStatus()
    void refreshPlayers()
  }, [refreshStatus, refreshPlayers])

  return (
    <motion.div
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
      className="mx-auto max-w-6xl px-4 py-6 sm:px-6 sm:py-8"
    >
      <header className="mb-6 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Logo />
          <h1 className="text-lg font-semibold sm:text-xl">MC Manager</h1>
        </div>
        <Button variant="ghost" onClick={onDisconnect} className="min-h-0 px-3 py-2 text-xs">
          Change key
        </Button>
      </header>

      <div className="grid grid-cols-1 gap-4 sm:gap-5 lg:grid-cols-3">
        <StatusCard status={status} loading={loading} onChanged={refreshAll} onError={showError} />
        <ConsoleCard
          lines={lines}
          connected={connected}
          running={running}
          onSend={send}
          onClear={clear}
        />
        <PlayersCard
          players={players}
          loading={playersLoading}
          error={playersError}
          onRefresh={refreshPlayers}
        />
      </div>

      <Toast message={toast} onDone={() => setToast(null)} />
    </motion.div>
  )
}
