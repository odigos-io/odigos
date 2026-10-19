package litellm

import (
	"context"
	"strings"
	"testing"
	"time"

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

func boolPtr(b bool) *bool    { return &b }
func strPtr(s string) *string { return &s }

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
		Provider: strPtr("hosted_vllm"),
		Name:     strPtr("interrogation"),
		ApiBase:  strPtr("http://odigos-vllm:8000/v1"),
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
	helmConfig := interrogationConfig(true, &common.InterrogationModelConfiguration{Provider: strPtr("openai"), Name: strPtr("gpt-4o-mini")})
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
		Provider: strPtr("anthropic"),
		Name:     strPtr("claude-sonnet-4-5"),
		ApiBase:  strPtr("https://llm.example.com/v1"),
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

func TestReconcile_SelectedCredentialFeedsRuntimeSecret(t *testing.T) {
	ns := env.GetCurrentNamespace()
	cfg := interrogationConfig(true, &common.InterrogationModelConfiguration{Provider: strPtr("anthropic"), Name: strPtr("claude-sonnet-4-5")})
	cfg.Interrogation.CredentialSecretName = k8sconsts.OdigosLocalUIAICredentialSecretPrefix + "selected"
	selected := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{
		Namespace: ns, Name: cfg.Interrogation.CredentialSecretName,
		Labels: map[string]string{k8sconsts.OdigosLocalUIAIConfigSourceLabel: k8sconsts.OdigosLocalUIAIConfigSourceValue},
	}, Data: map[string][]byte{k8sconsts.OdigosLiteLLMUpstreamApiKeySecretKey: []byte("ui-key")}}
	r := newReconciler(
		configMap(ns, consts.OdigosConfigurationName, cfg),
		configMap(ns, consts.OdigosEffectiveConfigName, cfg),
		liteLLMSecret(ns, "helm-key"), selected, liteLLMDeployment(ns),
	)
	reconcile(t, r)

	var runtime corev1.Secret
	if err := r.Client.Get(context.Background(), types.NamespacedName{Namespace: ns, Name: k8sconsts.OdigosLiteLLMRuntimeSecretName}, &runtime); err != nil {
		t.Fatal(err)
	}
	if got := string(runtime.Data[k8sconsts.OdigosLiteLLMUpstreamApiKeySecretKey]); got != "ui-key" {
		t.Errorf("runtime key = %q, want selected UI key", got)
	}
	configHash := getConfigHash(t, r.Client, ns)
	if configHash == "" || runtime.Annotations[credentialFingerprintAnnotation] != configHash {
		t.Fatalf("config hash not shared by runtime Secret and Deployment")
	}

	cfg.Interrogation.CredentialSecretName = ""
	if err := r.Client.Update(context.Background(), configMap(ns, consts.OdigosEffectiveConfigName, cfg)); err != nil {
		t.Fatal(err)
	}
	reconcile(t, r)
	if err := r.Client.Get(context.Background(), types.NamespacedName{Namespace: ns, Name: k8sconsts.OdigosLiteLLMRuntimeSecretName}, &runtime); err != nil {
		t.Fatal(err)
	}
	if got := string(runtime.Data[k8sconsts.OdigosLiteLLMUpstreamApiKeySecretKey]); got != "helm-key" {
		t.Errorf("runtime key = %q, want Helm key after the selection is cleared", got)
	}
	if got := getConfigHash(t, r.Client, ns); got == configHash {
		t.Error("clearing the selected credential did not roll the proxy")
	}
}

func TestReconcile_EmptyCredentialStillCreatesRuntimeKey(t *testing.T) {
	ns := env.GetCurrentNamespace()
	cfg := interrogationConfig(true, nil)
	r := newReconciler(configMap(ns, consts.OdigosConfigurationName, cfg), configMap(ns, consts.OdigosEffectiveConfigName, cfg), liteLLMSecret(ns, ""), liteLLMDeployment(ns))
	reconcile(t, r)
	var runtime corev1.Secret
	if err := r.Client.Get(context.Background(), types.NamespacedName{Namespace: ns, Name: k8sconsts.OdigosLiteLLMRuntimeSecretName}, &runtime); err != nil {
		t.Fatal(err)
	}
	key, exists := runtime.Data[k8sconsts.OdigosLiteLLMUpstreamApiKeySecretKey]
	if !exists || key == nil {
		t.Fatal("runtime Secret should hold an explicit empty key for proxy bootstrap")
	}
}

