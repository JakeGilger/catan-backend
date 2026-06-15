package auth

import (
    "encoding/json"
    "net/http"
    "net/url"
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

    user := model.User{ID: util.GenerateID(), Username: req.Username, DisplayName: req.Username, PublicProfile: true, Resources: map[string]int{}, Preferences: map[string]string{}}
    token := util.GenerateID()

    store.AddUser(token, user)
    util.WriteJSON(w, LoginResponse{Token: token, User: user})
}

type UpdateProfileRequest struct {
    DisplayName string            `json:"displayName,omitempty"`
    AvatarURL   string            `json:"avatarUrl,omitempty"`
    Bio         string            `json:"bio,omitempty"`
    Preferences map[string]string `json:"preferences,omitempty"`
    PublicProfile *bool          `json:"publicProfile,omitempty"`
}

// isValidImageURL checks if the URL is a reasonable image URL
func isValidImageURL(imageURL string) bool {
    if imageURL == "" {
        return true // empty is okay
    }
    u, err := url.Parse(imageURL)
    if err != nil {
        return false
    }
    // must have http/https scheme
    if u.Scheme != "http" && u.Scheme != "https" {
        return false
    }
    // must have host
    if u.Host == "" {
        return false
    }
    return true
}

func ProfileHandler(w http.ResponseWriter, r *http.Request, user model.User) {
    switch r.Method {
    case http.MethodGet:
        util.WriteJSON(w, map[string]model.User{"user": user})
        return
    case http.MethodPost, http.MethodPut:
        var req UpdateProfileRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            http.Error(w, "invalid request body", http.StatusBadRequest)
            return
        }
        if req.DisplayName != "" {
            user.DisplayName = req.DisplayName
        }
        if req.AvatarURL != "" {
            if !isValidImageURL(req.AvatarURL) {
                http.Error(w, "invalid avatar url", http.StatusBadRequest)
                return
            }
            user.AvatarURL = req.AvatarURL
        }
        if req.Bio != "" {
            user.Bio = req.Bio
        }
        if req.Preferences != nil {
            if user.Preferences == nil {
                user.Preferences = map[string]string{}
            }
            for k, v := range req.Preferences {
                user.Preferences[k] = v
            }
        }
        if req.PublicProfile != nil {
            user.PublicProfile = *req.PublicProfile
        }

        store.UpdateUser(user)
        util.WriteJSON(w, map[string]model.User{"user": user})
        return
    default:
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }
}

// PublicUserHandler serves public user profiles at /api/users/{userId}
func PublicUserHandler(w http.ResponseWriter, r *http.Request) {
    userID := strings.TrimPrefix(r.URL.Path, "/api/users/")
    if userID == "" {
        http.Error(w, "user id required", http.StatusBadRequest)
        return
    }

    user, ok := store.GetUserByID(userID)
    if !ok {
        http.Error(w, "user not found", http.StatusNotFound)
        return
    }

    if !user.PublicProfile {
        // return blank object for privacy
        util.WriteJSON(w, map[string]any{})
        return
    }

    // return public-safe subset
    public := map[string]any{
        "id": user.ID,
        "displayName": user.DisplayName,
        "avatarUrl": user.AvatarURL,
        "bio": user.Bio,
    }
    util.WriteJSON(w, map[string]any{"user": public})
}

// UsersListHandler serves a list of all public users at /api/users
func UsersListHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }

    users := store.ListUsers()
    publicUsers := []map[string]any{}
    for _, u := range users {
        if u.PublicProfile {
            publicUsers = append(publicUsers, map[string]any{
                "id": u.ID,
                "displayName": u.DisplayName,
                "avatarUrl": u.AvatarURL,
                "bio": u.Bio,
            })
        }
    }

    util.WriteJSON(w, map[string]any{"users": publicUsers})
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
