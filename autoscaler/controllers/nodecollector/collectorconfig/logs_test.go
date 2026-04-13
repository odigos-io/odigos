package collectorconfig

import (
	"testing"

	"github.com/go-logr/logr"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1/instrumentationrules"
	"github.com/odigos-io/odigos/common/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetReceivers_AllNonEbpfSources_UsesFilelogOnly(t *testing.T) {
	sources := &odigosv1.InstrumentationConfigList{
		Items: []odigosv1.InstrumentationConfig{
			makeIC("default", "deployment-checkout", "checkout", false),
			makeIC("default", "deployment-catalog", "catalog", false),
		},
	}

	receivers, pipelineReceivers := getReceivers(logr.Discard(), sources, "odigos-system")

	require.Equal(t, []string{filelogReceiverName}, pipelineReceivers)
	require.Contains(t, receivers, filelogReceiverName)

	filelogCfg, ok := receivers[filelogReceiverName].(config.GenericMap)
	require.True(t, ok)

	includes, ok := filelogCfg["include"].([]string)
	require.True(t, ok)
	assert.ElementsMatch(t, []string{
		"/var/log/pods/default_checkout-*_*/*/*.log",
		"/var/log/pods/default_catalog-*_*/*/*.log",
	}, includes)
}

func TestGetReceivers_AllEbpfSources_UsesEbpfOnly(t *testing.T) {
	sources := &odigosv1.InstrumentationConfigList{
		Items: []odigosv1.InstrumentationConfig{
			makeIC("default", "deployment-checkout", "checkout", true),
			makeIC("default", "deployment-catalog", "catalog", true),
		},
	}

	receivers, pipelineReceivers := getReceivers(logr.Discard(), sources, "odigos-system")

	assert.Empty(t, receivers)
	assert.Equal(t, []string{odigosEbpfReceiverName}, pipelineReceivers)
}

func TestGetReceivers_MixedSources_KeepsFilelogAndEbpf(t *testing.T) {
	sources := &odigosv1.InstrumentationConfigList{
		Items: []odigosv1.InstrumentationConfig{
			makeIC("default", "deployment-checkout", "checkout", false),
			makeIC("default", "deployment-catalog", "catalog", true),
		},
	}

	receivers, pipelineReceivers := getReceivers(logr.Discard(), sources, "odigos-system")

	assert.Equal(t, []string{filelogReceiverName, odigosEbpfReceiverName}, pipelineReceivers)
	require.Contains(t, receivers, filelogReceiverName)

	filelogCfg, ok := receivers[filelogReceiverName].(config.GenericMap)
	require.True(t, ok)

	includes, ok := filelogCfg["include"].([]string)
	require.True(t, ok)
	assert.Equal(t, []string{"/var/log/pods/default_checkout-*_*/*/*.log"}, includes)
}

func makeIC(namespace, runtimeObjectName, ownerName string, ebpfEnabled bool) odigosv1.InstrumentationConfig {
	ic := odigosv1.InstrumentationConfig{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      runtimeObjectName,
			OwnerReferences: []metav1.OwnerReference{
				{Name: ownerName},
			},
		},
		Spec: odigosv1.InstrumentationConfigSpec{
			SdkConfigs: []odigosv1.SdkConfig{
				{
					EbpfLogCapture: &instrumentationrules.EbpfLogCapture{
						Enabled: boolPtr(ebpfEnabled),
					},
				},
			},
		},
	}
	return ic
}

func boolPtr(v bool) *bool {
	return &v
}
