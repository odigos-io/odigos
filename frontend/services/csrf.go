package services

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net/http"
	"sync"
	"time"
)

const (
	CSRFTokenHeader = "X-CSRF-Token"
	CSRFCookieName  = "csrf_token"
	CSRFTokenLength = 32
	CSRFTokenTTL    = 24 * time.Hour
)

var (
	ErrCSRFTokenMissing = errors.New("CSRF token missing")
	ErrCSRFTokenInvalid = errors.New("CSRF token invalid")
	ErrCSRFTokenExpired = errors.New("CSRF token expired")
)

type CSRFToken struct {
	Value     string
	ExpiresAt time.Time
}

type CSRFService struct {
	tokens map[string]CSRFToken
	mu     sync.RWMutex
}

var csrfService *CSRFService
var csrfOnce sync.Once

func GetCSRFService() *CSRFService {
	csrfOnce.Do(func() {
		csrfService = &CSRFService{
			tokens: make(map[string]CSRFToken),
		}
		// Start cleanup goroutine
		go csrfService.cleanup()
	})
	return csrfService
}

// GenerateToken creates a new CSRF token
func (c *CSRFService) GenerateToken() (string, error) {
	// Generate random bytes
	bytes := make([]byte, CSRFTokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	// Encode to base64
	token := base64.URLEncoding.EncodeToString(bytes)

	// Store token with expiration
	c.mu.Lock()
	c.tokens[token] = CSRFToken{
		Value:     token,
		ExpiresAt: time.Now().Add(CSRFTokenTTL),
	}
	c.mu.Unlock()

	return token, nil
}

// ValidateToken checks if the provided token is valid and not expired
func (c *CSRFService) ValidateToken(token string) error {
	if token == "" {
		return ErrCSRFTokenMissing
	}

	c.mu.RLock()
	storedToken, exists := c.tokens[token]
	c.mu.RUnlock()

	if !exists {
		return ErrCSRFTokenInvalid
	}

	if time.Now().After(storedToken.ExpiresAt) {
		// Clean up expired token
		c.mu.Lock()
		delete(c.tokens, token)
		c.mu.Unlock()
		return ErrCSRFTokenExpired
	}

	// Use constant-time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare([]byte(token), []byte(storedToken.Value)) != 1 {
		return ErrCSRFTokenInvalid
	}

	return nil
}

// ValidateRequest validates CSRF token from request header or form
func (c *CSRFService) ValidateRequest(r *http.Request) error {
	// Try to get token from header first
	token := r.Header.Get(CSRFTokenHeader)

	// If not in header, try form value
	if token == "" {
		token = r.FormValue("csrf_token")
	}

	// If still not found, try from query parameters
	if token == "" {
		token = r.URL.Query().Get("csrf_token")
	}

	return c.ValidateToken(token)
}

// SetCSRFCookie sets the CSRF token as a cookie
func (c *CSRFService) SetCSRFCookie(w http.ResponseWriter, r *http.Request, token string) {
	cookie := &http.Cookie{
		Name:     CSRFCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: false, // Allow JavaScript access for AJAX requests
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(CSRFTokenTTL.Seconds()),
	}
	http.SetCookie(w, cookie)
}

// GetCSRFToken retrieves CSRF token from cookie
func (c *CSRFService) GetCSRFToken(r *http.Request) string {
	cookie, err := r.Cookie(CSRFCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

// RefreshToken generates a new token and invalidates the old one
func (c *CSRFService) RefreshToken(oldToken string) (string, error) {
	// Remove old token if it exists
	if oldToken != "" {
		c.mu.Lock()
		delete(c.tokens, oldToken)
		c.mu.Unlock()
	}

	// Generate new token
	return c.GenerateToken()
}

// cleanup removes expired tokens periodically
func (c *CSRFService) cleanup() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		c.mu.Lock()
		for token, csrfToken := range c.tokens {
			if now.After(csrfToken.ExpiresAt) {
				delete(c.tokens, token)
			}
		}
		c.mu.Unlock()
	}
}
