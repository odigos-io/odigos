package workloadrollout

import (
	"testing"
	"encoding/hex"
	"github.com/stretchr/testify/assert"

	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
)

func TestHashForContainersConfig(t *testing.T) {
	// empty containers config
	hash, err := hashForContainersConfig([]odigosv1alpha1.ContainerConfig{})
	assert.NoError(t, err)
	assert.Equal(t, len(hash), 0)

	containersConfig := []odigosv1alpha1.ContainerConfig{
		{
			ContainerName:  "container1",
			Instrumented:   true,
			OtelDistroName: "otel-distro-1",
		},
		{
			ContainerName:  "container2",
			Instrumented:   false,
			OtelDistroName: "otel-distro-2",
		},
	}

	hash, err = hashForContainersConfig(containersConfig)
	assert.NoError(t, err)
	assert.Greater(t, len(hash), 0)

	// flip the order of the containers and check if the hash is the same

	containersConfig = []odigosv1alpha1.ContainerConfig{
		{
			ContainerName:  "container2",
			Instrumented:   false,
			OtelDistroName: "otel-distro-2",
		},
		{
			ContainerName:  "container1",
			Instrumented:   true,
			OtelDistroName: "otel-distro-1",
		},
	}

	hash2, err := hashForContainersConfig(containersConfig)
	assert.NoError(t, err)
	assert.Greater(t, len(hash2), 0)
	assert.Equal(t, hex.EncodeToString(hash), hex.EncodeToString(hash2))

	// flip the false instrumented flag and check if the hash is different
	containersConfig[1].Instrumented = true
	hash3, err := hashForContainersConfig(containersConfig)
	assert.NoError(t, err)
	assert.Greater(t, len(hash3), 0)
	assert.NotEqual(t, hex.EncodeToString(hash), hex.EncodeToString(hash3))

	// change the distro name and check if the hash is different
	containersConfig[1].OtelDistroName = "otel-distro-3"
	hash4, err := hashForContainersConfig(containersConfig)
	assert.NoError(t, err)
	assert.Greater(t, len(hash4), 0)
	assert.NotEqual(t, hex.EncodeToString(hash), hex.EncodeToString(hash4))
}
