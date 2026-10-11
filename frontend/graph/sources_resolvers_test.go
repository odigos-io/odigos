package graph

import (
	"context"
	"testing"

	odigosfake "github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned/fake"
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// A service name that closes the JSON string of a hand-built patch document and appends
// patch operations of its own: the first operation is rewritten into an "add" on
// /metadata/finalizers (duplicate JSON keys, last one wins), which wedges deletion of the Source.
const maliciousOtelServiceName = `evil", "op": "add", "path": "/metadata/finalizers", "value": ["odigos.io/injected"]}, {"op": "replace", "path": "/spec/otelServiceName", "value": "pwned`

func TestUpdateK8sActualSource_OtelServiceNameCannotInjectPatchOperations(t *testing.T) {
	source := &v1alpha1.Source{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-app-source",
			Namespace: "test-ns",
			Labels: map[string]string{
				k8sconsts.WorkloadNameLabel:      "my-app",
				k8sconsts.WorkloadNamespaceLabel: "test-ns",
				k8sconsts.WorkloadKindLabel:      "Deployment",
			},
		},
		Spec: v1alpha1.SourceSpec{
			Workload: k8sconsts.PodWorkload{
				Name:      "my-app",
				Namespace: "test-ns",
				Kind:      k8sconsts.WorkloadKindDeployment,
			},
			OtelServiceName: "original",
		},
	}

	fakeClient := odigosfake.NewSimpleClientset(source)
	previous := kube.DefaultClient
	kube.SetDefaultClient(&kube.Client{OdigosClient: fakeClient.OdigosV1alpha1()})
	t.Cleanup(func() { kube.SetDefaultClient(previous) })

	ctx := context.Background()
	serviceName := maliciousOtelServiceName
	ok, err := (&Resolver{}).Mutation().UpdateK8sActualSource(ctx,
		model.K8sSourceID{Namespace: "test-ns", Name: "my-app", Kind: model.K8sResourceKindDeployment},
		model.PatchSourceRequestInput{OtelServiceName: &serviceName},
	)
	require.NoError(t, err)
	assert.True(t, ok)

	updated, err := fakeClient.OdigosV1alpha1().Sources("test-ns").Get(ctx, "my-app-source", metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, maliciousOtelServiceName, updated.Spec.OtelServiceName,
		"the service name must be stored verbatim as a string")
	assert.Empty(t, updated.Finalizers, "no finalizer may be injected through the service name")
}
