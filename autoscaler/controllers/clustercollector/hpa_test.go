package clustercollector

import (
	"context"
	"strings"
	"testing"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	commonconfig "github.com/odigos-io/odigos/autoscaler/controllers/common"
	controllerconfig "github.com/odigos-io/odigos/autoscaler/controllers/controller_config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/version"
	apiregv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func hpaTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	if err := odigosv1.AddToScheme(s); err != nil {
		t.Fatalf("failed to add odigos scheme: %v", err)
	}
	if err := apiregv1.AddToScheme(s); err != nil {
		t.Fatalf("failed to add apiregistration scheme: %v", err)
	}
	return s
}

func hpaTestGateway() *odigosv1.CollectorsGroup {
	minReplicas := 1
	maxReplicas := 5
	return &odigosv1.CollectorsGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "odigos-gateway",
			Namespace: "odigos-system",
		},
		Spec: odigosv1.CollectorsGroupSpec{
			Role: odigosv1.CollectorsGroupRoleClusterGateway,
			ResourcesSettings: odigosv1.CollectorsGroupResourcesSettings{
				MinReplicas:        &minReplicas,
				MaxReplicas:        &maxReplicas,
				GomemlimitMiB:      400,
				CpuLimitMillicores: 1000,
			},
		},
	}
}

// withControllerConfigK8sVersion swaps the process wide controller config for the duration of a
// single test. The envtest suite in this package populates the same global.
func withControllerConfigK8sVersion(t *testing.T, v *version.Version) {
	t.Helper()
	previous := commonconfig.ControllerConfig
	t.Cleanup(func() { commonconfig.ControllerConfig = previous })
	commonconfig.ControllerConfig = &controllerconfig.ControllerConfig{
		K8sVersion:     v,
		CollectorImage: "otelcol",
	}
}

// An autoscaler that started without successful feature detection carries a nil K8sVersion.
// syncHPA compares that version against three thresholds, and the comparison dereferences the
// receiver, so a nil version used to panic the entire cluster collector reconcile.
func TestSyncHPA_NilKubernetesVersionReturnsErrorInsteadOfPanicking(t *testing.T) {
	withControllerConfigK8sVersion(t, nil)

	scheme := hpaTestScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).Build()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("syncHPA panicked on a nil kubernetes version: %v", r)
		}
	}()

	err := syncHPA(hpaTestGateway(), context.Background(), c, scheme)
	if err == nil {
		t.Fatal("expected syncHPA to fail when the kubernetes version was never detected")
	}
	if !strings.Contains(err.Error(), "kubernetes version") {
		t.Fatalf("expected an error naming the undetected kubernetes version, got: %v", err)
	}
}

// A detected version must still reach the version switch and be applied. This guards against the
// nil check above rejecting legitimate versions.
func TestSyncHPA_DetectedKubernetesVersionIsAccepted(t *testing.T) {
	for _, tc := range []string{"1.22.0", "1.24.0", "1.30.0"} {
		t.Run(tc, func(t *testing.T) {
			withControllerConfigK8sVersion(t, version.MustParse(tc))

			scheme := hpaTestScheme(t)
			c := fake.NewClientBuilder().WithScheme(scheme).Build()

			err := syncHPA(hpaTestGateway(), context.Background(), c, scheme)
			if err != nil && strings.Contains(err.Error(), "kubernetes version was not detected") {
				t.Fatalf("version %s was rejected as undetected", tc)
			}
		})
	}
}
