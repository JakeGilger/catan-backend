package auth

import (
    "encoding/json"
    "net/http"
    "strings"

    "catan-backend/internal/model"
    "catan-backend/internal/store"
    "catan-backend/internal/util"
)

type LoginRequest struct {
    Username string `json:"username"`
}

type LoginResponse struct {
    Token string     `json:"token"`
    User  model.User `json:"user"`
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }

    var req LoginRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid request body", http.StatusBadRequest)
        return
    }
    if strings.TrimSpace(req.Username) == "" {
        http.Error(w, "username is required", http.StatusBadRequest)
        return
    }

    user := model.User{ID: util.GenerateID(), Username: req.Username}
    token := util.GenerateID()

    store.AddUser(token, user)
    util.WriteJSON(w, LoginResponse{Token: token, User: user})
}

func AuthMiddleware(next func(http.ResponseWriter, *http.Request, model.User)) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        auth := r.Header.Get("Authorization")
        if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
            http.Error(w, "missing or invalid authorization header", http.StatusUnauthorized)
            return
        }

        token := strings.TrimPrefix(auth, "Bearer ")
        user, ok := store.GetUserByToken(token)
        if !ok {
            http.Error(w, "invalid token", http.StatusUnauthorized)
            return
        }

        next(w, r, user)
    }
}
