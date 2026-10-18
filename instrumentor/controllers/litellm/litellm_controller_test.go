package litellm

import (
	"context"
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
)

func boolPtr(b bool) *bool { return &b }

func configMap(ns, name string, cfg *common.OdigosConfiguration) *corev1.ConfigMap {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		panic(err)
	}
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: name},
		Data:       map[string]string{consts.OdigosConfigurationFileName: string(data)},
	}
}

func interrogationConfig(enabled bool, model *common.InterrogationModelConfiguration) *common.OdigosConfiguration {
	return &common.OdigosConfiguration{
		Interrogation: &common.InterrogationConfiguration{Enabled: boolPtr(enabled), Model: model},
	}
}

func liteLLMSecret(ns, upstreamApiKey string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: k8sconsts.OdigosLiteLLMName},
		Data:       map[string][]byte{k8sconsts.OdigosLiteLLMUpstreamApiKeySecretKey: []byte(upstreamApiKey)},
	}
}

func liteLLMDeployment(ns string) *appsv1.Deployment {
	return &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: k8sconsts.OdigosLiteLLMName}}
}

func newReconciler(objs ...client.Object) *liteLLMController {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		panic(err)
	}
	if err := appsv1.AddToScheme(scheme); err != nil {
		panic(err)
	}
	return &liteLLMController{
		Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build(),
		Scheme: scheme,
	}
}

func reconcile(t *testing.T, r *liteLLMController) {
	t.Helper()
	if _, err := r.Reconcile(context.Background(), ctrl.Request{}); err != nil {
		t.Fatalf("reconcile failed: %v", err)
	}
}

func getLiteLLMConfig(t *testing.T, c client.Client, ns string) (string, bool) {
	t.Helper()
	cm := &corev1.ConfigMap{}
	err := c.Get(context.Background(), types.NamespacedName{Namespace: ns, Name: k8sconsts.OdigosLiteLLMName}, cm)
	if apierrors.IsNotFound(err) {
		return "", false
	}
	if err != nil {
		t.Fatalf("unexpected error getting litellm config map: %v", err)
	}
	return cm.Data[k8sconsts.OdigosLiteLLMConfigFileName], true
}

func getConfigHash(t *testing.T, c client.Client, ns string) string {
	t.Helper()
	dep := &appsv1.Deployment{}
	if err := c.Get(context.Background(), types.NamespacedName{Namespace: ns, Name: k8sconsts.OdigosLiteLLMName}, dep); err != nil {
		t.Fatalf("unexpected error getting litellm deployment: %v", err)
	}
	return dep.Spec.Template.Annotations[k8sconsts.OdigosLiteLLMConfigHashAnnotation]
}

func TestRenderLiteLLMConfig(t *testing.T) {
	configYaml, err := renderLiteLLMConfig(&common.InterrogationModelConfiguration{
		Provider: "hosted_vllm",
		Name:     "interrogation",
		ApiBase:  "http://odigos-vllm:8000/v1",
	})
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	got := liteLLMConfig{}
	if err := yaml.Unmarshal([]byte(configYaml), &got); err != nil {
		t.Fatalf("rendered config is not valid yaml: %v", err)
	}
	want := liteLLMConfig{
		ModelList: []liteLLMModel{{
			ModelName: "interrogation",
			LiteLLMParams: liteLLMParams{
				Model:   "hosted_vllm/interrogation",
				ApiKey:  "os.environ/LITELLM_UPSTREAM_API_KEY",
				ApiBase: "http://odigos-vllm:8000/v1",
			},
		}},
		LiteLLMSettings: liteLLMSettings{DropParams: true},
		GeneralSettings: liteLLMGeneralSettings{MasterKey: "os.environ/LITELLM_MASTER_KEY"},
	}
	if len(got.ModelList) != 1 || got.ModelList[0] != want.ModelList[0] ||
		got.LiteLLMSettings != want.LiteLLMSettings || got.GeneralSettings != want.GeneralSettings {
		t.Errorf("rendered config = %+v, want %+v", got, want)
	}
}

