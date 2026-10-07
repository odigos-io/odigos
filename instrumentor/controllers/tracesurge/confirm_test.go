package tracesurge

import (
	"context"
	"fmt"
	"testing"
	"time"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	cacheutils "github.com/odigos-io/odigos/k8sutils/pkg/cache"
	instance "github.com/odigos-io/odigos/k8sutils/pkg/instrumentation_instance"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestInstanceTransform(t *testing.T) {
	healthy := true
	ii := &odigosv1.InstrumentationInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name: "payments-a-1", Namespace: "shop", ResourceVersion: "7",
			Labels:          map[string]string{odigosv1.OwnerPodNameLabel: "payments-a"},
			Annotations:     map[string]string{"big": "annotation"},
			OwnerReferences: []metav1.OwnerReference{{Name: "payments-a"}},
		},
		Spec: odigosv1.InstrumentationInstanceSpec{ContainerName: "app"},
		Status: odigosv1.InstrumentationInstanceStatus{
			Healthy:               &healthy,
			Message:               "running",
			IdentifyingAttributes: []odigosv1.Attribute{{Key: "process.pid", Value: "1"}},
			NonIdentifyingAttributes: []odigosv1.Attribute{
				{Key: "process.runtime.version", Value: "17"},
				{Key: instance.HeadSamplingAppliedAttribute, Value: "rule=50"},
				{Key: instance.ConfigErrorAttribute, Value: "bad"},
			},
		},
	}

	out, err := InstanceTransform(ii)
	require.NoError(t, err)
	stripped := out.(*odigosv1.InstrumentationInstance)
	assert.Equal(t, "payments-a-1", stripped.Name)
	assert.Equal(t, "7", stripped.ResourceVersion)
	assert.Equal(t, ii.Labels, stripped.Labels)
	assert.Empty(t, stripped.OwnerReferences)
	assert.Empty(t, stripped.Spec.ContainerName)
	assert.Nil(t, stripped.Status.Healthy)
	assert.Empty(t, stripped.Status.IdentifyingAttributes)
	assert.Equal(t, []odigosv1.Attribute{
		{Key: instance.HeadSamplingAppliedAttribute, Value: "rule=50"},
		{Key: instance.ConfigErrorAttribute, Value: "bad"},
	}, stripped.Status.NonIdentifyingAttributes)
	assert.True(t, cacheutils.IsObjectTransformed(stripped))

	again, err := InstanceTransform(stripped)
	require.NoError(t, err)
	assert.Same(t, stripped, again, "an object already transformed is returned as is")
}

func TestTargetStatusListsFewInstancesProblemsFirst(t *testing.T) {
	ctx := context.Background()
	e, c, _, ruleID := setup(t)
	for i := range 12 {
		applied := ruleID + "=80"
		if i == 3 || i == 7 {
			applied = ruleID + "=1"
		}
		require.NoError(t, c.Create(ctx, instrumentationInstance(payments, fmt.Sprintf("payments-%02d", i), applied)))
	}

	status, err := e.targetStatus(ctx, ruleID, odigosv1.TraceSurgeTarget{Workload: payments}, 80, nil, time.Now())
	require.NoError(t, err)
	assert.Equal(t, 12, status.Total)
	assert.Equal(t, 10, status.Confirmed)
	require.Len(t, status.Instances, maxListedInstances)
	assert.Equal(t, []string{"payments-03-1", "payments-07-1"}, []string{status.Instances[0].Name, status.Instances[1].Name}, "the ones that did not confirm come first")
	assert.Equal(t, instanceUnknown, status.Instances[0].State)
	assert.Equal(t, instanceConfirmed, status.Instances[2].State)
}
