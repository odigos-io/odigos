package recommendations

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/yaml"

	odigosapply "github.com/odigos-io/odigos/api/generated/odigos/applyconfiguration/odigos/v1alpha1"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	recommendationcatalog "github.com/odigos-io/odigos/recommendations"
)

const testNamespace = "odigos-test-ns"

// testContext carries a discarding logger so the code paths that log do not depend on
// a global logger being configured.
func testContext() context.Context {
	return logr.NewContext(context.Background(), logr.Discard())
}

func newFakeClient(t *testing.T, objects ...client.Object) client.WithWatch {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, odigosv1.AddToScheme(scheme))
	return fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(objects...).
		Build()
}

func newRecordingClient(t *testing.T, objects ...client.Object) *recordingClient {
	t.Helper()
	recorder := &recordingClient{}
	recorder.WithWatch = interceptor.NewClient(newFakeClient(t, objects...), interceptor.Funcs{
		Apply: func(ctx context.Context, c client.WithWatch, obj runtime.ApplyConfiguration, opts ...client.ApplyOption) error {
			if rec, ok := obj.(*odigosapply.RecommendationApplyConfiguration); ok {
				applyOpts := &client.ApplyOptions{}
				applyOpts.ApplyOptions(opts)
				recorded := recordedApply{
					name:      ptr.Deref(rec.Name, ""),
					namespace: ptr.Deref(rec.Namespace, ""),
					opts:      *applyOpts,
				}
				if rec.Spec != nil {
					recorded.spec = *rec.Spec
				}
				recorder.applies = append(recorder.applies, recorded)
			}
			return c.Apply(ctx, obj, opts...)
		},
	})
	return recorder
}

// recordedApply is a snapshot of a server side apply as it was sent by the controller. The fake
// client both drops explicit false values from apply configurations and overwrites the
// configuration with the merge result, so the snapshot is taken before the apply is forwarded.
type recordedApply struct {
	name      string
	namespace string
	spec      odigosapply.RecommendationSpecApplyConfiguration
	opts      client.ApplyOptions
}

type recordingClient struct {
	client.WithWatch
	applies []recordedApply
}

// single returns the only recommendation that was applied.
func (r *recordingClient) single(t *testing.T) recordedApply {
	t.Helper()
	require.Len(t, r.applies, 1)
	return r.applies[0]
}

// effectiveConfigMapWithYAML builds the odigos effective config ConfigMap the recommendations
// controller reads, with a raw YAML body.
func effectiveConfigMapWithYAML(ns string, cfgYAML string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      consts.OdigosEffectiveConfigName,
			Namespace: ns,
		},
		Data: map[string]string{
			consts.OdigosConfigurationFileName: cfgYAML,
		},
	}
}

func effectiveConfigMap(t *testing.T, cfg *common.OdigosConfiguration) *corev1.ConfigMap {
	t.Helper()
	cfgYAML, err := yaml.Marshal(cfg)
	require.NoError(t, err)
	return effectiveConfigMapWithYAML(testNamespace, string(cfgYAML))
}

// useTestNamespace points env.GetCurrentNamespace() at the namespace used by these tests.
func useTestNamespace(t *testing.T) {
	t.Helper()
	t.Setenv(consts.CurrentNamespaceEnvVar, testNamespace)
}

func useTier(t *testing.T, tier common.OdigosTier) {
	t.Helper()
	t.Setenv(consts.OdigosTierEnvVarName, string(tier))
}

// loadCatalog loads the recommendation manifests that ship with the odigos deployment.
func loadCatalog(t *testing.T) []recommendationcatalog.Recommendation {
	t.Helper()
	require.NoError(t, recommendationcatalog.Load())
	recs := recommendationcatalog.Get()
	require.NotEmpty(t, recs, "recommendations catalog should not be empty")
	return recs
}
