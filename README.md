# Streamly

A full-stack web application for streaming movies, TV shows, and live channels with user authentication, watch history, and dynamic subtitle resolution.

## Tech Stack

- **Backend**: Go + Gin + MongoDB
- **Frontend**: React + TypeScript + Vite + Tailwind CSS
- **Media**: HLS streaming, Timed subtitles, intro skipping, multi-source proxying

## Getting Started

### Backend

```bash
cd Backend
go run ./cmd/server
```

Requires: `MONGO_URI`, `JWT_SECRET`, `FRONTEND_ORIGIN` in `.env`

### Frontend

```bash
cd Frontend
npm install
npm run dev
```

## Project Structure

- `Frontend/src/Features/` — feature-owned pages, components, API clients, types, and stores. Player code is grouped into playback, subtitles, and ad detection.
- `Frontend/src/Core/`, `UI/`, `Layout/`, and `Utils/` — shared infrastructure, controls, navigation layout, and general helpers.
- `Backend/internal/features/` — auth, settings, library, admin, catalog, playback, and sports. Each feature owns its handlers and services; catalog caches and subtitle providers live with their feature.
- `Backend/internal/httpx/`, `middleware/`, `database/`, `models/`, `config/`, and `upstream/` — shared server infrastructure.
- `media/internal/` — catalog, VOD, live TV, playback quality, and external provider adapters. `client/` composes them; the root package exposes the media API.

Keep code with its owning feature. Share helpers only when multiple features need them, and import owners directly rather than adding global API, store, or type barrels. Follow `CLAUDE.md` for formatting.

## Validation

```bash
cd Frontend
npm test
npm run build

cd ../Backend
go test ./...

cd ../media
go test -short ./...
```

The media short suite skips tests that contact live providers. Run `go test ./...` in `media/` when those services and network access are available.