func TestReconcile_ProxyConfigMapRetainsHelmAdoptionMetadata(t *testing.T) {
	ns := env.GetCurrentNamespace()
	for _, bootstrap := range []bool{false, true} {
		name := "controller-created"
		if bootstrap {
			name = "bootstrap-adopted"
		}
		t.Run(name, func(t *testing.T) {
			cfg := interrogationConfig(true, nil)
			owner := configMap(ns, consts.OdigosConfigurationName, cfg)
			owner.UID = "configuration-uid"
			owner.Labels = map[string]string{"app.kubernetes.io/managed-by": "Helm"}
			owner.Annotations = map[string]string{
				"meta.helm.sh/release-name":      "odigos",
				"meta.helm.sh/release-namespace": ns,
			}
			objects := []client.Object{owner, configMap(ns, consts.OdigosEffectiveConfigName, cfg), liteLLMSecret(ns, "key"), liteLLMDeployment(ns)}
			if bootstrap {
				objects = append(objects, &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
					Namespace: ns, Name: k8sconsts.OdigosLiteLLMName,
					Labels: map[string]string{"app.kubernetes.io/managed-by": "Helm"},
					Annotations: map[string]string{
						"helm.sh/resource-policy":        "keep",
						"meta.helm.sh/release-name":      "odigos",
						"meta.helm.sh/release-namespace": ns,
					},
				}, Data: map[string]string{k8sconsts.OdigosLiteLLMConfigFileName: "legacy"}})
			}
			r := newReconciler(objects...)
			reconcile(t, r)
			var proxyCM corev1.ConfigMap
			if err := r.Client.Get(context.Background(), types.NamespacedName{Namespace: ns, Name: k8sconsts.OdigosLiteLLMName}, &proxyCM); err != nil {
				t.Fatal(err)
			}
			if proxyCM.Labels["app.kubernetes.io/managed-by"] != "Helm" ||
				proxyCM.Annotations["meta.helm.sh/release-name"] != "odigos" ||
				proxyCM.Annotations["meta.helm.sh/release-namespace"] != ns {
				t.Errorf("proxy ConfigMap has no Helm adoption metadata: labels=%v annotations=%v", proxyCM.Labels, proxyCM.Annotations)
			}
			if bootstrap && proxyCM.Annotations["helm.sh/resource-policy"] != "keep" {
				t.Error("bootstrap keep annotation was removed")
			}
			if len(proxyCM.OwnerReferences) != 1 || proxyCM.OwnerReferences[0].UID != owner.UID {
				t.Errorf("proxy ConfigMap was not owned by odigos-configuration: %+v", proxyCM.OwnerReferences)
			}
		})
	}
}

func TestReconcile_MissingSelectedCredentialKeepsConfigHash(t *testing.T) {
	ns := env.GetCurrentNamespace()
	cfg := interrogationConfig(true, nil)
	r := newReconciler(configMap(ns, consts.OdigosConfigurationName, cfg), configMap(ns, consts.OdigosEffectiveConfigName, cfg), liteLLMSecret(ns, "helm-key"), liteLLMDeployment(ns))
	reconcile(t, r)
	before := getConfigHash(t, r.Client, ns)
	cfg.Interrogation.CredentialSecretName = k8sconsts.OdigosLocalUIAICredentialSecretPrefix + "missing"
	if err := r.Client.Update(context.Background(), configMap(ns, consts.OdigosEffectiveConfigName, cfg)); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Reconcile(context.Background(), ctrl.Request{}); err == nil {
		t.Fatal("missing selected credential must fail")
	}
	if got := getConfigHash(t, r.Client, ns); got != before {
		t.Errorf("config hash changed on failed selection: %q -> %q", before, got)
	}
}

func TestCleanupOldCredentialsRetainsActivePreviousAndFresh(t *testing.T) {
	ns := env.GetCurrentNamespace()
	oldTime := metav1.NewTime(time.Now().Add(-2 * credentialRetention))
	newTime := metav1.NewTime(time.Now())
	newCredential := func(suffix string, created metav1.Time) *corev1.Secret {
		return &corev1.Secret{ObjectMeta: metav1.ObjectMeta{
			Namespace: ns, Name: k8sconsts.OdigosLocalUIAICredentialSecretPrefix + suffix,
			Labels:            map[string]string{k8sconsts.OdigosLocalUIAIConfigSourceLabel: k8sconsts.OdigosLocalUIAIConfigSourceValue},
			CreationTimestamp: created,
		}}
	}
	active := newCredential("active", oldTime)
	previous := newCredential("previous", oldTime)
	fresh := newCredential("fresh", newTime)
	stale := newCredential("stale", oldTime)
	r := newReconciler(active, previous, fresh, stale)
	cfg := &common.InterrogationConfiguration{CredentialSecretName: active.Name, PreviousCredentialSecretName: previous.Name}
	if err := r.cleanupOldCredentials(context.Background(), ns, cfg); err != nil {
		t.Fatal(err)
	}
	for _, secret := range []*corev1.Secret{active, previous, fresh} {
		if err := r.Client.Get(context.Background(), types.NamespacedName{Namespace: ns, Name: secret.Name}, &corev1.Secret{}); err != nil {
			t.Errorf("retained credential %s missing: %v", secret.Name, err)
		}
	}
	if err := r.Client.Get(context.Background(), types.NamespacedName{Namespace: ns, Name: stale.Name}, &corev1.Secret{}); !apierrors.IsNotFound(err) {
		t.Errorf("stale credential should be removed, got %v", err)
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
