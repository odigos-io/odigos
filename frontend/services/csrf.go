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

// CSRFService holds CSRF helpers. Validation uses the double-submit cookie pattern
// (header/form/query token must match the csrf_token cookie) so it works across
// multiple UI/backend replicas without shared in-memory state.
type CSRFService struct{}

var csrfService *CSRFService
var csrfOnce sync.Once

func GetCSRFService() *CSRFService {
	csrfOnce.Do(func() {
		csrfService = &CSRFService{}
	})
	return csrfService
}

func looksLikeIssuedCSRFToken(token string) bool {
	if token == "" {
		return false
	}
	decoded, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		return false
	}
	return len(decoded) == CSRFTokenLength
}

// GenerateToken creates a new CSRF token (entropy only; validity is double-submit vs cookie).
func (c *CSRFService) GenerateToken() (string, error) {
	bytes := make([]byte, CSRFTokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// ValidateToken reports whether a value looks like a token we would issue (shape check only).
// Per-request validation is ValidateRequest (double-submit).
func (c *CSRFService) ValidateToken(token string) error {
	if token == "" {
		return ErrCSRFTokenMissing
	}
	if !looksLikeIssuedCSRFToken(token) {
		return ErrCSRFTokenInvalid
	}
	return nil
}

// ValidateRequest checks double-submit: submitted token must match the csrf_token cookie.
func (c *CSRFService) ValidateRequest(r *http.Request) error {
	submit := r.Header.Get(CSRFTokenHeader)
	if submit == "" {
		submit = r.FormValue("csrf_token")
	}
	if submit == "" {
		submit = r.URL.Query().Get("csrf_token")
	}
	if submit == "" {
		return ErrCSRFTokenMissing
	}
	cookie := c.GetCSRFToken(r)
	if cookie == "" {
		return ErrCSRFTokenMissing
	}
	if subtle.ConstantTimeCompare([]byte(submit), []byte(cookie)) != 1 {
		return ErrCSRFTokenInvalid
	}
	if !looksLikeIssuedCSRFToken(submit) {
		return ErrCSRFTokenInvalid
	}
	return nil
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

// RefreshToken generates a new token and replaces the prior cookie on the next SetCSRFCookie.
func (c *CSRFService) RefreshToken(_ string) (string, error) {
	return c.GenerateToken()
}
