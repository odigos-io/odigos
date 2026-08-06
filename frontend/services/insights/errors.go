package insights

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

var (
	ErrBadRequest          = errors.New("insights request is invalid")
	ErrNotFound            = errors.New("insights resource was not found")
	ErrConflict            = errors.New("insights request conflicts with the current state")
	ErrUnprocessableEntity = errors.New("insights request failed validation")
	ErrValidation          = ErrUnprocessableEntity
	ErrUnavailable         = errors.New("insights service is unavailable")
	ErrInternal            = errors.New("insights service encountered an internal error")
	ErrEngineStarting      = errors.New("insights engine is starting")
)

type ErrorCode string

const (
	ErrorCodeBadRequest          ErrorCode = "bad_request"
	ErrorCodeNotFound            ErrorCode = "not_found"
	ErrorCodeConflict            ErrorCode = "conflict"
	ErrorCodeUnprocessableEntity ErrorCode = "unprocessable_entity"
	ErrorCodeUnavailable         ErrorCode = "unavailable"
	ErrorCodeInternal            ErrorCode = "internal_error"
)

// APIError is the error envelope returned by the Insights REST API.
type APIError struct {
	StatusCode int
	Code       ErrorCode
	Message    string
	Details    json.RawMessage
}

func (e *APIError) Error() string {
	if e.Code == "" {
		return fmt.Sprintf("insights API returned status %d: %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("insights API returned %s (status %d): %s", e.Code, e.StatusCode, e.Message)
}

func (e *APIError) Is(target error) bool {
	var category error
	switch e.Code {
	case ErrorCodeBadRequest:
		category = ErrBadRequest
	case ErrorCodeNotFound:
		category = ErrNotFound
	case ErrorCodeConflict:
		category = ErrConflict
	case ErrorCodeUnprocessableEntity:
		category = ErrUnprocessableEntity
	case ErrorCodeUnavailable:
		category = ErrUnavailable
	case ErrorCodeInternal:
		category = ErrInternal
	}
	if category == nil {
		switch e.StatusCode {
		case http.StatusBadRequest:
			category = ErrBadRequest
		case http.StatusNotFound:
			category = ErrNotFound
		case http.StatusConflict:
			category = ErrConflict
		case http.StatusUnprocessableEntity:
			category = ErrUnprocessableEntity
		case http.StatusServiceUnavailable:
			category = ErrUnavailable
		default:
			if e.StatusCode >= http.StatusInternalServerError {
				category = ErrInternal
			}
		}
	}
	return target == category
}

// EngineStartingError distinguishes the retryable engine warm-up state from
// other service-unavailable responses while preserving the API error details.
type EngineStartingError struct {
	*APIError
}

func (e *EngineStartingError) Unwrap() error {
	return ErrEngineStarting
}

func isEngineStarting(statusCode int, body []byte, apiErr *APIError) bool {
	if statusCode != http.StatusServiceUnavailable {
		return false
	}
	if apiErr != nil && containsEngineStarting(apiErr.Message) {
		return true
	}
	return containsEngineStarting(string(body))
}

func containsEngineStarting(value string) bool {
	value = strings.ToLower(value)
	return strings.Contains(value, "engine starting") ||
		strings.Contains(value, "engine is starting") ||
		strings.Contains(value, "engine not ready")
}

// Codes carried in the GraphQL error extensions, so the UI can tell "feature is
// off" from "engine still warming up" from a real failure without matching
// message text.
const (
	CodeNotEnabled     = "INSIGHTS_NOT_ENABLED"
	CodeEngineStarting = "INSIGHTS_ENGINE_STARTING"
	CodeBadRequest     = "INSIGHTS_BAD_REQUEST"
	CodeNotFound       = "INSIGHTS_NOT_FOUND"
	CodeConflict       = "INSIGHTS_CONFLICT"
	CodeValidation     = "INSIGHTS_VALIDATION_FAILED"
	CodeUnavailable    = "INSIGHTS_UNAVAILABLE"
	CodeInternal       = "INSIGHTS_INTERNAL_ERROR"
)

// GraphQLError wraps an insights error for return from a resolver, tagging it
// with the matching extensions code.
func GraphQLError(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	return &gqlerror.Error{
		Err:        err,
		Message:    err.Error(),
		Path:       graphql.GetPath(ctx),
		Extensions: map[string]any{"code": errorCode(err)},
	}
}

func errorCode(err error) string {
	switch {
	case errors.Is(err, ErrNotEnabled):
		return CodeNotEnabled
	// Engine warm-up is a service-unavailable response too, so it has to be
	// matched before ErrUnavailable.
	case errors.Is(err, ErrEngineStarting):
		return CodeEngineStarting
	case errors.Is(err, ErrBadRequest):
		return CodeBadRequest
	case errors.Is(err, ErrNotFound):
		return CodeNotFound
	case errors.Is(err, ErrConflict):
		return CodeConflict
	case errors.Is(err, ErrUnprocessableEntity):
		return CodeValidation
	case errors.Is(err, ErrUnavailable):
		return CodeUnavailable
	default:
		return CodeInternal
	}
}
