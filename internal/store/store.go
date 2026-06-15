package store

import (
    "sync"

    "catan-backend/internal/model"
)

var (
    mu           sync.RWMutex
    UsersByToken map[string]model.User
    UsersByID    map[string]model.User
    GamesByID    map[string]*model.Game
)

func Init() {
    mu.Lock()
    defer mu.Unlock()
    if UsersByToken == nil {
        UsersByToken = map[string]model.User{}
    }
    if UsersByID == nil {
        UsersByID = map[string]model.User{}
    }
    if GamesByID == nil {
        GamesByID = map[string]*model.Game{}
    }
}

// Reset clears all store data (for testing)
func Reset() {
    mu.Lock()
    defer mu.Unlock()
    UsersByToken = map[string]model.User{}
    UsersByID = map[string]model.User{}
    GamesByID = map[string]*model.Game{}
}

func ensureInitialized() {
    if UsersByToken == nil {
        UsersByToken = map[string]model.User{}
    }
    if UsersByID == nil {
        UsersByID = map[string]model.User{}
    }
    if GamesByID == nil {
        GamesByID = map[string]*model.Game{}
    }
}

func AddUser(token string, user model.User) {
    mu.Lock()
    defer mu.Unlock()
    ensureInitialized()
    UsersByToken[token] = user
    UsersByID[user.ID] = user
}

func GetUserByID(userID string) (model.User, bool) {
    mu.RLock()
    defer mu.RUnlock()
    if UsersByID == nil {
        return model.User{}, false
    }
    u, ok := UsersByID[userID]
    return u, ok
}

func UpdateUser(user model.User) {
    mu.Lock()
    defer mu.Unlock()
    ensureInitialized()
    UsersByID[user.ID] = user
    // also update any tokens that point to this user
    for t, u := range UsersByToken {
        if u.ID == user.ID {
            UsersByToken[t] = user
        }
    }
}

func GetUserByToken(token string) (model.User, bool) {
    mu.RLock()
    defer mu.RUnlock()
    if UsersByToken == nil {
        return model.User{}, false
    }
    user, ok := UsersByToken[token]
    return user, ok
}

func SaveGame(game *model.Game) {
    mu.Lock()
    defer mu.Unlock()
    ensureInitialized()
    GamesByID[game.ID] = game
}

func GetGame(gameID string) (*model.Game, bool) {
    mu.RLock()
    defer mu.RUnlock()
    if GamesByID == nil {
        return nil, false
    }
    game, ok := GamesByID[gameID]
    return game, ok
}

func ListGames() []*model.Game {
    mu.RLock()
    defer mu.RUnlock()
    if GamesByID == nil {
        return nil
    }
    games := make([]*model.Game, 0, len(GamesByID))
    for _, game := range GamesByID {
        games = append(games, game)
    }
    return games
}

func ListUsers() []model.User {
    mu.RLock()
    defer mu.RUnlock()
    if UsersByID == nil {
        return nil
    }
    users := make([]model.User, 0, len(UsersByID))
    for _, u := range UsersByID {
        users = append(users, u)
    }
    return users
}
