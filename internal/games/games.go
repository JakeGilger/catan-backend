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
            ID:     util.GenerateID(),
            Name:   fmt.Sprintf("Catan Game %s", util.GenerateID()[:8]),
            HostID: user.ID,
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
        game.State.Players = append(game.State.Players, model.User{ID: user.ID, Username: req.PlayerName, Resources: map[string]int{}})
        game.State.Logs = append(game.State.Logs, fmt.Sprintf("%s joined the game", req.PlayerName))
        game.State.UpdatedAt = util.NowUnix()
    }

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
    util.WriteJSON(w, map[string]*model.Game{"game": game})
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
