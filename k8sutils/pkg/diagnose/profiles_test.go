package diagnose

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/odigos-io/odigos/api/k8sconsts"
)

func TestRequestedStagesIncludesSortedProfilesAndSourceWorkloads(t *testing.T) {
	stages := RequestedStages(Options{
		IncludeCRDs:            true,
		IncludeProfiles:        true,
		IncludeMetrics:         true,
		IncludeConfigMaps:      true,
		IncludeSourceWorkloads: true,
	})
	require.Equal(t, []Stage{
		StageWorkloads,
		StageCRDs,
		"profiles/data-collection",
		"profiles/gateway",
		"profiles/odiglet",
		"profiles/ui",
		StageMetrics,
		StageConfigMaps,
		StageSourceWorkloads,
	}, stages)
}

func TestProfileServiceNamesStableOrder(t *testing.T) {
	first := ProfileServiceNames()
	second := ProfileServiceNames()
	require.Equal(t, first, second)
	require.Equal(t, []string{"data-collection", "gateway", "odiglet", "ui"}, first)
}

func TestFetchServiceProfilesCanceledParentDoesNotFailList(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	cancel()

	ctx, stop := CollectionContext(parent)
	defer stop()

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "odigos-ui-abc",
			Namespace: "odigos-system",
			Labels: map[string]string{
				"app": k8sconsts.UIAppLabelValue,
			},
		},
	}
	client := fake.NewSimpleClientset(pod)

	err := FetchServiceProfiles(ctx, client, NewDryRunBuilder(), "/tmp/profiles", "odigos-system", "ui")
	require.NoError(t, err)
}

func TestFetchServiceProfilesUnknownService(t *testing.T) {
	err := FetchServiceProfiles(context.Background(), fake.NewSimpleClientset(), NewDryRunBuilder(), "/tmp", "ns", "not-a-service")
	require.ErrorContains(t, err, "unknown profiling service")
}
