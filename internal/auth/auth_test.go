package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"catan-backend/internal/model"
	"catan-backend/internal/store"
)

func init() {
	store.Init()
}

func TestLoginCreatesPublicProfileByDefault(t *testing.T) {
	store.Reset()
	req, _ := http.NewRequest("POST", "/api/login", bytes.NewBufferString(`{"username":"testuser"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	LoginHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp LoginResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if !resp.User.PublicProfile {
		t.Error("expected new user to have PublicProfile: true")
	}
	if resp.User.DisplayName != "testuser" {
		t.Error("expected DisplayName to equal Username")
	}
	if resp.Token == "" {
		t.Error("expected token to be generated")
	}
}

func TestLoginRequiresUsername(t *testing.T) {
	store.Reset()
	req, _ := http.NewRequest("POST", "/api/login", bytes.NewBufferString(`{"username":""}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	LoginHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestProfileUpdateTogglePrivacy(t *testing.T) {
	store.Reset()
	// Create a user
	user := model.User{
		ID:            "testuser123",
		Username:      "testuser",
		DisplayName:   "Test User",
		PublicProfile: true,
		Resources:     map[string]int{},
		Preferences:   map[string]string{},
	}
	store.AddUser("token123", user)

	// Update to private
	updateReq := UpdateProfileRequest{
		PublicProfile: ptrBool(false),
	}
	body, _ := json.Marshal(updateReq)
	req, _ := http.NewRequest("PUT", "/api/profile", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	ProfileHandler(w, req, user)

	var resp map[string]model.User
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["user"].PublicProfile {
		t.Error("expected PublicProfile to be false after update")
	}
}

func TestProfileUpdateValidatesAvatarURL(t *testing.T) {
	store.Reset()
	user := model.User{
		ID:            "testuser123",
		Username:      "testuser",
		DisplayName:   "Test User",
		PublicProfile: true,
		Resources:     map[string]int{},
		Preferences:   map[string]string{},
	}
	store.AddUser("token123", user)

	// Try to update with invalid URL
	updateReq := UpdateProfileRequest{
		AvatarURL: "not-a-valid-url",
	}
	body, _ := json.Marshal(updateReq)
	req, _ := http.NewRequest("PUT", "/api/profile", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	ProfileHandler(w, req, user)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid URL, got %d", w.Code)
	}
}

func TestProfileUpdateAcceptsValidAvatarURL(t *testing.T) {
	store.Reset()
	user := model.User{
		ID:            "testuser123",
		Username:      "testuser",
		DisplayName:   "Test User",
		PublicProfile: true,
		Resources:     map[string]int{},
		Preferences:   map[string]string{},
	}
	store.AddUser("token123", user)

	// Update with valid URL
	validURL := "https://example.com/avatar.jpg"
	updateReq := UpdateProfileRequest{
		AvatarURL: validURL,
	}
	body, _ := json.Marshal(updateReq)
	req, _ := http.NewRequest("PUT", "/api/profile", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	ProfileHandler(w, req, user)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]model.User
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["user"].AvatarURL != validURL {
		t.Errorf("expected avatar URL to be %s, got %s", validURL, resp["user"].AvatarURL)
	}
}

func TestPublicUserHandlerReturnsEmptyForPrivateProfile(t *testing.T) {
	store.Reset()
	user := model.User{
		ID:            "privateuser",
		Username:      "privateuser",
		DisplayName:   "Private User",
		AvatarURL:     "https://example.com/avatar.jpg",
		Bio:           "Secret bio",
		PublicProfile: false,
		Resources:     map[string]int{},
		Preferences:   map[string]string{},
	}
	store.AddUser("token", user)

	req, _ := http.NewRequest("GET", "/api/users/privateuser", nil)
	w := httptest.NewRecorder()

	PublicUserHandler(w, req)

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)

	// Should be empty object
	if len(resp) > 0 && resp["user"] != nil {
		userData := resp["user"].(map[string]interface{})
		if displayName, ok := userData["displayName"].(string); ok && displayName != "" {
			t.Error("expected empty response for private user, but got data")
		}
	}
}

func TestPublicUserHandlerReturnsDataForPublicProfile(t *testing.T) {
	store.Reset()
	user := model.User{
		ID:            "publicuser",
		Username:      "publicuser",
		DisplayName:   "Public User",
		AvatarURL:     "https://example.com/avatar.jpg",
		Bio:           "Hello world",
		PublicProfile: true,
		Resources:     map[string]int{},
		Preferences:   map[string]string{},
	}
	store.AddUser("token", user)

	req, _ := http.NewRequest("GET", "/api/users/publicuser", nil)
	w := httptest.NewRecorder()

	PublicUserHandler(w, req)

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)

	userData := resp["user"].(map[string]interface{})
	if userData["displayName"] != "Public User" {
		t.Error("expected displayName in public response")
	}
	if userData["avatarUrl"] != "https://example.com/avatar.jpg" {
		t.Error("expected avatarUrl in public response")
	}
	if userData["bio"] != "Hello world" {
		t.Error("expected bio in public response")
	}
}

func TestPublicUserHandlerReturns404ForNonexistentUser(t *testing.T) {
	store.Reset()
	req, _ := http.NewRequest("GET", "/api/users/nonexistent", nil)
	w := httptest.NewRecorder()

	PublicUserHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestUsersListHandlerFiltersPrivateProfiles(t *testing.T) {
	store.Reset()

	// Add public user
	publicUser := model.User{
		ID:            "public1",
		Username:      "public1",
		DisplayName:   "Public One",
		PublicProfile: true,
		Resources:     map[string]int{},
		Preferences:   map[string]string{},
	}
	store.AddUser("token1", publicUser)

	// Add private user
	privateUser := model.User{
		ID:            "private1",
		Username:      "private1",
		DisplayName:   "Private One",
		PublicProfile: false,
		Resources:     map[string]int{},
		Preferences:   map[string]string{},
	}
	store.AddUser("token2", privateUser)

	// Add another public user
	publicUser2 := model.User{
		ID:            "public2",
		Username:      "public2",
		DisplayName:   "Public Two",
		PublicProfile: true,
		Resources:     map[string]int{},
		Preferences:   map[string]string{},
	}
	store.AddUser("token3", publicUser2)

	req, _ := http.NewRequest("GET", "/api/users", nil)
	w := httptest.NewRecorder()

	UsersListHandler(w, req)

	var resp map[string][]any
	json.NewDecoder(w.Body).Decode(&resp)

	users := resp["users"]
	if len(users) != 2 {
		t.Fatalf("expected 2 public users, got %d", len(users))
	}

	// Verify private user is not in list
	for _, u := range users {
		userData := u.(map[string]interface{})
		if userData["displayName"] == "Private One" {
			t.Error("expected private user to be filtered out")
		}
	}
}

func TestUsersListHandlerRequiresGET(t *testing.T) {
	store.Reset()
	req, _ := http.NewRequest("POST", "/api/users", nil)
	w := httptest.NewRecorder()

	UsersListHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestProfileUpdateCanUpdateAllFields(t *testing.T) {
	store.Reset()
	user := model.User{
		ID:            "testuser",
		Username:      "testuser",
		DisplayName:   "Old Name",
		Bio:           "Old bio",
		PublicProfile: true,
		Resources:     map[string]int{},
		Preferences:   map[string]string{},
	}
	store.AddUser("token", user)

	updateReq := UpdateProfileRequest{
		DisplayName:   "New Name",
		Bio:           "New bio",
		AvatarURL:     "https://example.com/new.jpg",
		PublicProfile: ptrBool(false),
		Preferences: map[string]string{
			"theme": "dark",
			"lang":  "en",
		},
	}
	body, _ := json.Marshal(updateReq)
	req, _ := http.NewRequest("PUT", "/api/profile", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	ProfileHandler(w, req, user)

	var resp map[string]model.User
	json.NewDecoder(w.Body).Decode(&resp)
	updatedUser := resp["user"]

	if updatedUser.DisplayName != "New Name" {
		t.Error("expected DisplayName to be updated")
	}
	if updatedUser.Bio != "New bio" {
		t.Error("expected Bio to be updated")
	}
	if updatedUser.AvatarURL != "https://example.com/new.jpg" {
		t.Error("expected AvatarURL to be updated")
	}
	if updatedUser.PublicProfile {
		t.Error("expected PublicProfile to be false")
	}
	if updatedUser.Preferences["theme"] != "dark" {
		t.Error("expected Preferences to be updated")
	}
}

func TestIsValidImageURL(t *testing.T) {
	testCases := []struct {
		url      string
		expected bool
	}{
		{"", true},                                  // empty is ok
		{"https://example.com/image.jpg", true},    // valid https
		{"http://example.com/image.png", true},     // valid http
		{"ftp://example.com/image.jpg", false},     // invalid scheme
		{"not-a-url", false},                       // invalid format
		{"https://", false},                        // no host
		{"//example.com/image.jpg", false},         // no scheme
	}

	for _, tc := range testCases {
		result := isValidImageURL(tc.url)
		if result != tc.expected {
			t.Errorf("isValidImageURL(%q) = %v, expected %v", tc.url, result, tc.expected)
		}
	}
}

// Helper function
func ptrBool(b bool) *bool {
	return &b
}
