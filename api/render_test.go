package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetCRD(t *testing.T) {
	crds, err := GetCRDs([]string{"odigos.io_instrumentationconfigs.yaml"})
	assert.NoError(t, err)
	assert.NotEmpty(t, crds)
	for _, crd := range crds {
		assert.NotEqual(t, "instrumentationconfigs.odigos.io", crd.Name)
	}
}