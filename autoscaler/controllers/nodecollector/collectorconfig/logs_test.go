package collectorconfig

import (
	"testing"

	"github.com/go-logr/logr"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1/instrumentationrules"
	"github.com/odigos-io/odigos/common/config"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetReceiversAllFilelogSources(t *testing.T) {
	sources := &odigosv1.InstrumentationConfigList{
		Items: []odigosv1.InstrumentationConfig{
			newLogsInstrumentationConfig("default", "deployment-a"),
			newLogsInstrumentationConfig("other", "deployment-b"),
		},
	}

	receivers, pipelineReceivers := getReceivers(logr.Discard(), sources, "odigos-system")

	assert.Equal(t, []string{filelogReceiverName}, pipelineReceivers)
	filelogConfig := receivers[filelogReceiverName].(config.GenericMap)
	assert.ElementsMatch(t, []string{
		"/var/log/pods/default_deployment-a-*_*/*/*.log",
		"/var/log/pods/other_deployment-b-*_*/*/*.log",
	}, filelogConfig["include"])
}

func TestGetReceiversAllEbpfSources(t *testing.T) {
	sources := &odigosv1.InstrumentationConfigList{
		Items: []odigosv1.InstrumentationConfig{
			withEbpfLogCapture(newLogsInstrumentationConfig("default", "deployment-a"), "app"),
			withEbpfLogCapture(newLogsInstrumentationConfig("other", "deployment-b"), "app"),
		},
	}

	receivers, pipelineReceivers := getReceivers(logr.Discard(), sources, "odigos-system")

	assert.Equal(t, []string{odigosEbpfReceiverName}, pipelineReceivers)
	assert.NotContains(t, receivers, filelogReceiverName)
}

func TestGetReceiversMixedEbpfAndFilelogSources(t *testing.T) {
	sources := &odigosv1.InstrumentationConfigList{
		Items: []odigosv1.InstrumentationConfig{
			withEbpfLogCapture(newLogsInstrumentationConfig("default", "deployment-a"), "app"),
			newLogsInstrumentationConfig("other", "deployment-b"),
		},
	}

	receivers, pipelineReceivers := getReceivers(logr.Discard(), sources, "odigos-system")

	assert.Equal(t, []string{filelogReceiverName, odigosEbpfReceiverName}, pipelineReceivers)
	filelogConfig := receivers[filelogReceiverName].(config.GenericMap)
	assert.Equal(t, []string{"/var/log/pods/other_deployment-b-*_*/*/*.log"}, filelogConfig["include"])
}

func TestGetReceiversMixedContainersInSource(t *testing.T) {
	source := withEbpfLogCapture(newLogsInstrumentationConfig("default", "deployment-a"), "app")
	source.Spec.Containers = append(source.Spec.Containers, odigosv1.ContainerAgentConfig{ContainerName: "sidecar"})
	sources := &odigosv1.InstrumentationConfigList{
		Items: []odigosv1.InstrumentationConfig{source},
	}

	receivers, pipelineReceivers := getReceivers(logr.Discard(), sources, "odigos-system")

	assert.Equal(t, []string{filelogReceiverName, odigosEbpfReceiverName}, pipelineReceivers)
	filelogConfig := receivers[filelogReceiverName].(config.GenericMap)
	assert.Equal(t, []string{"/var/log/pods/default_deployment-a-*_*/sidecar/*.log"}, filelogConfig["include"])
}

func newLogsInstrumentationConfig(namespace string, ownerName string) odigosv1.InstrumentationConfig {
	return odigosv1.InstrumentationConfig{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			OwnerReferences: []metav1.OwnerReference{
				{Name: ownerName},
			},
		},
	}
}

func withEbpfLogCapture(ic odigosv1.InstrumentationConfig, containerName string) odigosv1.InstrumentationConfig {
	enabled := true
	ic.Spec.Containers = append(ic.Spec.Containers, odigosv1.ContainerAgentConfig{
		ContainerName: containerName,
		Logs: &odigosv1.AgentLogsConfig{
			EbpfLogCapture: &instrumentationrules.EbpfLogCapture{Enabled: &enabled},
		},
	})
	return ic
}
