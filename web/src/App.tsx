import { useCallback, useState } from 'react'
import { clearConfig, getApiKey } from './lib/api'
import { Background } from './components/Background'
import { ConnectScreen } from './components/ConnectScreen'
import { Dashboard } from './components/Dashboard'
import { ErrorBoundary } from './components/ErrorBoundary'

export default function App() {
  const [connected, setConnected] = useState(() => Boolean(getApiKey()))

  const handleConnect = useCallback(() => setConnected(true), [])
  const handleDisconnect = useCallback(() => {
    clearConfig()
    setConnected(false)
  }, [])

  // NOTE: rendered as a plain conditional rather than wrapped in
  // <AnimatePresence mode="wait">. With React 19 + framer-motion the
  // "wait" exit of one custom-component screen didn't complete, so the
  // incoming screen never mounted (blank after clicking Connect). Each
  // screen still plays its own entrance animation on mount.
  return (
    <ErrorBoundary>
      <Background />
      {connected ? (
        <Dashboard onDisconnect={handleDisconnect} />
      ) : (
        <ConnectScreen onConnect={handleConnect} />
      )}
    </ErrorBoundary>
  )
}
