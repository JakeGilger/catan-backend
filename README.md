# Catan Backend

A simple Go backend for playing Catan over the internet.

## API Overview

- `POST /api/login` — login and receive a session token
- `GET /api/profile` — fetch current user's profile (auth required)
- `PUT /api/profile` — update current user's profile (auth required)
- `GET /api/users` — list all public user profiles
- `GET /api/users/{id}` — fetch a user's public profile (respects privacy setting)
- `POST /api/games/join` — join an existing game or create a new one
- `GET /api/games/:gameId` — fetch game state
- `POST /api/games/:gameId/save` — save/update game state
- `POST /api/games/:gameId/actions` — perform game actions that modify state

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


## Profile and User Management

### Get Current User Profile
```
GET /api/profile
Authorization: Bearer <token>

Response:
{
  "user": {
    "id": "user-id",
    "username": "username",
    "displayName": "Display Name",
    "avatarUrl": "https://example.com/avatar.jpg",
    "bio": "User bio",
    "publicProfile": true,
    "preferences": { "theme": "dark" }
  }
}
```

### Update Current User Profile
```
PUT /api/profile
Authorization: Bearer <token>

Request:
{
  "displayName": "New Name",
  "avatarUrl": "https://example.com/new-avatar.jpg",
  "bio": "Updated bio",
  "publicProfile": false,
  "preferences": { "theme": "light", "lang": "en" }
}

Response: (same as GET /api/profile)
```

### List Public User Profiles
```
GET /api/users

Response:
{
  "users": [
    {
      "id": "user-id-1",
      "displayName": "Public User 1",
      "avatarUrl": "https://example.com/avatar.jpg",
      "bio": "User bio"
    }
  ]
}
```

Note: Only users with `publicProfile: true` are included in the list.

### Get Specific User Profile
```
GET /api/users/{userId}

Response (if publicProfile: true):
{
  "user": {
    "id": "user-id",
    "displayName": "Display Name",
    "avatarUrl": "https://example.com/avatar.jpg",
    "bio": "User bio"
  }
}

Response (if publicProfile: false):
{} (empty object)
```

## Game Management

### Join or Create a Game
```
POST /api/games/join
Authorization: Bearer <token>

Request:
{
  "gameId": "optional-game-id-to-join",
  "playerName": "Your Player Name"
}

Response:
{
  "game": {
    "id": "game-id",
    "name": "Catan Game abc12345",
    "hostId": "user-id",
    "state": {
      "board": {
        "hexes": [...],
        "settlements": [...],
        "roads": [...]
      },
      "players": [
        {
          "id": "player-id",
          "username": "player-name",
          "displayName": "Display Name",
          "resources": { "brick": 0, "lumber": 0, "ore": 0, "grain": 0, "wool": 0 }
        }
      ],
      "currentPlayerIndex": 0,
      "turnNumber": 1,
      "logs": ["Player joined the game"],
      "updatedAt": 1718459200
    }
  }
}
```

If `gameId` is omitted, a new game is created. If `gameId` is provided, the player joins that game.

### List All Games
```
GET /api/games
Authorization: Bearer <token>

Response:
{
  "games": [
    {
      "id": "game-id-1",
      "name": "Catan Game abc12345",
      "hostId": "user-id",
      "state": { ... }
    }
  ]
}
```

### Get Game State
```
GET /api/games/{gameId}
Authorization: Bearer <token>

Response:
{
  "game": {
    "id": "game-id",
    "name": "Catan Game abc12345",
    "hostId": "user-id",
    "state": { ... }
  }
}
```

### Save Game State
```
POST /api/games/{gameId}/save
Authorization: Bearer <token>

Request:
{
  "state": {
    "board": { ... },
    "players": [ ... ],
    "currentPlayerIndex": 0,
    "turnNumber": 1,
    "logs": [ ... ]
  }
}

Response:
{
  "game": { ... }
}
```

This endpoint updates the game state directly. The server broadcasts the new state to all WebSocket subscribers for the game.

## Game Actions

Game actions modify the game state. All actions are performed via:

```
POST /api/games/{gameId}/actions
Authorization: Bearer <token>

Request:
{
  "actionType": "<action>",
  "payload": { ... }
}

Response:
{
  "game": { ... }
}
```

### Build Settlement
```
Request payload:
{
  "actionType": "buildSettlement",
  "payload": {
    "location": { "q": 0, "r": 1 }
  }
}
```

**Resource cost:** 1 brick, 1 lumber, 1 wool, 1 grain
**Validation:**
- Sufficient resources
- Valid hex location
- Minimum distance from other settlements (3+ hexes away)

### Build Road
```
Request payload:
{
  "actionType": "buildRoad",
  "payload": {
    "start": { "q": 0, "r": 1 },
    "end": { "q": 1, "r": 0 }
  }
}
```

**Resource cost:** 1 brick, 1 lumber
**Validation:**
- Sufficient resources
- Adjacent vertices
- Road not already placed

### Build City (Upgrade Settlement)
```
Request payload:
{
  "actionType": "buildCity",
  "payload": {
    "location": { "q": 0, "r": 1 }
  }
}
```

**Resource cost:** 2 grain, 3 ore
**Validation:**
- Sufficient resources
- Settlement exists at location
- Settlement owned by current player

### End Turn
```
Request payload:
{
  "actionType": "endTurn"
}
```

**Validation:**
- Current player only

**Effect:**
- Advances to next player
- Increments turn number

All successful actions are broadcast to WebSocket subscribers and logged in the game state.

## Run locally

1. Start the server:

```bash
go run main.go
```

The server listens on `http://localhost:4000`.

## API notes

- Authentication uses a bearer token returned by `/api/login`.
- Game state is stored in memory in this demo.
