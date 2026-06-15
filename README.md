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

WebSocket endpoint:
- `GET /ws/games/{gameId}` — open a WebSocket connection (use `Authorization: Bearer <token>` header).
	- The server will push JSON messages of the form `{ "game": { ... } }` whenever the game state changes.
	- Example client (browser):

```javascript
const ws = new WebSocket('ws://localhost:4000/ws/games/your-game-id', {
	headers: { Authorization: 'Bearer <token>' }
});
ws.onmessage = (ev) => {
	const msg = JSON.parse(ev.data);
	console.log('game update', msg.game);
};
```

## Run locally

1. Start the server:

```bash
go run main.go
```

The server listens on `http://localhost:4000`.

## API notes

- Authentication uses a bearer token returned by `/api/login`.
- Game state is stored in memory in this demo.
