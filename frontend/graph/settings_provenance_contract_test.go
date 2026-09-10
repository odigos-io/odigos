package graph

import (
	"context"
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"
)

// The provenance map is produced in package services, keyed by the YAML config key, and
// consumed in package graph, keyed by the helm value path. Nothing links the two at
// compile time: a key that only one side knows about is a "reconciled from" badge that
// silently reads odigos-configuration no matter what the user changed in the UI.

const settingsProvenanceNamespace = "odigos-settings-provenance-test"

func settingsProvenanceClient(t *testing.T, configs map[string]*common.OdigosConfiguration) client.Client {
	t.Helper()

	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))

	objects := make([]client.Object, 0, len(configs))
	for name, config := range configs {
		raw, err := yaml.Marshal(config)
		require.NoError(t, err)
		objects = append(objects, &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: settingsProvenanceNamespace},
			Data:       map[string]string{consts.OdigosConfigurationFileName: string(raw)},
		})
	}

	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()
}

// Every key the overlay recorder can emit must be looked up by the renderer. Feeding one
// key at a time with a source nothing else uses makes the lookup observable: if the key
// is never read, no entry carries the sentinel.
func TestEveryOverlayProvenanceKeyIsLookedUpByTheEffectiveConfigRenderer(t *testing.T) {
	t.Setenv(consts.CurrentNamespaceEnvVar, settingsProvenanceNamespace)

	effective := fullyPopulatedOdigosConfig()
	c := settingsProvenanceClient(t, map[string]*common.OdigosConfiguration{
		consts.OdigosConfigurationName: effective,
		consts.OdigosLocalUiConfigName: effective,
	})

	provenance, err := services.ComputeProvenance(context.Background(), c, effective)
	require.NoError(t, err)
	require.NotEmpty(t, provenance)

	const sentinel = "sentinel-source"
	for yamlKey, source := range provenance {
		require.Equal(t, consts.OdigosLocalUiConfigName, source,
			"the fixture only carries a local UI overlay, so %q must not be attributed elsewhere", yamlKey)

		t.Run(yamlKey, func(t *testing.T) {
			got, err := EffectiveConfigToModel(effective, map[string]string{yamlKey: sentinel})
			require.NoError(t, err)

			var helmPaths []string
			for _, entry := range got.Provenance {
				if entry.ReconciledFrom == sentinel {
					helmPaths = append(helmPaths, entry.HelmPath)
				}
			}

			assert.NotEmpty(t, helmPaths,
				"services.recordOverlayProvenance records %q but EffectiveConfigToModel never looks it up, "+
					"so the settings screen keeps showing this field as coming from the helm baseline", yamlKey)
		})
	}
}

// The whole round trip a settings page load performs: read the three ConfigMaps, compute
// provenance, render the model. Each badge must name the source that actually set the
// value, with the local UI overlay winning over the remote one and profiles picked up for
// fields no overlay mentions.
func TestSettingsProvenanceEndToEnd(t *testing.T) {
	t.Setenv(consts.CurrentNamespaceEnvVar, settingsProvenanceNamespace)

	base := &common.OdigosConfiguration{
		ClusterName:       "from-helm",
		ImagePrefix:       "registry.example.com",
		RollbackGraceTime: "1m",
	}
	remote := &common.OdigosConfiguration{
		ClusterName:       "from-remote",
		RollbackGraceTime: "2m",
		Rollout:           &common.RolloutConfiguration{MaxConcurrentRollouts: 3},
	}
	local := &common.OdigosConfiguration{
		ClusterName: "from-local",
	}
	effective := &common.OdigosConfiguration{
		ClusterName:           "from-local",
		ImagePrefix:           "registry.example.com",
		RollbackGraceTime:     "2m",
		Rollout:               &common.RolloutConfiguration{MaxConcurrentRollouts: 3},
		AllowConcurrentAgents: effBoolPtr(true),
	}

	c := settingsProvenanceClient(t, map[string]*common.OdigosConfiguration{
		consts.OdigosConfigurationName: base,
		consts.OdigosRemoteConfigName:  remote,
		consts.OdigosLocalUiConfigName: local,
	})

	provenance, err := services.ComputeProvenance(context.Background(), c, effective)
	require.NoError(t, err)

	got, err := EffectiveConfigToModel(effective, provenance)
	require.NoError(t, err)

	byPath := provenanceByHelmPath(got.Provenance)
	assert.Equal(t, consts.OdigosLocalUiConfigName, byPath["clusterName"])
	assert.Equal(t, consts.OdigosRemoteConfigName, byPath["autoRollback.graceTime"])
	assert.Equal(t, consts.OdigosRemoteConfigName, byPath["rollout.maxConcurrentRollouts"])
	assert.Equal(t, "profile", byPath["allowConcurrentAgents.enabled"])
	assert.Equal(t, "odigos-configuration", byPath["imagePrefix"])

	assert.Equal(t, effStrPtr("from-local"), got.ClusterName)
	assert.Equal(t, effStrPtr("2m"), got.AutoRollback.GraceTime)
}
