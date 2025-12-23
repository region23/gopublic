package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
)

// GenerateSecureToken creates a cryptographically secure API token.
// Returns a token in format: sk_live_<base64-encoded-32-random-bytes>
func GenerateSecureToken() (string, error) {
	bytes := make([]byte, 32) // 256 bits of entropy
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "sk_live_" + base64.URLEncoding.EncodeToString(bytes), nil
}

// HashToken returns SHA256 hash of the token for secure storage.
// We store the hash, not the plaintext token.
func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
