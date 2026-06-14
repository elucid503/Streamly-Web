# Streamly

A full-stack web application for streaming movies, TV shows, and live channels with user authentication, watch history, and dynamic subtitle resolution.

## Tech Stack

- **Backend**: Go + Gin + MongoDB
- **Frontend**: React + TypeScript + Vite + Tailwind CSS
- **Media**: HLS streaming, ML-powered subtitle/intro detection, multi-source proxying

## Getting Started

### Backend

```bash
cd backend
go run ./cmd/server
```

Requires: `MONGO_URI`, `JWT_SECRET`, `FRONTEND_ORIGIN` in `.env`

### Frontend

```bash
cd frontend
npm install
npm run dev
```

## Project Structure

- `backend/` — Go API server with media catalog, streaming, and user management
- `frontend/` — React SPA with video player and content browsing
- `media/` — Shared media processing library
