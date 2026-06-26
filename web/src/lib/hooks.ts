import { useCallback, useEffect, useRef, useState } from 'react'
import {
  ApiError,
  consoleWebSocketUrl,
  getStatus,
  listPlayers,
  type Player,
  type StatusData,
} from './api'

const MAX_LINES = 1000
const POLL_MS = 5000

export interface ConsoleLine {
  id: number
  text: string
}

/** Polls /api/status while enabled. Calls onAuthError on a 401 (bad key). */
export function useServerStatus(enabled: boolean, onAuthError: () => void) {
  const [status, setStatus] = useState<StatusData | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  const refresh = useCallback(async () => {
    setLoading(true)
    try {
      setStatus(await getStatus())
      setError(null)
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        onAuthError()
        return
      }
      setError(err instanceof Error ? err.message : 'failed to load status')
    } finally {
      setLoading(false)
    }
  }, [onAuthError])

  useEffect(() => {
    if (!enabled) return
    void refresh()
    const id = window.setInterval(() => void refresh(), POLL_MS)
    return () => clearInterval(id)
  }, [enabled, refresh])

  return { status, error, loading, refresh }
}

/** Loads the player list. A missing usercache (500) is treated as "no players". */
export function usePlayers(enabled: boolean) {
  const [players, setPlayers] = useState<Player[]>([])
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  const refresh = useCallback(async () => {
    setLoading(true)
    try {
      setPlayers(await listPlayers())
      setError(null)
    } catch (err) {
      if (err instanceof ApiError && err.status === 500) {
        setPlayers([])
        setError(null)
      } else {
        setError(err instanceof Error ? err.message : 'failed to load players')
      }
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    if (enabled) void refresh()
  }, [enabled, refresh])

  return { players, error, loading, refresh }
}

/** Opens the console WebSocket while the server is running and buffers lines. */
export function useConsole(running: boolean) {
  const [lines, setLines] = useState<ConsoleLine[]>([])
  const [connected, setConnected] = useState(false)
  const wsRef = useRef<WebSocket | null>(null)
  const counter = useRef(0)

  const append = useCallback((text: string) => {
    setLines((prev) => {
      const next = [...prev, { id: counter.current++, text }]
      return next.length > MAX_LINES ? next.slice(next.length - MAX_LINES) : next
    })
  }, [])

  useEffect(() => {
    if (!running) {
      setConnected(false)
      return
    }
    const socket = new WebSocket(consoleWebSocketUrl())
    wsRef.current = socket
    socket.onopen = () => setConnected(true)
    socket.onmessage = (e) => {
      if (typeof e.data === 'string') append(e.data)
    }
    socket.onclose = () => {
      setConnected(false)
      if (wsRef.current === socket) wsRef.current = null
    }
    return () => socket.close()
  }, [running, append])

  const send = useCallback(
    (cmd: string): boolean => {
      const ws = wsRef.current
      if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(cmd)
        append(`> ${cmd}`)
        return true
      }
      return false
    },
    [append],
  )

  const clear = useCallback(() => setLines([]), [])

  return { lines, connected, send, clear }
}