func TestRenderLiteLLMConfig_Defaults(t *testing.T) {
	configYaml, err := renderLiteLLMConfig(nil)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	if !strings.Contains(configYaml, "model: openai/gpt-4o-mini") {
		t.Errorf("rendered config does not default to openai/gpt-4o-mini:\n%s", configYaml)
	}
	if strings.Contains(configYaml, "api_base") {
		t.Errorf("rendered config has api_base although none was set:\n%s", configYaml)
	}
}

func TestReconcile_InterrogationDisabledIsNoop(t *testing.T) {
	ns := env.GetCurrentNamespace()
	cfg := interrogationConfig(false, nil)
	r := newReconciler(
		configMap(ns, consts.OdigosConfigurationName, cfg),
		configMap(ns, consts.OdigosEffectiveConfigName, cfg),
		liteLLMDeployment(ns),
	)

	reconcile(t, r)

	if _, found := getLiteLLMConfig(t, r.Client, ns); found {
		t.Errorf("litellm config map was created although interrogation is disabled")
	}
	if hash := getConfigHash(t, r.Client, ns); hash != "" {
		t.Errorf("litellm deployment was annotated although interrogation is disabled: %q", hash)
	}
}

func TestReconcile_RendersConfigAndRollsOnChange(t *testing.T) {
	ns := env.GetCurrentNamespace()
	helmConfig := interrogationConfig(true, &common.InterrogationModelConfiguration{Provider: "openai", Name: "gpt-4o-mini"})
	r := newReconciler(
		configMap(ns, consts.OdigosConfigurationName, helmConfig),
		configMap(ns, consts.OdigosEffectiveConfigName, helmConfig),
		liteLLMSecret(ns, "key-1"),
		liteLLMDeployment(ns),
	)

	reconcile(t, r)

	configYaml, found := getLiteLLMConfig(t, r.Client, ns)
	if !found || !strings.Contains(configYaml, "model: openai/gpt-4o-mini") {
		t.Fatalf("litellm config map not rendered from effective config: %q", configYaml)
	}
	firstHash := getConfigHash(t, r.Client, ns)
	if firstHash == "" {
		t.Fatalf("litellm deployment was not annotated with the config hash")
	}

	reconcile(t, r)
	if hash := getConfigHash(t, r.Client, ns); hash != firstHash {
		t.Errorf("config hash changed without a config change: %q -> %q", firstHash, hash)
	}

	effective := configMap(ns, consts.OdigosEffectiveConfigName, interrogationConfig(true, &common.InterrogationModelConfiguration{
		Provider: "anthropic",
		Name:     "claude-sonnet-4-5",
		ApiBase:  "https://llm.example.com/v1",
	}))
	if err := r.Client.Update(context.Background(), effective); err != nil {
		t.Fatalf("failed to update effective config: %v", err)
	}
	reconcile(t, r)

	configYaml, _ = getLiteLLMConfig(t, r.Client, ns)
	if !strings.Contains(configYaml, "model: anthropic/claude-sonnet-4-5") || !strings.Contains(configYaml, "api_base: https://llm.example.com/v1") {
		t.Errorf("litellm config map not updated from effective config:\n%s", configYaml)
	}
	modelHash := getConfigHash(t, r.Client, ns)
	if modelHash == firstHash {
		t.Errorf("config hash did not change after the model changed")
	}

	if err := r.Client.Update(context.Background(), liteLLMSecret(ns, "key-2")); err != nil {
		t.Fatalf("failed to update litellm secret: %v", err)
	}
	reconcile(t, r)
	if hash := getConfigHash(t, r.Client, ns); hash == modelHash {
		t.Errorf("config hash did not change after the upstream API key changed")
	}
}

func TestReconcile_MissingDeploymentIsSkipped(t *testing.T) {
	ns := env.GetCurrentNamespace()
	cfg := interrogationConfig(true, nil)
	r := newReconciler(
		configMap(ns, consts.OdigosConfigurationName, cfg),
		configMap(ns, consts.OdigosEffectiveConfigName, cfg),
	)

	reconcile(t, r)

	if _, found := getLiteLLMConfig(t, r.Client, ns); found {
		t.Errorf("litellm config map was created without a litellm deployment")
	}
}
