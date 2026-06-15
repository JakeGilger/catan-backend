package main

import (
    "fmt"
    "log"
    "net/http"
    "strings"

    "catan-backend/internal/auth"
    "catan-backend/internal/games"
    "catan-backend/internal/store"
    "catan-backend/internal/ws"
)

func main() {
    store.Init()
    hub := ws.NewHub()
    ws.SetDefaultHub(hub)

    mux := http.NewServeMux()
    mux.HandleFunc("/api/login", auth.LoginHandler)
    mux.HandleFunc("/api/games/join", auth.AuthMiddleware(games.JoinGameHandler))
    mux.HandleFunc("/api/games", auth.AuthMiddleware(games.GamesHandler))
    mux.HandleFunc("/api/games/", auth.AuthMiddleware(games.GamesHandler))

    // WebSocket endpoint for game updates: /ws/games/{gameId}
    mux.HandleFunc("/ws/games/", func(w http.ResponseWriter, r *http.Request) {
        gameID := strings.TrimPrefix(r.URL.Path, "/ws/games/")
        // optional auth check via Bearer token
        authHeader := r.Header.Get("Authorization")
        if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
            http.Error(w, "missing or invalid authorization header", http.StatusUnauthorized)
            return
        }
        token := strings.TrimPrefix(authHeader, "Bearer ")
        if _, ok := store.GetUserByToken(token); !ok {
            http.Error(w, "invalid token", http.StatusUnauthorized)
            return
        }
        hub.ServeWS(w, r, gameID)
    })

    fmt.Println("Catan backend listening on http://localhost:4000")
    log.Fatal(http.ListenAndServe(":4000", mux))
}
