# maxx-next

Multi-provider AI proxy with a built-in admin UI, routing, and usage tracking.

## Features
- Proxy endpoints for Claude, OpenAI, Gemini, and Codex formats
- Admin API and Web UI
- Provider routing, retries, and quotas
- SQLite-backed storage

## Docker Compose (recommended)
The service stores its data under `/data` in the container. The compose file
already mounts a named volume so the SQLite DB is persisted.

```
docker compose up -d
```

Full example:
```
services:
  maxx:
    image: ghcr.io/bowl42/maxx:latest
    container_name: maxx-next
    restart: unless-stopped
    ports:
      - "9880:9880"
    volumes:
      - maxx-data:/data
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:9880/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s

volumes:
  maxx-data:
    driver: local
```

## Local Development
Backend:
```
go run cmd/maxx/main.go
```

Frontend:
```
cd web
npm install
npm run dev
```

## Endpoints
- Admin API: http://localhost:9880/admin/
- Web UI: http://localhost:9880/
- WebSocket: ws://localhost:9880/ws
- Claude: http://localhost:9880/v1/messages
- OpenAI: http://localhost:9880/v1/chat/completions
- Codex: http://localhost:9880/v1/responses
- Gemini: http://localhost:9880/v1beta/models/{model}:generateContent

## Data
Default database path (non-Docker): `~/.config/maxx/maxx.db`  
Docker data directory: `/data` (mounted via `docker-compose.yml`)
