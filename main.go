package main

import (
    "fmt"
    "log"
    "net/http"

    "catan-backend/internal/auth"
    "catan-backend/internal/games"
    "catan-backend/internal/store"
)

func main() {
    store.Init()

    mux := http.NewServeMux()
    mux.HandleFunc("/api/login", auth.LoginHandler)
    mux.HandleFunc("/api/games/join", auth.AuthMiddleware(games.JoinGameHandler))
    mux.HandleFunc("/api/games", auth.AuthMiddleware(games.GamesHandler))
    mux.HandleFunc("/api/games/", auth.AuthMiddleware(games.GamesHandler))

    fmt.Println("Catan backend listening on http://localhost:4000")
    log.Fatal(http.ListenAndServe(":4000", mux))
}
