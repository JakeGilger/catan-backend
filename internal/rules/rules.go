package rules

import (
    "encoding/json"
    "errors"
    "fmt"
    "strings"

    "catan-backend/internal/model"
)

type BuildSettlementPayload struct {
    Location model.VertexCoord `json:"location"`
}

type BuildRoadPayload struct {
    Start model.VertexCoord `json:"start"`
    End   model.VertexCoord `json:"end"`
}

type BuildCityPayload struct {
    Location model.VertexCoord `json:"location"`
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

type ActionValidationError struct {
    Message string `json:"message"`
}

func (e ActionValidationError) Error() string {
    return e.Message
}

func NewDefaultBoard() model.Board {
    var hexes []model.HexTile
    for r := -2; r <= 2; r++ {
        qMin := max(-2, -r-2)
        qMax := min(2, -r+2)
        for q := qMin; q <= qMax; q++ {
            hexes = append(hexes, model.HexTile{
                Coordinate: model.HexCoord{Q: q, R: r},
                Resource:   "",
                Number:     0,
            })
        }
    }
    return model.Board{Hexes: hexes, Settlements: []model.Settlement{}, Roads: []model.Road{}}
}

func CurrentPlayer(game *model.Game) (model.User, error) {
    if len(game.State.Players) == 0 {
        return model.User{}, errors.New("no players in game")
    }
    if game.State.CurrentPlayerIndex < 0 || game.State.CurrentPlayerIndex >= len(game.State.Players) {
        return model.User{}, errors.New("invalid current player")
    }
    return game.State.Players[game.State.CurrentPlayerIndex], nil
}

func ValidateBuildSettlement(game *model.Game, user model.User, rawPayload json.RawMessage) error {
    if err := validateTurn(game, user); err != nil {
        return err
    }

    var payload BuildSettlementPayload
    if err := json.Unmarshal(rawPayload, &payload); err != nil {
        return ActionValidationError{Message: "invalid settlement payload"}
    }
    if !isValidVertex(payload.Location) {
        return ActionValidationError{Message: "invalid settlement location"}
    }
    if settlementAt(game, payload.Location) {
        return ActionValidationError{Message: "settlement already exists at that location"}
    }
    if adjacentSettlementExists(game, payload.Location) {
        return ActionValidationError{Message: "cannot place a settlement adjacent to another settlement"}
    }
    return validateResources(user, settlementCost)
}

func ValidateBuildRoad(game *model.Game, user model.User, rawPayload json.RawMessage) error {
    if err := validateTurn(game, user); err != nil {
        return err
    }

    var payload BuildRoadPayload
    if err := json.Unmarshal(rawPayload, &payload); err != nil {
        return ActionValidationError{Message: "invalid road payload"}
    }
    if !isValidRoadSegment(payload.Start, payload.End) {
        return ActionValidationError{Message: "road must connect two adjacent vertices"}
    }
    if roadExists(game, payload.Start, payload.End) {
        return ActionValidationError{Message: "road already exists on that edge"}
    }
    if !roadConnectsToPlayerNetwork(game, user, payload.Start, payload.End) {
        return ActionValidationError{Message: "road must connect to your settlement or road network"}
    }
    return validateResources(user, roadCost)
}

func ValidateBuildCity(game *model.Game, user model.User, rawPayload json.RawMessage) error {
    if err := validateTurn(game, user); err != nil {
        return err
    }

    var payload BuildCityPayload
    if err := json.Unmarshal(rawPayload, &payload); err != nil {
        return ActionValidationError{Message: "invalid city payload"}
    }
    if !isValidVertex(payload.Location) {
        return ActionValidationError{Message: "invalid city location"}
    }

    settlement, ok := settlementForLocation(game, payload.Location)
    if !ok || settlement.OwnerID != user.ID {
        return ActionValidationError{Message: "you must own an existing settlement at that location"}
    }
    if settlement.IsCity {
        return ActionValidationError{Message: "there is already a city at that location"}
    }
    return validateResources(user, cityCost)
}

func ValidateEndTurn(game *model.Game, user model.User) error {
    return validateTurn(game, user)
}

func validateTurn(game *model.Game, user model.User) error {
    if len(game.State.Players) == 0 {
        return ActionValidationError{Message: "no players in game"}
    }
    if game.State.CurrentPlayerIndex < 0 || game.State.CurrentPlayerIndex >= len(game.State.Players) {
        return ActionValidationError{Message: "invalid current player index"}
    }
    current := game.State.Players[game.State.CurrentPlayerIndex]
    if current.ID != user.ID {
        return ActionValidationError{Message: "it is not your turn"}
    }
    return nil
}

func settlementAt(game *model.Game, location model.VertexCoord) bool {
    for _, settlement := range game.State.Board.Settlements {
        if settlement.Location == location {
            return true
        }
    }
    return false
}

func playerSettlementAt(game *model.Game, user model.User, location model.VertexCoord) bool {
    for _, settlement := range game.State.Board.Settlements {
        if settlement.Location == location && settlement.OwnerID == user.ID {
            return true
        }
    }
    return false
}

func adjacentSettlementExists(game *model.Game, location model.VertexCoord) bool {
    for _, neighbor := range vertexNeighbors(location) {
        if settlementAt(game, neighbor) {
            return true
        }
    }
    return false
}

func isValidVertex(location model.VertexCoord) bool {
    return abs(location.Q) <= 5 && abs(location.R) <= 5
}

func vertexNeighbors(location model.VertexCoord) []model.VertexCoord {
    return []model.VertexCoord{
        {Q: location.Q + 1, R: location.R},
        {Q: location.Q - 1, R: location.R},
        {Q: location.Q, R: location.R + 1},
        {Q: location.Q, R: location.R - 1},
        {Q: location.Q + 1, R: location.R - 1},
        {Q: location.Q - 1, R: location.R + 1},
    }
}

func isValidRoadSegment(start, end model.VertexCoord) bool {
    for _, neighbor := range vertexNeighbors(start) {
        if neighbor == end {
            return true
        }
    }
    return false
}

func roadExists(game *model.Game, start, end model.VertexCoord) bool {
    for _, road := range game.State.Board.Roads {
        if (road.Start == start && road.End == end) || (road.Start == end && road.End == start) {
            return true
        }
    }
    return false
}

func roadConnectsToPlayerNetwork(game *model.Game, user model.User, start, end model.VertexCoord) bool {
    if playerSettlementAt(game, user, start) || playerSettlementAt(game, user, end) {
        return true
    }
    for _, road := range game.State.Board.Roads {
        if road.OwnerID != user.ID {
            continue
        }
        if road.Start == start || road.End == start || road.Start == end || road.End == end {
            return true
        }
    }
    return false
}

func validateResources(user model.User, cost map[string]int) error {
    if hasResources(user, cost) {
        return nil
    }
    return ActionValidationError{Message: fmt.Sprintf("insufficient resources: %s", resourceShortfall(user, cost))}
}

func hasResources(user model.User, cost map[string]int) bool {
    for resource, amount := range cost {
        if user.Resources[resource] < amount {
            return false
        }
    }
    return true
}

func resourceShortfall(user model.User, cost map[string]int) string {
    var shortfalls []string
    for resource, amount := range cost {
        missing := amount - user.Resources[resource]
        if missing > 0 {
            shortfalls = append(shortfalls, fmt.Sprintf("%s: %d", resource, missing))
        }
    }
    return strings.Join(shortfalls, ", ")
}

func settlementForLocation(game *model.Game, location model.VertexCoord) (*model.Settlement, bool) {
    for i := range game.State.Board.Settlements {
        if game.State.Board.Settlements[i].Location == location {
            return &game.State.Board.Settlements[i], true
        }
    }
    return nil, false
}

func abs(value int) int {
    if value < 0 {
        return -value
    }
    return value
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}

func max(a, b int) int {
    if a > b {
        return a
    }
    return b
}
