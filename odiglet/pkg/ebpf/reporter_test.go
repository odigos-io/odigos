package ebpf

import (
	"context"
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/api/agentsignalconfig"
	"github.com/odigos-io/odigos/common/api/sampling"
	instance "github.com/odigos-io/odigos/k8sutils/pkg/instrumentation_instance"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestOnConfigReportsHeadSamplingOnlyWhenAsked(t *testing.T) {
	ctx := context.Background()
	scheme := runtime.NewScheme()
	require.NoError(t, odigosv1.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "payments-a", Namespace: "shop", UID: "uid-1"}}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(pod).WithStatusSubresource(&odigosv1.InstrumentationInstance{}).Build()
	r := &k8sReporter{client: c}
	details := &K8sProcessDetails{Pod: pod, ContainerName: "app",
		Pw: &k8sconsts.PodWorkload{Namespace: "shop", Kind: k8sconsts.WorkloadKindDeployment, Name: "payments"}}
	fifty := 50.0
	headSampling := &sampling.HeadSamplingConfig{NoisyOperations: []sampling.NoisyOperation{{Id: "abc", PercentageAtMost: &fifty}}}
	config := func(report bool) *odigosv1.ContainerAgentConfig {
		return &odigosv1.ContainerAgentConfig{ContainerName: "app",
			Traces: &agentsignalconfig.AgentTracesConfig{HeadSampling: headSampling, ReportHeadSampling: report}}
	}
	instances := func() []odigosv1.InstrumentationInstance {
		t.Helper()
		var list odigosv1.InstrumentationInstanceList
		require.NoError(t, c.List(ctx, &list, client.InNamespace("shop")))
		return list.Items
	}

	require.NoError(t, r.OnConfig(ctx, 42, nil, details, config(false)))
	require.NoError(t, r.OnConfig(ctx, 42, assert.AnError, details, config(false)))
	require.NoError(t, r.OnConfig(ctx, 42, nil, details, &odigosv1.ContainerAgentConfig{ContainerName: "app"}))
	assert.Empty(t, instances(), "a container no trace surge covers writes nothing")

	require.NoError(t, r.OnConfig(ctx, 42, nil, details, config(true)))
	written := instances()
	require.Len(t, written, 1)
	attrs := map[string]string{}
	for _, a := range written[0].Status.NonIdentifyingAttributes {
		attrs[a.Key] = a.Value
	}
	assert.Equal(t, "abc=50", attrs[instance.HeadSamplingAppliedAttribute])
}
