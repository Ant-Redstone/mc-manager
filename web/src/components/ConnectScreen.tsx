import { useState, type FormEvent } from 'react'
import { motion } from 'framer-motion'
import { getApiBase, getApiKey, setApiBase, setApiKey } from '../lib/api'
import { Button, Logo } from './ui'

export function ConnectScreen({ onConnect }: { onConnect: () => void }) {
  const [base, setBase] = useState(getApiBase())
  const [key, setKey] = useState(getApiKey())
  const [error, setError] = useState<string | null>(null)

  function submit(e: FormEvent) {
    e.preventDefault()
    if (!key.trim()) {
      setError('An API key is required')
      return
    }
    setApiBase(base.trim())
    setApiKey(key.trim())
    onConnect()
  }

  return (
    <motion.div
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
      className="flex min-h-screen items-center justify-center p-4"
    >
      <motion.form
        onSubmit={submit}
        initial={{ opacity: 0, y: 24, scale: 0.97 }}
        animate={{ opacity: 1, y: 0, scale: 1 }}
        transition={{ duration: 0.45, ease: 'easeOut' }}
        className="glass-card w-full max-w-md p-7 sm:p-9"
      >
        <div className="mb-7 flex items-center gap-3">
          <Logo />
          <div>
            <h1 className="text-xl font-semibold">MC Manager</h1>
            <p className="text-sm text-text-secondary">Connect to your server API</p>
          </div>
        </div>

        <label className="mb-4 block">
          <span className="mb-1.5 block text-sm font-medium text-text-secondary">API URL</span>
          <input
            value={base}
            onChange={(e) => setBase(e.target.value)}
            placeholder="blank = same origin (http://host:8080)"
            className="glass-input w-full px-3.5 py-2.5 text-sm placeholder:text-text-muted"
          />
        </label>

        <label className="mb-6 block">
          <span className="mb-1.5 block text-sm font-medium text-text-secondary">API key</span>
          <input
            type="password"
            value={key}
            onChange={(e) => {
              setKey(e.target.value)
              setError(null)
            }}
            placeholder="your API key"
            autoFocus
            className="glass-input w-full px-3.5 py-2.5 text-sm placeholder:text-text-muted"
          />
        </label>

        {error && <p className="mb-4 text-sm text-status-error">{error}</p>}

        <Button type="submit" variant="primary" className="w-full">
          Connect
        </Button>
      </motion.form>
    </motion.div>
  )
}
