# Catan Backend

A simple Go backend for playing Catan over the internet.

## Features

- `POST /api/login` — login and receive a session token
- `POST /api/games/join` — join an existing game or create a new one
- `GET /api/games/:gameId` — fetch game state
- `POST /api/games/:gameId/save` — save/update game state
- `POST /api/games/:gameId/actions` — perform game actions that modify state

Supported action types:
- `buildSettlement` — payload: `{ "location": { "q": 0, "r": 1 } }`
- `buildRoad` — payload: `{ "start": { "q": 0, "r": 1 }, "end": { "q": 1, "r": 0 } }`
- `buildCity` — payload: `{ "location": { "q": 0, "r": 1 } }`
- `endTurn` — no payload required

Resource cost checks are enforced for built actions.

## Run locally

1. Start the server:

```bash
go run main.go
```

The server listens on `http://localhost:4000`.

## API notes

- Authentication uses a bearer token returned by `/api/login`.
- Game state is stored in memory in this demo.
