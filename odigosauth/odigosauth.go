package odigosauth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	expectedIssuer  = "https://odigos.io"
	expectedSubject = "https://odigos.io/onprem"
)

func parseTokenUnverified(tokenString string) (jwt.MapClaims, error) {
	// NOTE: This intentionally does NOT verify the JWT signature.
	// It is only suitable for checking token shape and non-cryptographic claims (like exp/iss/sub/aud).
	parser := jwt.NewParser()
	token, _, err := parser.ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims")
	}

	return claims, nil
}

func checkTokenAttributes(tokenString string) (string, error) {
	claims, err := parseTokenUnverified(tokenString)
	if err != nil {
		return "", err
	}

	expTime, expErr := claims.GetExpirationTime()
	if expErr != nil {
		return "", fmt.Errorf("failed to get expiration time from token: %w", expErr)
	}
	if expTime == nil {
		return "", fmt.Errorf("missing exp claim")
	}
	if time.Now().After(expTime.Time) {
		expirationDuration := time.Since(expTime.Time)
		roundedDuration := expirationDuration.Round(time.Minute)
		return "", fmt.Errorf("token is expired for %v, contact Odigos support to issue a new one", roundedDuration)
	}

	iss, ok := claims["iss"].(string)
	if !ok || iss != expectedIssuer {
		return "", fmt.Errorf("invalid iss")
	}

	sub, ok := claims["sub"].(string)
	if !ok || sub != expectedSubject {
		return "", fmt.Errorf("invalid sub")
	}

	aud, ok := claims["aud"].(string)
	if !ok || aud == "" {
		return "", fmt.Errorf("missing aud claim")
	}

	return aud, nil
}

// / This function validates the Odigos onprem token and checks claims and the expiration time
func ValidateToken(onpremToken string) error {
	if onpremToken == "" {
		return fmt.Errorf("missing Odigos Pro token")
	}

	trimmedOnpremToken := strings.TrimSpace(onpremToken)
	_, err := checkTokenAttributes(trimmedOnpremToken)
	if err != nil {
		return fmt.Errorf("failed to verify onprem token: %w", err)
	}

	return nil
}

func ExtractJWTPayload(token string) (map[string]interface{}, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT token format")
	}

	// Decode the payload (second part of the JWT)
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode JWT payload: %w", err)
	}

	// Parse the payload as JSON
	var payload map[string]interface{}
	err = json.Unmarshal(payloadBytes, &payload)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JWT payload: %w", err)
	}

	return payload, nil
}
