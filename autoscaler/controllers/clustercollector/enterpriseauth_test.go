package clustercollector

import (
	"testing"

	"github.com/odigos-io/odigos/common/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddEnterpriseAuthExtension_NilExtensionsMap(t *testing.T) {
	c := &config.Config{}

	addEnterpriseAuthExtension(c)

	require.Contains(t, c.Extensions, odigosEnterpriseAuthExtensionName)
	assert.Equal(t, []string{odigosEnterpriseAuthExtensionName}, c.Service.Extensions)
}

func TestAddEnterpriseAuthExtension_KeepsExistingExtensions(t *testing.T) {
	c := &config.Config{
		Extensions: config.GenericMap{
			"health_check": config.GenericMap{"endpoint": "0.0.0.0:13133"},
			"pprof":        config.GenericMap{"endpoint": "0.0.0.0:1777"},
		},
		Service: config.Service{
			Extensions: []string{"health_check", "pprof"},
		},
	}

	addEnterpriseAuthExtension(c)

	assert.Contains(t, c.Extensions, "health_check")
	assert.Contains(t, c.Extensions, "pprof")
	assert.Contains(t, c.Extensions, odigosEnterpriseAuthExtensionName)
	assert.Equal(t, []string{"health_check", "pprof", odigosEnterpriseAuthExtensionName}, c.Service.Extensions)
}

// A duplicate entry in service.extensions is a collector config error, so the helper must stay
// idempotent even if it ends up being called more than once on the same config.
func TestAddEnterpriseAuthExtension_Idempotent(t *testing.T) {
	c := &config.Config{}

	addEnterpriseAuthExtension(c)
	addEnterpriseAuthExtension(c)

	assert.Equal(t, []string{odigosEnterpriseAuthExtensionName}, c.Service.Extensions)
	assert.Len(t, c.Extensions, 1)
}
