package properties

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetTextCreated(t *testing.T) {
	assert.Equal(t, "created", GetTextCreated(true))
	assert.Equal(t, "not created", GetTextCreated(false))
}

func TestGetSuccessOrTransitioning(t *testing.T) {
	assert.Equal(t, PropertyStatusSuccess, GetSuccessOrTransitioning(true))
	assert.Equal(t, PropertyStatusTransitioning, GetSuccessOrTransitioning(false))
}

func TestGetSuccessOrError(t *testing.T) {
	assert.Equal(t, PropertyStatusSuccess, GetSuccessOrError(true))
	assert.Equal(t, PropertyStatusError, GetSuccessOrError(false))
}
