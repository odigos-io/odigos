package services

import (
	"context"
	"encoding/json"
	"testing"

	odigosfake "github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned/fake"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stesting "k8s.io/client-go/testing"
)

// setupFakeSourceClient points kube.DefaultClient at a fake odigos clientset holding source,
// and returns a pointer to the raw patch document sent for the Source resource.
func setupFakeSourceClient(t *testing.T, source *v1alpha1.Source) *[]byte {
	t.Helper()

	fakeClient := odigosfake.NewSimpleClientset(source)
	var capturedPatch []byte
	fakeClient.PrependReactor("patch", "sources", func(action k8stesting.Action) (bool, runtime.Object, error) {
		capturedPatch = action.(k8stesting.PatchAction).GetPatch()
		return false, nil, nil
	})

	previous := kube.DefaultClient
	kube.SetDefaultClient(&kube.Client{OdigosClient: fakeClient.OdigosV1alpha1()})
	t.Cleanup(func() { kube.SetDefaultClient(previous) })

	return &capturedPatch
}

func TestUpdateSourceCRDSpec_OtelServiceNameWithQuotesIsNotInterpretedAsPatchOps(t *testing.T) {
	// A service name that closes the JSON string and appends patch operations of its own.
	// If the patch document is built by string concatenation, the first operation is
	// rewritten into an "add" on /metadata/finalizers, which wedges deletion of the Source.
	maliciousName := `evil", "op": "add", "path": "/metadata/finalizers", "value": ["odigos.io/injected"]}, {"op": "replace", "path": "/spec/otelServiceName", "value": "pwned`

	source := &v1alpha1.Source{
		ObjectMeta: metav1.ObjectMeta{Name: "my-app-source", Namespace: "test-ns"},
		Spec:       v1alpha1.SourceSpec{OtelServiceName: "original"},
	}
	capturedPatch := setupFakeSourceClient(t, source)

	updated, err := UpdateSourceCRDSpec(context.Background(), "test-ns", "my-app-source", "otelServiceName", maliciousName)
	require.NoError(t, err)

	t.Logf("patch document sent to the API server: %s", string(*capturedPatch))

	var ops []map[string]any
	require.NoError(t, json.Unmarshal(*capturedPatch, &ops), "patch document must be valid JSON")
	require.Len(t, ops, 1, "patch document must contain exactly one operation")
	assert.Equal(t, "replace", ops[0]["op"])
	assert.Equal(t, "/spec/otelServiceName", ops[0]["path"])
	assert.Equal(t, maliciousName, ops[0]["value"], "the service name must be carried as a JSON string value")

	assert.Equal(t, maliciousName, updated.Spec.OtelServiceName)
	assert.Empty(t, updated.Finalizers, "no finalizer may be injected through the service name")
}

func TestUpdateSourceCRDSpec_BoolValue(t *testing.T) {
	source := &v1alpha1.Source{
		ObjectMeta: metav1.ObjectMeta{Name: "my-app-source", Namespace: "test-ns"},
		Spec:       v1alpha1.SourceSpec{DisableInstrumentation: false},
	}
	setupFakeSourceClient(t, source)

	updated, err := UpdateSourceCRDSpec(context.Background(), "test-ns", "my-app-source", "disableInstrumentation", true)
	require.NoError(t, err)
	assert.True(t, updated.Spec.DisableInstrumentation)
}
