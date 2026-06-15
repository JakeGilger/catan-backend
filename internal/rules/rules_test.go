package rules

import (
    "encoding/json"
    "testing"

    "catan-backend/internal/model"
)

func TestValidateBuildSettlementSuccess(t *testing.T) {
    user := model.User{ID: "user-1", Username: "Alice", Resources: map[string]int{"brick": 1, "lumber": 1, "wool": 1, "grain": 1}}
    game := &model.Game{
        State: model.GameState{
            Board:              NewDefaultBoard(),
            Players:            []model.User{user},
            CurrentPlayerIndex: 0,
            TurnNumber:         1,
        },
    }

    payload := json.RawMessage(`{"location":{"q":0,"r":0}}`)
    err := ValidateBuildSettlement(game, user, payload)
    if err != nil {
        t.Fatalf("expected settlement validation to succeed, got %v", err)
    }
}

func TestValidateBuildSettlementAdjacentError(t *testing.T) {
    user := model.User{ID: "user-1", Username: "Alice", Resources: map[string]int{"brick": 1, "lumber": 1, "wool": 1, "grain": 1}}
    board := NewDefaultBoard()
    board.Settlements = []model.Settlement{{OwnerID: "user-1", Location: model.VertexCoord{Q: 1, R: 0}, IsCity: false}}
    game := &model.Game{
        State: model.GameState{
            Board:              board,
            Players:            []model.User{user},
            CurrentPlayerIndex: 0,
            TurnNumber:         1,
        },
    }

    payload := json.RawMessage(`{"location":{"q":0,"r":0}}`)
    err := ValidateBuildSettlement(game, user, payload)
    if err == nil {
        t.Fatal("expected settlement validation to fail when adjacent settlement exists")
    }
}

func TestValidateBuildCitySuccess(t *testing.T) {
    user := model.User{ID: "user-1", Username: "Alice", Resources: map[string]int{"grain": 2, "ore": 3}}
    board := NewDefaultBoard()
    board.Settlements = []model.Settlement{{OwnerID: "user-1", Location: model.VertexCoord{Q: 0, R: 0}, IsCity: false}}
    game := &model.Game{
        State: model.GameState{
            Board:              board,
            Players:            []model.User{user},
            CurrentPlayerIndex: 0,
            TurnNumber:         1,
        },
    }

    payload := json.RawMessage(`{"location":{"q":0,"r":0}}`)
    err := ValidateBuildCity(game, user, payload)
    if err != nil {
        t.Fatalf("expected city validation to succeed, got %v", err)
    }
}

func TestValidateBuildCityResourceError(t *testing.T) {
    user := model.User{ID: "user-1", Username: "Alice", Resources: map[string]int{"grain": 1, "ore": 2}}
    board := NewDefaultBoard()
    board.Settlements = []model.Settlement{{OwnerID: "user-1", Location: model.VertexCoord{Q: 0, R: 0}, IsCity: false}}
    game := &model.Game{
        State: model.GameState{
            Board:              board,
            Players:            []model.User{user},
            CurrentPlayerIndex: 0,
            TurnNumber:         1,
        },
    }

    payload := json.RawMessage(`{"location":{"q":0,"r":0}}`)
    err := ValidateBuildCity(game, user, payload)
    if err == nil {
        t.Fatal("expected city validation to fail when resources are insufficient")
    }
}

func TestValidateBuildCityOwnershipError(t *testing.T) {
    user := model.User{ID: "user-2", Username: "Bob", Resources: map[string]int{"grain": 2, "ore": 3}}
    board := NewDefaultBoard()
    board.Settlements = []model.Settlement{{OwnerID: "user-1", Location: model.VertexCoord{Q: 0, R: 0}, IsCity: false}}
    game := &model.Game{
        State: model.GameState{
            Board: board,
            Players: []model.User{model.User{ID: "user-1", Username: "Alice", Resources: map[string]int{"grain": 2, "ore": 3}}, user},
            CurrentPlayerIndex: 1,
            TurnNumber:         1,
        },
    }

    payload := json.RawMessage(`{"location":{"q":0,"r":0}}`)
    err := ValidateBuildCity(game, user, payload)
    if err == nil {
        t.Fatal("expected city validation to fail when player does not own settlement")
    }
}
