# MC Manager — Web UI

A dark-themed **React + TypeScript + Vite** frontend for the mc-manager API, styled with **Tailwind CSS** and animated with **Framer Motion**.

Features:

- **Connect screen** — enter your `API_KEY` (and optionally the API URL). Stored in `localStorage`; reset with **Change key**.
- **Server** — live status (running/stopped) with **Start** / **Stop** and 5s polling. Shows PID/uptime when the API exposes them.
- **Console** — live output over the `/api/console` WebSocket with a command input.
- **Players** — list with online / op / whitelist / banned status.

## Develop

```bash
cd web
npm install
npm run dev
```

The dev server runs on <http://localhost:5173> and proxies `/api` (REST **and** the `/api/console` WebSocket) to `http://localhost:8080`, so the app uses same-origin requests and needs no CORS in dev. Point it elsewhere with:

```bash
VITE_API_PROXY=http://my-host:8080 npm run dev
```

On first load, enter your `API_KEY`; leave the API URL blank to use the same origin (what the dev proxy expects). The WebSocket sends the key as `?key=`, which the server's auth middleware accepts.

## Build

```bash
npm run build      # type-checks (tsc) then bundles to web/dist
npm run preview    # serve the production build locally
```

The static files in `dist/` can be served by any static host — or by the Go server itself.

## Stack

React 19 · TypeScript · Vite · Tailwind CSS · Framer Motion. No backend code; it talks to the existing HTTP + WebSocket API.
