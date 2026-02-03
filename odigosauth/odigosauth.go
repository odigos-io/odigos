package odigosauth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	expectedIssuer  = "https://odigos.io"
	expectedSubject = "https://odigos.io/onprem"
)

func checkTokenAttributes(tokenPayload map[string]interface{}) (string, error) {
	// exp
	exp, ok := tokenPayload["exp"]
	if !ok {
		return "", fmt.Errorf("missing exp claim")
	}
	expFloat, ok := exp.(float64)
	if !ok {
		// json.Unmarshal uses float64 for numbers; if tokenPayload was built differently, be explicit.
		return "", fmt.Errorf("invalid exp claim type")
	}

	expTime := time.Unix(int64(expFloat), 0)
	if time.Now().After(expTime) {
		expirationDuration := time.Since(expTime)
		roundedDuration := expirationDuration.Round(time.Minute)
		return "", fmt.Errorf("token is expired for %v, contact Odigos support to issue a new one", roundedDuration)
	}

	// iss
	iss, ok := tokenPayload["iss"].(string)
	if !ok || iss != expectedIssuer {
		return "", fmt.Errorf("invalid iss")
	}

	// sub
	sub, ok := tokenPayload["sub"].(string)
	if !ok || sub != expectedSubject {
		return "", fmt.Errorf("invalid sub")
	}

	// aud (support both string and string array)
	switch aud := tokenPayload["aud"].(type) {
	case string:
		if aud == "" {
			return "", fmt.Errorf("missing aud claim")
		}
		return aud, nil
	case []interface{}:
		if len(aud) == 0 {
			return "", fmt.Errorf("missing aud claim")
		}
		aud0, ok := aud[0].(string)
		if !ok || aud0 == "" {
			return "", fmt.Errorf("invalid aud claim")
		}
		return aud0, nil
	default:
		return "", fmt.Errorf("missing aud claim")
	}
}

// This function validates the Odigos onprem token and checks claims and the expiration time
func ValidateToken(onpremToken string) (map[string]interface{}, error) {
	if onpremToken == "" {
		return nil, fmt.Errorf("missing Odigos Pro token")
	}

	trimmedOnpremToken := strings.TrimSpace(onpremToken)
	tokenPayload, err := extractJWTPayload(trimmedOnpremToken)
	if err != nil {
		return nil, err
	}
	_, err = checkTokenAttributes(tokenPayload)
	if err != nil {
		return nil, err
	}

	return tokenPayload, nil
}

func extractJWTPayload(token string) (map[string]interface{}, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT token format")
	}

	// Decode the payload (second part of the JWT)
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode JWT payload")
	}

	// Parse the payload as JSON
	var payload map[string]interface{}
	err = json.Unmarshal(payloadBytes, &payload)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JWT payload: %w", err)
	}

	return payload, nil
}
