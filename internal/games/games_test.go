package games

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"

    "catan-backend/internal/model"
    "catan-backend/internal/rules"
    "catan-backend/internal/store"
)

func TestHandleGameActionBuildCityHappyPath(t *testing.T) {
    user := model.User{ID: "user-1", Username: "Alice", Resources: map[string]int{"grain": 2, "ore": 3}}
    board := rules.NewDefaultBoard()
    board.Settlements = []model.Settlement{{OwnerID: user.ID, Location: model.VertexCoord{Q: 0, R: 0}, IsCity: false}}
    game := &model.Game{
        State: model.GameState{
            Board:              board,
            Players:            []model.User{user},
            CurrentPlayerIndex: 0,
            TurnNumber:         1,
        },
    }

    body := strings.NewReader(`{"actionType":"buildCity","payload":{"location":{"q":0,"r":0}}}`)
    req := httptest.NewRequest(http.MethodPost, "/api/games/abc/actions", body)
    rec := httptest.NewRecorder()

    handleGameAction(rec, req, game, user)
    if rec.Code != http.StatusOK {
        t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
    }

    var response map[string]*model.Game
    if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
        t.Fatalf("failed to decode response: %v", err)
    }

    gotGame, ok := response["game"]
    if !ok {
        t.Fatalf("expected response game field")
    }
    if len(gotGame.State.Board.Settlements) != 1 {
        t.Fatalf("expected one settlement, got %d", len(gotGame.State.Board.Settlements))
    }
    if !gotGame.State.Board.Settlements[0].IsCity {
        t.Fatal("expected settlement to be upgraded to city")
    }
    if gotGame.State.Players[0].Resources["grain"] != 0 || gotGame.State.Players[0].Resources["ore"] != 0 {
        t.Fatal("expected city build resources to be deducted")
    }
}

func TestHandleGameActionBuildCityInsufficientResources(t *testing.T) {
    user := model.User{ID: "user-1", Username: "Alice", Resources: map[string]int{"grain": 1, "ore": 2}}
    board := rules.NewDefaultBoard()
    board.Settlements = []model.Settlement{{OwnerID: user.ID, Location: model.VertexCoord{Q: 0, R: 0}, IsCity: false}}
    game := &model.Game{
        State: model.GameState{
            Board:              board,
            Players:            []model.User{user},
            CurrentPlayerIndex: 0,
            TurnNumber:         1,
        },
    }

    body := strings.NewReader(`{"actionType":"buildCity","payload":{"location":{"q":0,"r":0}}}`)
    req := httptest.NewRequest(http.MethodPost, "/api/games/abc/actions", body)
    rec := httptest.NewRecorder()

    handleGameAction(rec, req, game, user)
    if rec.Code != http.StatusBadRequest {
        t.Fatalf("expected status 400, got %d", rec.Code)
    }
    if !strings.Contains(rec.Body.String(), "insufficient resources") {
        t.Fatalf("expected insufficient resources message, got %s", rec.Body.String())
    }
}

func TestJoinGameCreatesLeaderAndKeepsLobbyState(t *testing.T) {
    store.Reset()
    leader := model.User{ID: "leader-1", Username: "Leader"}
    reqBody := strings.NewReader(`{"playerName":"Leader","gameId":""}`)
    req := httptest.NewRequest(http.MethodPost, "/api/games/join", reqBody)
    rec := httptest.NewRecorder()

    JoinGameHandler(rec, req, leader)

    if rec.Code != http.StatusOK {
        t.Fatalf("expected status 200, got %d", rec.Code)
    }

    var response map[string]*model.Game
    if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
        t.Fatalf("failed to decode response: %v", err)
    }
    got := response["game"]
    if got.LeaderID != leader.ID {
        t.Fatalf("expected leader id %s, got %s", leader.ID, got.LeaderID)
    }
    if got.Started {
        t.Fatal("expected new game to start in lobby mode")
    }
}

func TestLeaderCanStartGame(t *testing.T) {
    leader := model.User{ID: "leader-1", Username: "Leader"}
    game := &model.Game{ID: "game-1", LeaderID: leader.ID, State: model.GameState{Players: []model.User{leader}}}
    req := httptest.NewRequest(http.MethodPost, "/api/games/game-1/start", nil)
    rec := httptest.NewRecorder()

    handleStartGame(rec, req, game, leader)
    if rec.Code != http.StatusOK {
        t.Fatalf("expected status 200, got %d", rec.Code)
    }
    if !game.Started {
        t.Fatal("expected game to be started")
    }
}

func TestNonLeaderCannotStartGame(t *testing.T) {
    leader := model.User{ID: "leader-1", Username: "Leader"}
    other := model.User{ID: "player-2", Username: "Player"}
    game := &model.Game{ID: "game-1", LeaderID: leader.ID, State: model.GameState{Players: []model.User{leader, other}}}
    req := httptest.NewRequest(http.MethodPost, "/api/games/game-1/start", nil)
    rec := httptest.NewRecorder()

    handleStartGame(rec, req, game, other)
    if rec.Code != http.StatusUnauthorized {
        t.Fatalf("expected status 401, got %d", rec.Code)
    }
}

func TestLeaderCanAssignAnotherLeader(t *testing.T) {
    leader := model.User{ID: "leader-1", Username: "Leader"}
    other := model.User{ID: "player-2", Username: "Player"}
    game := &model.Game{ID: "game-1", LeaderID: leader.ID, State: model.GameState{Players: []model.User{leader, other}}}
    body := strings.NewReader(`{"leaderId":"player-2"}`)
    req := httptest.NewRequest(http.MethodPut, "/api/games/game-1/leader", body)
    rec := httptest.NewRecorder()

    handleAssignLeader(rec, req, game, leader)
    if rec.Code != http.StatusOK {
        t.Fatalf("expected status 200, got %d", rec.Code)
    }
    if game.LeaderID != other.ID {
        t.Fatalf("expected leader to be assigned to %s, got %s", other.ID, game.LeaderID)
    }
}

func TestLeaderLeavingFallsBackToNextPlayer(t *testing.T) {
    leader := model.User{ID: "leader-1", Username: "Leader"}
    other := model.User{ID: "player-2", Username: "Player"}
    game := &model.Game{ID: "game-1", LeaderID: leader.ID, State: model.GameState{Players: []model.User{leader, other}}}
    req := httptest.NewRequest(http.MethodPost, "/api/games/game-1/leave", nil)
    rec := httptest.NewRecorder()

    handleLeaveGame(rec, req, game, leader)
    if rec.Code != http.StatusOK {
        t.Fatalf("expected status 200, got %d", rec.Code)
    }
    if game.LeaderID != other.ID {
        t.Fatalf("expected fallback leader %s, got %s", other.ID, game.LeaderID)
    }
    if playerInGame(game, leader.ID) {
        t.Fatal("expected leader to be removed from the game")
    }
}
