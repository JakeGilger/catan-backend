package games

import (
    "encoding/json"
    "fmt"
    "net/http"
    "strings"

    "catan-backend/internal/model"
    "catan-backend/internal/rules"
    "catan-backend/internal/store"
    "catan-backend/internal/util"
    "catan-backend/internal/ws"
)

type JoinGameRequest struct {
    GameID     string `json:"gameId,omitempty"`
    PlayerName string `json:"playerName"`
}

type SaveGameRequest struct {
    State model.GameState `json:"state"`
}

type ActionRequest struct {
    ActionType string          `json:"actionType"`
    Payload    json.RawMessage `json:"payload,omitempty"`
}

type AssignLeaderRequest struct {
    LeaderID string `json:"leaderId,omitempty"`
}

func JoinGameHandler(w http.ResponseWriter, r *http.Request, user model.User) {
    if r.Method != http.MethodPost {
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }

    var req JoinGameRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid request body", http.StatusBadRequest)
        return
    }
    if strings.TrimSpace(req.PlayerName) == "" {
        http.Error(w, "playerName is required", http.StatusBadRequest)
        return
    }

    game, ok := store.GetGame(req.GameID)
    if !ok {
        game = &model.Game{
            ID:       util.GenerateID(),
            Name:     fmt.Sprintf("Catan Game %s", util.GenerateID()[:8]),
            HostID:   user.ID,
            LeaderID: user.ID,
            Started:  false,
            State: model.GameState{
                Board:              rules.NewDefaultBoard(),
                Players:            []model.User{},
                CurrentPlayerIndex: 0,
                TurnNumber:         1,
                Logs:               []string{},
                UpdatedAt:          util.NowUnix(),
            },
        }
    }

    if !playerInGame(game, user.ID) {
        if game.Started {
            http.Error(w, "cannot join a game that has already started", http.StatusBadRequest)
            return
        }
        game.State.Players = append(game.State.Players, model.User{ID: user.ID, Username: req.PlayerName, Resources: map[string]int{}})
        game.State.Logs = append(game.State.Logs, fmt.Sprintf("%s joined the game", req.PlayerName))
        game.State.UpdatedAt = util.NowUnix()
    }

    ensureLeader(game)
    store.SaveGame(game)
    util.WriteJSON(w, map[string]*model.Game{"game": game})
}

func GamesHandler(w http.ResponseWriter, r *http.Request, user model.User) {
    path := strings.TrimPrefix(r.URL.Path, "/api/games")
    if path == "" || path == "/" {
        if r.Method == http.MethodGet {
            handleListGames(w, r)
            return
        }
    }

    parts := strings.Split(strings.Trim(path, "/"), "/")
    if len(parts) == 0 || parts[0] == "" {
        http.NotFound(w, r)
        return
    }

    gameID := parts[0]
    game, ok := store.GetGame(gameID)
    if !ok {
        http.Error(w, "game not found", http.StatusNotFound)
        return
    }

    if len(parts) == 1 && r.Method == http.MethodGet {
        util.WriteJSON(w, map[string]*model.Game{"game": game})
        return
    }
    if len(parts) == 2 {
        switch parts[1] {
        case "save":
            handleSaveGame(w, r, game)
            return
        case "actions":
            handleGameAction(w, r, game, user)
            return
        case "start":
            handleStartGame(w, r, game, user)
            return
        case "leader":
            handleAssignLeader(w, r, game, user)
            return
        case "leave":
            handleLeaveGame(w, r, game, user)
            return
        }
    }

    http.NotFound(w, r)
}

func handleListGames(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }

    games := store.ListGames()
    util.WriteJSON(w, map[string]any{"games": games})
}

func handleSaveGame(w http.ResponseWriter, r *http.Request, game *model.Game) {
    if r.Method != http.MethodPost {
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }

    var req SaveGameRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid request body", http.StatusBadRequest)
        return
    }

    game.State = req.State
    game.State.UpdatedAt = util.NowUnix()
    store.SaveGame(game)
    // broadcast updated game state to WS subscribers
    ws.BroadcastGame(game)
    util.WriteJSON(w, map[string]*model.Game{"game": game})
}

