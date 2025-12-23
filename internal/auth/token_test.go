package auth

import (
	"strings"
	"testing"
)

func TestGenerateSecureToken(t *testing.T) {
	token, err := GenerateSecureToken()
	if err != nil {
		t.Fatalf("GenerateSecureToken() error = %v", err)
	}

	// Check prefix
	if !strings.HasPrefix(token, "sk_live_") {
		t.Errorf("Token should start with 'sk_live_', got %s", token)
	}

	// Check length (prefix + base64 of 32 bytes = 8 + 43 = 51 chars)
	if len(token) < 40 {
		t.Errorf("Token too short: %d chars", len(token))
	}
}

func TestGenerateSecureToken_Uniqueness(t *testing.T) {
	tokens := make(map[string]bool)

	for i := 0; i < 100; i++ {
		token, err := GenerateSecureToken()
		if err != nil {
			t.Fatalf("GenerateSecureToken() error = %v", err)
		}

		if tokens[token] {
			t.Errorf("Duplicate token generated: %s", token)
		}
		tokens[token] = true
	}
}

func TestHashToken(t *testing.T) {
	token := "sk_live_test123"

	hash1 := HashToken(token)
	hash2 := HashToken(token)

	// Same input should produce same hash
	if hash1 != hash2 {
		t.Errorf("HashToken not deterministic: %s != %s", hash1, hash2)
	}

	// Hash should be hex string of SHA256 (64 chars)
	if len(hash1) != 64 {
		t.Errorf("Hash length should be 64, got %d", len(hash1))
	}

	// Different input should produce different hash
	hash3 := HashToken("different_token")
	if hash1 == hash3 {
		t.Error("Different tokens produced same hash")
	}
}
