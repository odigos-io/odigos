package collectorconfig

import (
	"testing"

	"github.com/go-logr/logr"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1/instrumentationrules"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetReceivers_DefaultsToFilelogWhenNoSources(t *testing.T) {
	receivers, pipelineReceivers := getReceivers(logr.Discard(), &odigosv1.InstrumentationConfigList{}, "odigos-system")

	assert.Equal(t, []string{filelogReceiverName}, pipelineReceivers)
	require.Contains(t, receivers, filelogReceiverName)
}

func TestGetReceivers_UsesOnlyEbpfWhenAllSourcesEnableEbpf(t *testing.T) {
	sources := &odigosv1.InstrumentationConfigList{
		Items: []odigosv1.InstrumentationConfig{
			newInstrumentationConfig("default", "svc-a", true),
			newInstrumentationConfig("default", "svc-b", true),
		},
	}

	receivers, pipelineReceivers := getReceivers(logr.Discard(), sources, "odigos-system")

	assert.Equal(t, []string{odigosEbpfReceiverName}, pipelineReceivers)
	assert.Empty(t, receivers)
}

func TestGetReceivers_MixedSourcesKeepFilelogAndEnableEbpf(t *testing.T) {
	sources := &odigosv1.InstrumentationConfigList{
		Items: []odigosv1.InstrumentationConfig{
			newInstrumentationConfig("default", "svc-ebpf", true),
			newInstrumentationConfig("default", "svc-filelog", false),
		},
	}

	receivers, pipelineReceivers := getReceivers(logr.Discard(), sources, "odigos-system")

	assert.Equal(t, []string{filelogReceiverName, odigosEbpfReceiverName}, pipelineReceivers)
	require.Contains(t, receivers, filelogReceiverName)

	filelogCfg, ok := receivers[filelogReceiverName].(config.GenericMap)
	require.True(t, ok)

	includes, ok := filelogCfg["include"].([]string)
	require.True(t, ok)
	assert.Equal(t, []string{"/var/log/pods/default_svc-filelog-*_*/*/*.log"}, includes)
}

func newInstrumentationConfig(namespace string, workloadName string, ebpfEnabled bool) odigosv1.InstrumentationConfig {
	var ebpfLogCapture *instrumentationrules.EbpfLogCapture
	if ebpfEnabled {
		ebpfLogCapture = &instrumentationrules.EbpfLogCapture{Enabled: boolPtr(true)}
	}

	return odigosv1.InstrumentationConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      workloadName,
			Namespace: namespace,
			OwnerReferences: []metav1.OwnerReference{
				{Name: workloadName, Kind: "Deployment"},
			},
		},
		Spec: odigosv1.InstrumentationConfigSpec{
			SdkConfigs: []odigosv1.SdkConfig{
				{
					Language:       common.GoProgrammingLanguage,
					EbpfLogCapture: ebpfLogCapture,
				},
			},
		},
	}
}

func boolPtr(v bool) *bool {
	return &v
}
