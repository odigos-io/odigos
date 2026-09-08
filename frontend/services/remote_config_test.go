package services

import (
	"context"
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/yaml"
)

func remoteConfigTestClient(t *testing.T, remoteConfigYaml string) {
	t.Helper()
	ns := env.GetCurrentNamespace()

	objs := []runtime.Object{&v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: consts.OdigosConfigurationName, Namespace: ns, UID: "owner-uid"},
	}}
	if remoteConfigYaml != "" {
		objs = append(objs, &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      consts.OdigosRemoteConfigName,
				Namespace: ns,
				Labels:    map[string]string{k8sconsts.OdigosSystemConfigLabelKey: "remote"},
			},
			Data: map[string]string{consts.OdigosConfigurationFileName: remoteConfigYaml},
		})
	}

	kube.SetDefaultClient(&kube.Client{Interface: k8sfake.NewSimpleClientset(objs...)})
}

func readRemoteConfig(t *testing.T) *common.OdigosConfiguration {
	t.Helper()
	cm, err := kube.DefaultClient.CoreV1().ConfigMaps(env.GetCurrentNamespace()).
		Get(context.Background(), consts.OdigosRemoteConfigName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get remote config: %v", err)
	}
	cfg := &common.OdigosConfiguration{}
	if err := yaml.Unmarshal([]byte(cm.Data[consts.OdigosConfigurationFileName]), cfg); err != nil {
		t.Fatalf("parse remote config: %v", err)
	}
	return cfg
}

// TestUpdateRemoteConfig_PreservesUnrelatedFields covers the odigos-remote-config
// ConfigMap having more than one writer: the central backend / updateRemoteConfig
// mutation owns `rollout`, while configureProfilingCache owns `profiling.ui`.
// A writer that only manages a subset of the document must not drop the rest,
// because the scheduler merges the whole document over the helm baseline - so
// losing `rollout.automaticRolloutDisabled: true` silently re-enables automatic
// rollout and restarts every workload with a pending agent config change.
func TestUpdateRemoteConfig_PreservesUnrelatedFields(t *testing.T) {
	existing := `
clusterName: prod-us-east
ignoredNamespaces:
  - kube-system
  - payments
rollout:
  automaticRolloutDisabled: true
  maxConcurrentRollouts: 3
`
	remoteConfigTestClient(t, existing)

	err := UpdateRemoteConfig(context.Background(), func(cfg *common.OdigosConfiguration) {
		if cfg.Profiling == nil {
			cfg.Profiling = &common.ProfilingConfiguration{}
		}
		cfg.Profiling.Ui = &common.ProfilingUiConfiguration{MaxSlots: 50}
	})
	if err != nil {
		t.Fatalf("UpdateRemoteConfig: %v", err)
	}

	cfg := readRemoteConfig(t)

	if cfg.Profiling == nil || cfg.Profiling.Ui == nil || cfg.Profiling.Ui.MaxSlots != 50 {
		t.Fatalf("profiling.ui was not persisted: %+v", cfg.Profiling)
	}
	if cfg.Rollout == nil {
		t.Fatal("rollout block was dropped: automatic rollout silently re-enabled")
	}
	if cfg.Rollout.AutomaticRolloutDisabled == nil || !*cfg.Rollout.AutomaticRolloutDisabled {
		t.Fatalf("rollout.automaticRolloutDisabled was dropped: %+v", cfg.Rollout)
	}
	if cfg.Rollout.MaxConcurrentRollouts != 3 {
		t.Fatalf("rollout.maxConcurrentRollouts was dropped: %+v", cfg.Rollout)
	}
	if len(cfg.IgnoredNamespaces) != 2 {
		t.Fatalf("ignoredNamespaces was dropped: %+v", cfg.IgnoredNamespaces)
	}
	if cfg.ClusterName != "prod-us-east" {
		t.Fatalf("clusterName was dropped: %q", cfg.ClusterName)
	}
}

// TestUpdateRemoteConfig_CreatesWhenMissing keeps the create path (with the owner
// reference used for helm-uninstall GC) working when no remote config exists yet.
func TestUpdateRemoteConfig_CreatesWhenMissing(t *testing.T) {
	remoteConfigTestClient(t, "")

	err := UpdateRemoteConfig(context.Background(), func(cfg *common.OdigosConfiguration) {
		cfg.Rollout = &common.RolloutConfiguration{MaxConcurrentRollouts: 5}
	})
	if err != nil {
		t.Fatalf("UpdateRemoteConfig: %v", err)
	}

	cm, err := kube.DefaultClient.CoreV1().ConfigMaps(env.GetCurrentNamespace()).
		Get(context.Background(), consts.OdigosRemoteConfigName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get remote config: %v", err)
	}
	if len(cm.OwnerReferences) != 1 || cm.OwnerReferences[0].Name != consts.OdigosConfigurationName {
		t.Fatalf("owner reference missing: %+v", cm.OwnerReferences)
	}
	if cfg := readRemoteConfig(t); cfg.Rollout == nil || cfg.Rollout.MaxConcurrentRollouts != 5 {
		t.Fatalf("rollout not persisted on create: %+v", cfg.Rollout)
	}
}
