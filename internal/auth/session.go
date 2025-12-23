package auth

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/securecookie"
)

// SessionManager handles secure cookie encoding/decoding
type SessionManager struct {
	sc       *securecookie.SecureCookie
	isSecure bool // Whether to set Secure flag on cookies
}

// SessionData represents the data stored in session cookie
type SessionData struct {
	UserID    uint  `json:"user_id"`
	CreatedAt int64 `json:"created_at"`
}

// NewSessionManager creates a new session manager.
// Keys are read from environment or generated randomly (not recommended for production).
func NewSessionManager(isSecure bool) *SessionManager {
	hashKey := getOrGenerateKey("SESSION_HASH_KEY", 32)
	blockKey := getOrGenerateKey("SESSION_BLOCK_KEY", 32)

	sc := securecookie.New(hashKey, blockKey)
	sc.MaxAge(30 * 24 * 60 * 60) // 30 days

	return &SessionManager{
		sc:       sc,
		isSecure: isSecure,
	}
}

// getOrGenerateKey reads key from environment or generates a random one
func getOrGenerateKey(envVar string, length int) []byte {
	keyHex := os.Getenv(envVar)
	if keyHex != "" {
		key, err := hex.DecodeString(keyHex)
		if err == nil && len(key) >= length {
			return key[:length]
		}
		log.Printf("Warning: %s is invalid, generating random key", envVar)
	}

	// Generate random key (sessions won't persist across restarts)
	key := make([]byte, length)
	if _, err := rand.Read(key); err != nil {
		log.Fatalf("Failed to generate session key: %v", err)
	}
	log.Printf("Warning: %s not set, using random key (sessions won't persist)", envVar)
	return key
}

// SetSession creates a signed session cookie
func (sm *SessionManager) SetSession(w http.ResponseWriter, userID uint) error {
	data := SessionData{
		UserID:    userID,
		CreatedAt: time.Now().Unix(),
	}

	encoded, err := sm.sc.Encode("session", data)
	if err != nil {
		return err
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    encoded,
		Path:     "/",
		MaxAge:   30 * 24 * 60 * 60, // 30 days
		Secure:   sm.isSecure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	return nil
}

// GetSession reads and validates session cookie
func (sm *SessionManager) GetSession(r *http.Request) (*SessionData, error) {
	cookie, err := r.Cookie("session")
	if err != nil {
		return nil, err
	}

	var data SessionData
	if err := sm.sc.Decode("session", cookie.Value, &data); err != nil {
		return nil, err
	}

	return &data, nil
}

// ClearSession removes the session cookie
func (sm *SessionManager) ClearSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Secure:   sm.isSecure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}
