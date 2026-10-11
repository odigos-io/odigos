package insights

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

func TestGraphQLErrorCarriesCode(t *testing.T) {
	_, invalidID := ParseID("not-a-number")
	require.Error(t, invalidID)

	tests := []struct {
		name string
		err  error
		code string
	}{
		{name: "not enabled", err: ErrNotEnabled, code: CodeNotEnabled},
		{name: "wrapped not enabled", err: fmt.Errorf("listing findings: %w", ErrNotEnabled), code: CodeNotEnabled},
		{name: "bad request", err: ErrBadRequest, code: CodeBadRequest},
		{name: "invalid id argument", err: invalidID, code: CodeBadRequest},
		{name: "not found", err: ErrNotFound, code: CodeNotFound},
		{name: "conflict", err: ErrConflict, code: CodeConflict},
		{name: "validation", err: ErrValidation, code: CodeValidation},
		{name: "unavailable", err: ErrUnavailable, code: CodeUnavailable},
		{name: "internal", err: ErrInternal, code: CodeInternal},
		{name: "unclassified", err: fmt.Errorf("connection refused"), code: CodeInternal},
		// Engine warm-up arrives as a 503, so it must win over unavailable.
		{name: "engine starting", err: decodeAPIError(http.StatusServiceUnavailable, []byte("engine starting")), code: CodeEngineStarting},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := GraphQLError(context.Background(), tt.err)

			var gqlErr *gqlerror.Error
			require.ErrorAs(t, err, &gqlErr)
			assert.Equal(t, tt.code, gqlErr.Extensions["code"])
			assert.Equal(t, tt.err.Error(), gqlErr.Message)
			assert.ErrorIs(t, err, tt.err)
		})
	}
}

func TestGraphQLErrorPassesThroughNil(t *testing.T) {
	assert.NoError(t, GraphQLError(context.Background(), nil))
}
