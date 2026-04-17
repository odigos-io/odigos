package services

import (
	"context"
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestUpsertLocalUiConfigDoesNotMutateInputCacheObject(t *testing.T) {
	ctx := context.Background()

	const ns = "odigos-system"
	t.Setenv(consts.CurrentNamespaceEnvVar, ns)

	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatalf("failed adding core scheme: %v", err)
	}

	existing := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      consts.OdigosLocalUiConfigName,
			Namespace: ns,
		},
		Data: map[string]string{
			consts.OdigosConfigurationFileName: "clusterName: before\n",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(existing).
		Build()

	if err := upsertLocalUiConfig(ctx, fakeClient, func(cfg *common.OdigosConfiguration) {
		cfg.ClusterName = "after"
	}); err != nil {
		t.Fatalf("upsertLocalUiConfig failed: %v", err)
	}

	// The original object used to seed the fake client must stay unchanged.
	// This guards against mutating cache-owned/shared objects in place.
	got := existing.Data[consts.OdigosConfigurationFileName]
	if got != "clusterName: before\n" {
		t.Fatalf("input object mutated unexpectedly: got %q", got)
	}
}
