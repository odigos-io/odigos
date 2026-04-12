package collectorconfig

import (
	"testing"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1/instrumentationrules"
	"github.com/odigos-io/odigos/common/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2/klogr"
)

func TestGetReceivers_AllNonEbpfUsesFilelogOnly(t *testing.T) {
	sources := &odigosv1.InstrumentationConfigList{
		Items: []odigosv1.InstrumentationConfig{
			makeInstrumentationConfig("default", "deployment-coupon", "coupon", false),
			makeInstrumentationConfig("shop", "deployment-cart", "cart", false),
		},
	}

	receivers, pipelineReceivers := getReceivers(klogr.New(), sources, "odigos-system")

	assert.Equal(t, []string{filelogReceiverName}, pipelineReceivers)
	require.Contains(t, receivers, filelogReceiverName)
	assert.NotContains(t, pipelineReceivers, odigosEbpfReceiverName)
}

func TestGetReceivers_AllEbpfUsesEbpfOnly(t *testing.T) {
	sources := &odigosv1.InstrumentationConfigList{
		Items: []odigosv1.InstrumentationConfig{
			makeInstrumentationConfig("default", "deployment-coupon", "coupon", true),
			makeInstrumentationConfig("shop", "deployment-cart", "cart", true),
		},
	}

	receivers, pipelineReceivers := getReceivers(klogr.New(), sources, "odigos-system")

	assert.Equal(t, []string{odigosEbpfReceiverName}, pipelineReceivers)
	assert.Empty(t, receivers)
}

func TestGetReceivers_MixedEbpfAndNonEbpfKeepsBothReceivers(t *testing.T) {
	sources := &odigosv1.InstrumentationConfigList{
		Items: []odigosv1.InstrumentationConfig{
			makeInstrumentationConfig("default", "deployment-coupon", "coupon", true),
			makeInstrumentationConfig("shop", "deployment-cart", "cart", false),
		},
	}

	receivers, pipelineReceivers := getReceivers(klogr.New(), sources, "odigos-system")

	assert.Equal(t, []string{filelogReceiverName, odigosEbpfReceiverName}, pipelineReceivers)

	filelogCfgAny, ok := receivers[filelogReceiverName]
	require.True(t, ok)
	filelogCfg, ok := filelogCfgAny.(config.GenericMap)
	require.True(t, ok)

	includesAny, ok := filelogCfg["include"]
	require.True(t, ok)
	includes, ok := includesAny.([]string)
	require.True(t, ok)

	// Only non-eBPF sources should be included in filelog globs.
	assert.Equal(t, []string{"/var/log/pods/shop_cart-*_*/*/*.log"}, includes)
}

func makeInstrumentationConfig(namespace string, name string, ownerName string, ebpfEnabled bool) odigosv1.InstrumentationConfig {
	ic := odigosv1.InstrumentationConfig{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			OwnerReferences: []v1.OwnerReference{
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       ownerName,
					UID:        types.UID(ownerName + "-uid"),
				},
			},
		},
		Spec: odigosv1.InstrumentationConfigSpec{
			SdkConfigs: []odigosv1.SdkConfig{
				{
					Language: "javascript",
				},
			},
		},
	}

	if ebpfEnabled {
		on := true
		ic.Spec.SdkConfigs[0].EbpfLogCapture = &instrumentationrules.EbpfLogCapture{
			Enabled: &on,
		}
	}

	return ic
}
