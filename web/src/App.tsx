import { useCallback, useState } from 'react'
import { AnimatePresence } from 'framer-motion'
import { clearConfig, getApiKey } from './lib/api'
import { Background } from './components/Background'
import { ConnectScreen } from './components/ConnectScreen'
import { Dashboard } from './components/Dashboard'

export default function App() {
  const [connected, setConnected] = useState(() => Boolean(getApiKey()))

  const handleConnect = useCallback(() => setConnected(true), [])
  const handleDisconnect = useCallback(() => {
    clearConfig()
    setConnected(false)
  }, [])

  return (
    <>
      <Background />
      <AnimatePresence mode="wait">
        {connected ? (
          <Dashboard key="dash" onDisconnect={handleDisconnect} />
        ) : (
          <ConnectScreen key="connect" onConnect={handleConnect} />
        )}
      </AnimatePresence>
    </>
  )
}
