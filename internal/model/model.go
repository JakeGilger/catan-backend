package model

type User struct {
    ID        string            `json:"id"`
    Username  string            `json:"username"`
    DisplayName string          `json:"displayName,omitempty"`
    AvatarURL string            `json:"avatarUrl,omitempty"`
    Bio       string            `json:"bio,omitempty"`
    PublicProfile bool          `json:"publicProfile"`
    Preferences map[string]string `json:"preferences,omitempty"`
    Resources map[string]int    `json:"resources"`
    Stats     PlayerStats       `json:"stats,omitempty"`
}

type PlayerStats struct {
    Wins       int `json:"wins"`
    Losses     int `json:"losses"`
    GamesPlayed int `json:"gamesPlayed"`
}

type HexCoord struct {
    Q int `json:"q"`
    R int `json:"r"`
}

type VertexCoord struct {
    Q int `json:"q"`
    R int `json:"r"`
}

type HexTile struct {
    Coordinate HexCoord `json:"coordinate"`
    Resource   string   `json:"resource"`
    Number     int      `json:"number"`
}

type Settlement struct {
    OwnerID  string      `json:"ownerId"`
    Location VertexCoord `json:"location"`
    IsCity   bool        `json:"isCity"`
}

type Road struct {
    OwnerID string      `json:"ownerId"`
    Start   VertexCoord `json:"start"`
    End     VertexCoord `json:"end"`
}

type Board struct {
    Hexes       []HexTile    `json:"hexes"`
    Settlements []Settlement `json:"settlements"`
    Roads       []Road       `json:"roads"`
}

type GameState struct {
    Board              Board   `json:"board"`
    Players            []User  `json:"players"`
    CurrentPlayerIndex int     `json:"currentPlayerIndex"`
    TurnNumber         int     `json:"turnNumber"`
    Logs               []string `json:"logs"`
    UpdatedAt          int64   `json:"updatedAt"`
}

type Game struct {
    ID     string    `json:"id"`
    Name   string    `json:"name"`
    HostID string    `json:"hostId"`
    State  GameState `json:"state"`
}
