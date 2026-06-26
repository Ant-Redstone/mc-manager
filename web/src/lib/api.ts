// Typed client for the mc-manager API.

export interface ApiResponse<T> {
  success: boolean
  data?: T
  error?: string
}

export interface StatusData {
  running: boolean
  // Optional fields — present when the API exposes process monitoring.
  pid?: number
  uptime_seconds?: number
  started_at?: string
}

export interface Player {
  uuid: string
  name: string
  online: boolean
  is_op: boolean
  is_banned: boolean
  is_whitelisted: boolean
}

export class ApiError extends Error {
  status: number
  constructor(message: string, status: number) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

const KEY_STORAGE = 'mcm.apiKey'
const BASE_STORAGE = 'mcm.apiBase'

export function getApiKey(): string {
  return localStorage.getItem(KEY_STORAGE) ?? ''
}

export function setApiKey(key: string): void {
  localStorage.setItem(KEY_STORAGE, key)
}

// Base URL of the API. An empty string means "same origin", which works
// behind the Vite dev proxy and when the API serves the built frontend.
export function getApiBase(): string {
  return localStorage.getItem(BASE_STORAGE) ?? ''
}

export function setApiBase(base: string): void {
  let b = base.trim().replace(/\/+$/, '')
  // Default a bare host (e.g. "mine.lomokwa.com") to https, otherwise the
  // value is treated as a relative path and the request hits this origin.
  if (b && !/^[a-z]+:\/\//i.test(b)) b = `https://${b}`
  localStorage.setItem(BASE_STORAGE, b)
}

export function clearConfig(): void {
  localStorage.removeItem(KEY_STORAGE)
  localStorage.removeItem(BASE_STORAGE)
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers)
  headers.set('X-API-Key', getApiKey())
  if (init.body) headers.set('Content-Type', 'application/json')

  let res: Response
  try {
    res = await fetch(getApiBase() + path, { ...init, headers })
  } catch (err) {
    throw new ApiError(`network error: ${(err as Error).message}`, 0)
  }

  let body: ApiResponse<T> | null = null
  try {
    body = (await res.json()) as ApiResponse<T>
  } catch {
    // Non-JSON response (e.g. a proxy error) — handled below.
  }

  if (!res.ok || (body !== null && body.success === false)) {
    throw new ApiError(body?.error ?? `request failed (${res.status})`, res.status)
  }
  return body?.data as T
}

export function getStatus(): Promise<StatusData> {
  return request<StatusData>('/api/status')
}

export function startServer(): Promise<unknown> {
  return request<unknown>('/api/start', {
    method: 'POST',
    body: JSON.stringify({ createLaunchScript: false, configureProperties: false, properties: {} }),
  })
}

export function stopServer(): Promise<unknown> {
  return request<unknown>('/api/stop', { method: 'POST' })
}

export function listPlayers(): Promise<Player[]> {
  return request<Player[]>('/api/players')
}

// WebSocket URL for the live console. The API key is a query parameter because
// browsers can't set headers on WebSocket connections; the server's auth
// middleware also accepts ?key=.
export function consoleWebSocketUrl(): string {
  const httpBase = getApiBase() || window.location.origin
  const wsBase = httpBase.replace(/^http/, 'ws')
  return `${wsBase}/api/console?key=${encodeURIComponent(getApiKey())}`
}
