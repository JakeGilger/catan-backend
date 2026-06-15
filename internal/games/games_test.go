package games

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"

    "catan-backend/internal/model"
    "catan-backend/internal/rules"
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
