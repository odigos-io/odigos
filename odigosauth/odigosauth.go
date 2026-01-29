package odigosauth

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// RsaPublicKeyString is the RSA public key used to verify Odigos enterprise tokens
const RsaPublicKeyString = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAqjGWuv6zbpSmSA5Ygaw5
7ZLIO7U9bNjyWYzdVyToeayRT7sfKY313W0vbRgZySxEOwyeiHYrF/vvKn2MQx4e
V9d7oNDnL8otNzNSp7S6BsXxZCALlk8YtVPQOV2a7uzrtNK4AS/EB6NuqZHJwvH9
DmqEr0dkIbvoBjtWUo70ez6lBE1sQEShoORb11CVnGQwDBaeIXTL6+ajCFzyea0D
l9NYI4XJhhdZGedb44mhNNiigJ09z+5KHemhwUHwrLvnZKrAKkUtZlNvu8JR9eU8
daEeTbWsOQsnXuxSqxTlEaRddXbDNZjRyP7KtlMIrTII+MYEoigxHkO2nZmCBYhP
wQIDAQAB
-----END PUBLIC KEY-----`

// testPublicKeyString is used for testing purposes to override the default public key
var testPublicKeyString string

func parseToken(tokenString string) (*jwt.Token, error) {
	// Use test public key if set, otherwise use the default
	publicKeyStr := RsaPublicKeyString
	if testPublicKeyString != "" {
		publicKeyStr = testPublicKeyString
	}

	// Parse the RSA public key
	publicKey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(publicKeyStr))
	if err != nil {
		return nil, err
	}
	// Parse the token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return publicKey, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			expTime, expTimeErr := token.Claims.GetExpirationTime()
			if expTimeErr != nil {
				return nil, fmt.Errorf("failed to get expiration time from token: %w", errors.Join(err, expTimeErr))
			}

			now := time.Now()
			expirationDuration := now.Sub(expTime.Time)
			roundedDuration := expirationDuration.Round(time.Minute)
			return nil, fmt.Errorf("token is expired for %v, contact Odigos support to issue a new one", roundedDuration)
		}

		return nil, err
	}

	return token, nil
}

func checkTokenAttributes(tokenString string) (string, error) {
	token, err := parseToken(tokenString)
	if err != nil {
		return "", err
	}
	// Check if the token is valid
	claims, claimsok := token.Claims.(jwt.MapClaims)
	if !claimsok {
		return "", fmt.Errorf("invalid claims")
	}

	if !token.Valid {
		return "", fmt.Errorf("invalid token")
	}

	iss, ok := claims["iss"].(string)
	if !ok || iss != "https://odigos.io" {
		return "", fmt.Errorf("invalid iss")
	}

	sub, ok := claims["sub"].(string)
	if !ok || sub != "https://odigos.io/onprem" {
		return "", fmt.Errorf("invalid sub")
	}

	aud, ok := claims["aud"].(string)
	if !ok || aud == "" {
		return "", fmt.Errorf("missing aud claim")
	}

	return aud, nil
}

func ValidateToken(onpremToken string) error {
	if onpremToken == "" {
		return fmt.Errorf("missing Odigos Pro token")
	}

	trimmedOnpremToken := strings.TrimSpace(onpremToken)
	aud, err := checkTokenAttributes(trimmedOnpremToken)
	if err != nil {
		return fmt.Errorf("failed to verify onprem token: %w", err)
	}

	fmt.Println("Odigos onprem token verified", "audience", aud)
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