func handleStartGame(w http.ResponseWriter, r *http.Request, game *model.Game, user model.User) {
    if r.Method != http.MethodPost {
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }
    if user.ID != game.LeaderID {
        http.Error(w, "only the leader can start the game", http.StatusUnauthorized)
        return
    }
    if game.Started {
        http.Error(w, "game has already started", http.StatusBadRequest)
        return
    }

    game.Started = true
    game.State.Logs = append(game.State.Logs, fmt.Sprintf("%s started the game", user.Username))
    game.State.UpdatedAt = util.NowUnix()
    store.SaveGame(game)
    ws.BroadcastGame(game)
    util.WriteJSON(w, map[string]*model.Game{"game": game})
}

func handleAssignLeader(w http.ResponseWriter, r *http.Request, game *model.Game, user model.User) {
    if r.Method != http.MethodPut {
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }
    if user.ID != game.LeaderID {
        http.Error(w, "only the leader can assign a new leader", http.StatusUnauthorized)
        return
    }

    var req AssignLeaderRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid request body", http.StatusBadRequest)
        return
    }

    if req.LeaderID == "" {
        if len(game.State.Players) <= 1 {
            http.Error(w, "no other player available to assign leadership", http.StatusBadRequest)
            return
        }
        for _, player := range game.State.Players {
            if player.ID != user.ID {
                req.LeaderID = player.ID
                break
            }
        }
    }

    if req.LeaderID == user.ID {
        util.WriteJSON(w, map[string]*model.Game{"game": game})
        return
    }

    if !playerInGame(game, req.LeaderID) {
        http.Error(w, "leader must be an existing player", http.StatusBadRequest)
        return
    }

    game.LeaderID = req.LeaderID
    leaderName := req.LeaderID
    for _, player := range game.State.Players {
        if player.ID == req.LeaderID {
            leaderName = player.Username
            break
        }
    }
    game.State.Logs = append(game.State.Logs, fmt.Sprintf("%s is now the leader", leaderName))
    game.State.UpdatedAt = util.NowUnix()
    store.SaveGame(game)
    ws.BroadcastGame(game)
    util.WriteJSON(w, map[string]*model.Game{"game": game})
}

func handleLeaveGame(w http.ResponseWriter, r *http.Request, game *model.Game, user model.User) {
    if r.Method != http.MethodPost {
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }
    if !playerInGame(game, user.ID) {
        http.Error(w, "player not in game", http.StatusBadRequest)
        return
    }
    idx := playerIndex(game, user.ID)
    game.State.Players = append(game.State.Players[:idx], game.State.Players[idx+1:]...)
    game.State.Logs = append(game.State.Logs, fmt.Sprintf("%s left the game", user.Username))
    if user.ID == game.LeaderID {
        if len(game.State.Players) > 0 {
            game.LeaderID = game.State.Players[0].ID
            game.State.Logs = append(game.State.Logs, fmt.Sprintf("%s is now the leader", game.State.Players[0].Username))
        } else {
            game.LeaderID = ""
        }
    }
    game.State.UpdatedAt = util.NowUnix()
    store.SaveGame(game)
    ws.BroadcastGame(game)
    util.WriteJSON(w, map[string]*model.Game{"game": game})
}

