package k8sconfig

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
)

// The Destination CR is the only production implementation of K8sExporterConfigurer, and
// clustercollector passes it by value.
var _ K8sExporterConfigurer = odigosv1.Destination{}

var (
	_ K8sConfiger = &Clickhouse{}
	_ K8sConfiger = &GoogleCloud{}
)

// clustercollector dispatches on the map key (`k8sConfigers[dest.GetType()]`), and a destination
// type that is not a key here silently gets no kubernetes-level configuration at all: no error, no
// condition, just a collector that cannot find its credentials. The keys are spelled out rather
// than taken from the common constants so that renaming a constant cannot silently move a key.
func TestLoadK8sConfigers_RegistersEveryKubernetesSpecificDestination(t *testing.T) {
	configers := LoadK8sConfigers()

	require.Len(t, configers, 3)

	assert.IsType(t, &Clickhouse{}, configers["clickhouse"])
	assert.IsType(t, &GoogleCloud{}, configers["googlecloud"])
	assert.IsType(t, &GoogleCloud{}, configers["googlecloudotlp"])
}

// Both Google Cloud destination types need the same credentials file, so they share one configer.
// Its DestType() therefore cannot match both keys: the registry key is the contract, DestType() is
// only the link back to the common config implementation.
func TestLoadK8sConfigers_KeyMatchesDestTypeExceptForTheGoogleCloudOtlpAlias(t *testing.T) {
	aliases := map[common.DestinationType]common.DestinationType{
		common.GoogleCloudOTLPDestinationType: common.GoogleCloudDestinationType,
	}

	for destType, configer := range LoadK8sConfigers() {
		expected := destType
		if aliased, isAlias := aliases[destType]; isAlias {
			expected = aliased
		}
		assert.Equal(t, expected, configer.DestType(),
			"the configer registered under %q reports the wrong destination type", destType)
	}
}
