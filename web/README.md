# Open OSCAR Console

Operator UI for the Open OSCAR Server management API.

## Stack

- React 19 + TypeScript + Vite
- nginx reverse-proxy (`/api` → management API `:8080`)
- Professional dark operator UI with selective neon glow accents

## Features

| Area | Management API coverage |
|------|-------------------------|
| Dashboard | version, live API probe, account/session/room/key stats |
| Users | create/delete/password, account patch, linked accounts, feedbag |
| Sessions | list, kick |
| Chat rooms | public CRUD, private list |
| Directory | categories + keywords |
| Web API keys | create/list/toggle/delete |
| Instant message | admin relay |

## Local development

Requires the management API on `http://127.0.0.1:8080` (default from `docker compose` / server).

```bash
cd web
npm ci
npm run dev
```

Vite serves the UI on `http://127.0.0.1:3000` and proxies `/api/*` to the management API.

## Production (Docker)

From the repository root:

```bash
docker compose up -d --build web
# UI: http://localhost:3000
```

The `web` service builds a static SPA and serves it with nginx. API traffic is proxied to the `open-oscar-server` service on the compose network.

## Scripts

| Command | Description |
|---------|-------------|
| `npm run dev` | Vite dev server + API proxy |
| `npm run build` | Typecheck + production bundle |
| `npm run preview` | Preview production build |
