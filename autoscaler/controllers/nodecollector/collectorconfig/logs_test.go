package collectorconfig

import (
	"testing"

	"github.com/go-logr/logr"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1/instrumentationrules"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetReceivers_DefaultsToFilelog(t *testing.T) {
	receivers, pipelineReceivers := getReceivers(logr.Discard(), nil, "odigos-system")

	require.Contains(t, receivers, filelogReceiverName)
	require.Equal(t, []string{filelogReceiverName}, pipelineReceivers)
}

func TestGetReceivers_UsesEbpfAndKeepsFilelogFallback(t *testing.T) {
	enabled := true
	sources := &odigosv1.InstrumentationConfigList{
		Items: []odigosv1.InstrumentationConfig{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "demo",
					Namespace: "default",
					OwnerReferences: []metav1.OwnerReference{
						{Name: "demo-deployment"},
					},
				},
				Spec: odigosv1.InstrumentationConfigSpec{
					SdkConfigs: []odigosv1.SdkConfig{
						{
							EbpfLogCapture: &instrumentationrules.EbpfLogCapture{
								Enabled: &enabled,
							},
						},
					},
				},
			},
		},
	}

	receivers, pipelineReceivers := getReceivers(logr.Discard(), sources, "odigos-system")

	require.Contains(t, receivers, filelogReceiverName)
	require.Equal(t, []string{odigosEbpfReceiverName, filelogReceiverName}, pipelineReceivers)
}
