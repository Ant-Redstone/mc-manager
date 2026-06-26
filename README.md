# MC Manager
MC Manager is a tool for managing Minecraft servers, built with my homelab in mind. It allows you to easily create, manage, and monitor minecraft servers from a web interface or command line.

The web interface — a React + TypeScript + Vite app — lives in [`web/`](web/): server status, start/stop, a live console, and the player list. See [`web/README.md`](web/README.md) to run it.

## Tech Requirements / Stack
- Go 1.23+
- Docker

## Installation
1. Clone the repository:
   ```bash
   git clone https://github.com/lomokwa/mc-manager.git
   cd mc-manager
   ```

2. Start the application using Docker Compose:
   ```bash
   docker compose up --build
   ```

3. Access the API endpoints through `http://localhost:8080`.

## Configuration

Configuration comes from environment variables, loaded from a `.env` file if one is present (see [`.env.example`](.env.example)).

- **`API_KEY`** (required) — shared secret that must accompany every `/api/*` request. If it is empty or unset the server still starts, but all `/api/*` requests are rejected with `401`.
- **`CORS_ALLOWED_ORIGINS`** (optional) — comma-separated list of browser origins allowed by CORS. Defaults to `http://localhost:5173,http://localhost:8080` (the Vite dev server and the API host).
- **`PORT`** (optional) — port the HTTP server listens on. Defaults to `8080`.

### Authenticating requests

Send the key on every `/api/*` request, either as a header or a query parameter:

```bash
curl -H "X-API-Key: $API_KEY" http://localhost:8080/api/status
# or
curl "http://localhost:8080/api/status?key=$API_KEY"
```

The Swagger UI is public and does not require the key.

# Current Tasks
- [x] Implement server start functionality
- [x] Add server stop functionality
- [ ] Add file upload/download capabilities
- [ ] Add server configuration management
- [ ] Add file management features
- [ ] Implement server logs viewing
- [ ] Add server status monitoring
- [ ] Implement server monitoring features
- [ ] Add user authentication and authorization
- [ ] Implement server backup functionality
- [ ] Add a minimal "lobby" server that users are redirected to when the server gets shutdown / restarted
 
# Docs
API documentation is served via Swagger UI at `http://localhost:8080/api/docs/index.html`.

Docs are generated from comment annotations on handlers using [swaggo](https://github.com/swaggo/swag). To regenerate after editing annotations:
```bash
swag init
```
