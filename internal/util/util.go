package util

import (
    "crypto/rand"
    "encoding/hex"
    "encoding/json"
    "net/http"
    "time"
)

func GenerateID() string {
    bytes := make([]byte, 16)
    _, _ = rand.Read(bytes)
    return hex.EncodeToString(bytes)
}

func NowUnix() int64 {
    return time.Now().Unix()
}

func WriteJSON(w http.ResponseWriter, payload any) {
    w.Header().Set("Content-Type", "application/json")
    if err := json.NewEncoder(w).Encode(payload); err != nil {
        http.Error(w, "failed to encode response", http.StatusInternalServerError)
    }
}
