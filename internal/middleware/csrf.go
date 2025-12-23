package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	csrfCookieName = "csrf_token"
	csrfHeaderName = "X-CSRF-Token"
	csrfFormField  = "csrf_token"
)

// CSRFConfig holds configuration for CSRF middleware
type CSRFConfig struct {
	Secure bool // Whether to set Secure flag on cookie
}

// generateCSRFToken creates a random token
func generateCSRFToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// SetCSRFToken middleware generates CSRF token for GET requests
func SetCSRFToken(cfg *CSRFConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if token already exists
		if _, err := c.Cookie(csrfCookieName); err != nil {
			// Generate new token
			token := generateCSRFToken()

			http.SetCookie(c.Writer, &http.Cookie{
				Name:     csrfCookieName,
				Value:    token,
				Path:     "/",
				MaxAge:   24 * 60 * 60, // 24 hours
				Secure:   cfg.Secure,
				HttpOnly: false, // Must be readable by JavaScript
				SameSite: http.SameSiteLaxMode,
			})

			// Make token available to templates
			c.Set("csrf_token", token)
		} else {
			// Read existing token for templates
			token, _ := c.Cookie(csrfCookieName)
			c.Set("csrf_token", token)
		}

		c.Next()
	}
}

// ValidateCSRF middleware validates CSRF token for unsafe methods
func ValidateCSRF() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip safe methods
		method := c.Request.Method
		if method == "GET" || method == "HEAD" || method == "OPTIONS" {
			c.Next()
			return
		}

		// Get token from cookie
		cookieToken, err := c.Cookie(csrfCookieName)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "CSRF token missing",
			})
			return
		}

		// Get token from header or form
		requestToken := c.GetHeader(csrfHeaderName)
		if requestToken == "" {
			requestToken = c.PostForm(csrfFormField)
		}

		// Validate
		if requestToken == "" || requestToken != cookieToken {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "CSRF token invalid",
			})
			return
		}

		c.Next()
	}
}

// GetCSRFToken returns the current CSRF token from context
func GetCSRFToken(c *gin.Context) string {
	if token, exists := c.Get("csrf_token"); exists {
		return token.(string)
	}
	return ""
}