func handleGameAction(w http.ResponseWriter, r *http.Request, game *model.Game, user model.User) {
    if r.Method != http.MethodPost {
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }

    var req ActionRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid request body", http.StatusBadRequest)
        return
    }
    if strings.TrimSpace(req.ActionType) == "" {
        http.Error(w, "actionType is required", http.StatusBadRequest)
        return
    }

    var err error
    switch req.ActionType {
    case "buildSettlement":
        err = rules.ValidateBuildSettlement(game, user, req.Payload)
        if err == nil {
            var payload rules.BuildSettlementPayload
            _ = json.Unmarshal(req.Payload, &payload)
            game.State.Board.Settlements = append(game.State.Board.Settlements, model.Settlement{
                OwnerID:  user.ID,
                Location: payload.Location,
                IsCity:   false,
            })
            deductResources(game, user, settlementCost)
            game.State.Logs = append(game.State.Logs, fmt.Sprintf("%s built a settlement at %v", user.Username, payload.Location))
        }
    case "buildRoad":
        err = rules.ValidateBuildRoad(game, user, req.Payload)
        if err == nil {
            var payload rules.BuildRoadPayload
            _ = json.Unmarshal(req.Payload, &payload)
            game.State.Board.Roads = append(game.State.Board.Roads, model.Road{
                OwnerID: user.ID,
                Start:   payload.Start,
                End:     payload.End,
            })
            deductResources(game, user, roadCost)
            game.State.Logs = append(game.State.Logs, fmt.Sprintf("%s built a road between %v and %v", user.Username, payload.Start, payload.End))
        }
    case "buildCity":
        err = rules.ValidateBuildCity(game, user, req.Payload)
        if err == nil {
            var payload rules.BuildCityPayload
            _ = json.Unmarshal(req.Payload, &payload)
            upgradeSettlement(game, user, payload.Location)
            deductResources(game, user, cityCost)
            game.State.Logs = append(game.State.Logs, fmt.Sprintf("%s upgraded settlement at %v into a city", user.Username, payload.Location))
        }
    case "endTurn":
        err = rules.ValidateEndTurn(game, user)
        if err == nil {
            game.State.CurrentPlayerIndex = (game.State.CurrentPlayerIndex + 1) % len(game.State.Players)
            game.State.TurnNumber++
            game.State.Logs = append(game.State.Logs, fmt.Sprintf("%s ended their turn", user.Username))
        }
    default:
        http.Error(w, "unknown action type", http.StatusBadRequest)
        return
    }

    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    game.State.UpdatedAt = util.NowUnix()
    store.SaveGame(game)
    // broadcast update for all connected websocket clients
    ws.BroadcastGame(game)
    util.WriteJSON(w, map[string]*model.Game{"game": game})
}

func ensureLeader(game *model.Game) {
    if game.LeaderID != "" && playerInGame(game, game.LeaderID) {
        return
    }
    if len(game.State.Players) > 0 {
        game.LeaderID = game.State.Players[0].ID
        game.State.Logs = append(game.State.Logs, fmt.Sprintf("%s is now the leader", game.State.Players[0].Username))
    }
}

var settlementCost = map[string]int{
    "brick": 1,
    "lumber": 1,
    "wool":   1,
    "grain":  1,
}

var roadCost = map[string]int{
    "brick": 1,
    "lumber": 1,
}

var cityCost = map[string]int{
    "grain": 2,
    "ore":   3,
}

func playerInGame(game *model.Game, userID string) bool {
    for _, player := range game.State.Players {
        if player.ID == userID {
            return true
        }
    }
    return false
}

func playerIndex(game *model.Game, userID string) int {
    for i, player := range game.State.Players {
        if player.ID == userID {
            return i
        }
    }
    return -1
}

func deductResources(game *model.Game, user model.User, cost map[string]int) {
    idx := playerIndex(game, user.ID)
    if idx < 0 {
        return
    }
    for key, amount := range cost {
        game.State.Players[idx].Resources[key] -= amount
        if game.State.Players[idx].Resources[key] < 0 {
            game.State.Players[idx].Resources[key] = 0
        }
    }
}

func upgradeSettlement(game *model.Game, user model.User, location model.VertexCoord) {
    for i := range game.State.Board.Settlements {
        if game.State.Board.Settlements[i].OwnerID == user.ID && game.State.Board.Settlements[i].Location == location {
            game.State.Board.Settlements[i].IsCity = true
            return
        }
    }
}
