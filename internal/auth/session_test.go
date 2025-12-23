package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSessionManager_SetAndGetSession(t *testing.T) {
	sm := NewSessionManager(false)

	// Create a response recorder
	w := httptest.NewRecorder()

	// Set session
	err := sm.SetSession(w, 123)
	if err != nil {
		t.Fatalf("SetSession() error = %v", err)
	}

	// Check cookie was set
	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("No cookies set")
	}

	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "session" {
			sessionCookie = c
			break
		}
	}

	if sessionCookie == nil {
		t.Fatal("Session cookie not found")
	}

	// Create request with the cookie
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(sessionCookie)

	// Get session
	session, err := sm.GetSession(req)
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}

	if session.UserID != 123 {
		t.Errorf("UserID = %d, want 123", session.UserID)
	}

	if session.CreatedAt == 0 {
		t.Error("CreatedAt should not be 0")
	}
}

func TestSessionManager_InvalidCookie(t *testing.T) {
	sm := NewSessionManager(false)

	// Create request with invalid cookie
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  "session",
		Value: "invalid-cookie-value",
	})

	// Get session should fail
	_, err := sm.GetSession(req)
	if err == nil {
		t.Error("GetSession() should fail with invalid cookie")
	}
}

func TestSessionManager_NoCookie(t *testing.T) {
	sm := NewSessionManager(false)

	req := httptest.NewRequest("GET", "/", nil)

	_, err := sm.GetSession(req)
	if err == nil {
		t.Error("GetSession() should fail without cookie")
	}
}

func TestSessionManager_ClearSession(t *testing.T) {
	sm := NewSessionManager(false)

	w := httptest.NewRecorder()
	sm.ClearSession(w)

	cookies := w.Result().Cookies()
	for _, c := range cookies {
		if c.Name == "session" {
			if c.MaxAge >= 0 {
				t.Error("ClearSession should set MaxAge < 0")
			}
			return
		}
	}

	t.Fatal("Session cookie not found in response")
}
