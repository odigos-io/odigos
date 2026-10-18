package litellm

import (
	"context"
	"encoding/json"
	"errors"
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

type failingRuntimeSecretClient struct {
	client.Client
	seenApplying *bool
}

func (c *failingRuntimeSecretClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	if obj.GetName() == k8sconsts.OdigosLiteLLMRuntimeSecretName {
		var statusCM corev1.ConfigMap
		if err := c.Client.Get(ctx, types.NamespacedName{Namespace: obj.GetNamespace(), Name: k8sconsts.OdigosInterrogationStatusConfigMapName}, &statusCM); err == nil {
			var status proxyStatus
			if json.Unmarshal([]byte(statusCM.Data["proxy"]), &status) == nil && status.State == "APPLYING" {
				*c.seenApplying = true
			}
		}
		return errors.New("secret body contained sk-private")
	}
	return c.Client.Create(ctx, obj, opts...)
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

func getRevision(t *testing.T, c client.Client, ns string) string {
	t.Helper()
	dep := &appsv1.Deployment{}
	if err := c.Get(context.Background(), types.NamespacedName{Namespace: ns, Name: k8sconsts.OdigosLiteLLMName}, dep); err != nil {
		t.Fatalf("unexpected error getting litellm deployment: %v", err)
	}
	return dep.Spec.Template.Annotations[k8sconsts.OdigosAISettingsRevisionAnnotation]
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
	if hash := getRevision(t, r.Client, ns); hash != "" {
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
	firstHash := getRevision(t, r.Client, ns)
	if firstHash == "" {
		t.Fatalf("litellm deployment was not annotated with the config hash")
	}

	reconcile(t, r)
	if hash := getRevision(t, r.Client, ns); hash != firstHash {
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
	modelHash := getRevision(t, r.Client, ns)
	if modelHash == firstHash {
		t.Errorf("config hash did not change after the model changed")
	}

	if err := r.Client.Update(context.Background(), liteLLMSecret(ns, "key-2")); err != nil {
		t.Fatalf("failed to update litellm secret: %v", err)
	}
	reconcile(t, r)
	if hash := getRevision(t, r.Client, ns); hash == modelHash {
		t.Errorf("config hash did not change after the upstream API key changed")
	}
}

func TestReconcile_SelectedCredentialWhileAnalysisPaused(t *testing.T) {
	ns := env.GetCurrentNamespace()
	cfg := interrogationConfig(true, &common.InterrogationModelConfiguration{Provider: strPtr("anthropic"), Name: strPtr("claude-sonnet-4-5")})
	cfg.Interrogation.AnalysisEnabled = boolPtr(false)
	cfg.Interrogation.CredentialSecretName = k8sconsts.OdigosLocalUIAICredentialSecretPrefix + "selected"
	cfg.Interrogation.UpdateID = "update-1"
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
	revision := getRevision(t, r.Client, ns)
	if revision == "" || runtime.Annotations[k8sconsts.OdigosAISettingsRevisionAnnotation] != revision {
		t.Fatalf("revision not shared by runtime Secret and Deployment")
	}
	var proxyCM corev1.ConfigMap
	if err := r.Client.Get(context.Background(), types.NamespacedName{Namespace: ns, Name: k8sconsts.OdigosLiteLLMName}, &proxyCM); err != nil {
		t.Fatal(err)
	}
	if proxyCM.Annotations[k8sconsts.OdigosAISettingsRevisionAnnotation] != revision {
		t.Error("proxy ConfigMap does not carry selected revision")
	}
	statusName := types.NamespacedName{Namespace: ns, Name: k8sconsts.OdigosInterrogationStatusConfigMapName}
	var statusCM corev1.ConfigMap
	if err := r.Client.Get(context.Background(), statusName, &statusCM); err != nil {
		t.Fatal(err)
	}
	var status proxyStatus
	if err := json.Unmarshal([]byte(statusCM.Data["proxy"]), &status); err != nil {
		t.Fatal(err)
	}
	if status.State != "APPLYING" || status.Revision != revision || status.UpdateID != "update-1" ||
		!status.APIKeyConfigured || status.APIKeySource != "LOCAL_UI" {
		t.Errorf("proxy status = %+v, want APPLYING for selected update", status)
	}
	statusCM.Data["analysis"] = `{"state":"PAUSED"}`
	if err := r.Client.Update(context.Background(), &statusCM); err != nil {
		t.Fatal(err)
	}
	var dep appsv1.Deployment
	if err := r.Client.Get(context.Background(), types.NamespacedName{Namespace: ns, Name: k8sconsts.OdigosLiteLLMName}, &dep); err != nil {
		t.Fatal(err)
	}
	dep.Status.ObservedGeneration = dep.Generation
	dep.Status.Replicas = 1
	dep.Status.UpdatedReplicas = 1
	dep.Status.ReadyReplicas = 1
	dep.Status.AvailableReplicas = 1
	if err := r.Client.Status().Update(context.Background(), &dep); err != nil {
		t.Fatal(err)
	}
	reconcile(t, r)
	if err := r.Client.Get(context.Background(), statusName, &statusCM); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(statusCM.Data["proxy"]), &status); err != nil {
		t.Fatal(err)
	}
	if status.State != "READY" || status.Revision != revision || statusCM.Data["analysis"] != `{"state":"PAUSED"}` {
		t.Errorf("ready status did not preserve analysis writer: %+v %+v", status, statusCM.Data)
	}
}

func TestReconcile_AdoptsStatusCreatedByAnalysisWriter(t *testing.T) {
	ns := env.GetCurrentNamespace()
	cfg := interrogationConfig(true, nil)
	owner := configMap(ns, consts.OdigosConfigurationName, cfg)
	owner.UID = "configuration-uid"
	statusCM := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
		Namespace: ns, Name: k8sconsts.OdigosInterrogationStatusConfigMapName,
	}, Data: map[string]string{"analysis": `{"state":"PAUSED"}`}}
	r := newReconciler(owner, configMap(ns, consts.OdigosEffectiveConfigName, cfg), liteLLMSecret(ns, "helm-key"), liteLLMDeployment(ns), statusCM)
	reconcile(t, r)
	if err := r.Client.Get(context.Background(), types.NamespacedName{Namespace: ns, Name: statusCM.Name}, statusCM); err != nil {
		t.Fatal(err)
	}
	if statusCM.Data["analysis"] != `{"state":"PAUSED"}` {
		t.Fatal("status update removed analysis writer data")
	}
	if len(statusCM.OwnerReferences) != 1 || statusCM.OwnerReferences[0].UID != owner.UID {
		t.Errorf("status ConfigMap was not adopted: %+v", statusCM.OwnerReferences)
	}
	var status proxyStatus
	if err := json.Unmarshal([]byte(statusCM.Data["proxy"]), &status); err != nil {
		t.Fatal(err)
	}
	if status.APIKeySource != "HELM" || !status.APIKeyConfigured {
		t.Errorf("proxy status did not identify Helm credential: %+v", status)
	}
}

func TestReconcile_UpdatesCredentialSourceWithoutProxyRollout(t *testing.T) {
	ns := env.GetCurrentNamespace()
	cfg := interrogationConfig(true, nil)
	r := newReconciler(configMap(ns, consts.OdigosConfigurationName, cfg), configMap(ns, consts.OdigosEffectiveConfigName, cfg), liteLLMSecret(ns, "same-key"), liteLLMDeployment(ns))
	reconcile(t, r)
	revision := getRevision(t, r.Client, ns)
	selectedName := k8sconsts.OdigosLocalUIAICredentialSecretPrefix + "same-key"
	selected := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{
		Namespace: ns, Name: selectedName,
		Labels: map[string]string{k8sconsts.OdigosLocalUIAIConfigSourceLabel: k8sconsts.OdigosLocalUIAIConfigSourceValue},
	}, Data: map[string][]byte{k8sconsts.OdigosLiteLLMUpstreamApiKeySecretKey: []byte("same-key")}}
	if err := r.Client.Create(context.Background(), selected); err != nil {
		t.Fatal(err)
	}
	cfg.Interrogation.CredentialSecretName = selectedName
	if err := r.Client.Update(context.Background(), configMap(ns, consts.OdigosEffectiveConfigName, cfg)); err != nil {
		t.Fatal(err)
	}
	reconcile(t, r)
	if got := getRevision(t, r.Client, ns); got != revision {
		t.Errorf("unchanged key triggered a rollout: %q -> %q", revision, got)
	}
	var statusCM corev1.ConfigMap
	if err := r.Client.Get(context.Background(), types.NamespacedName{Namespace: ns, Name: k8sconsts.OdigosInterrogationStatusConfigMapName}, &statusCM); err != nil {
		t.Fatal(err)
	}
	var status proxyStatus
	if err := json.Unmarshal([]byte(statusCM.Data["proxy"]), &status); err != nil {
		t.Fatal(err)
	}
	if status.APIKeySource != "LOCAL_UI" || !status.APIKeyConfigured {
		t.Errorf("credential source was not refreshed: %+v", status)
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
	var statusCM corev1.ConfigMap
	if err := r.Client.Get(context.Background(), types.NamespacedName{Namespace: ns, Name: k8sconsts.OdigosInterrogationStatusConfigMapName}, &statusCM); err != nil {
		t.Fatal(err)
	}
	var status proxyStatus
	if err := json.Unmarshal([]byte(statusCM.Data["proxy"]), &status); err != nil {
		t.Fatal(err)
	}
	if status.APIKeyConfigured || status.APIKeySource != "NONE" {
		t.Errorf("empty upstream key status = %+v", status)
	}
}

func TestReconcile_RecreatesDeletedStatusConfigMap(t *testing.T) {
	ns := env.GetCurrentNamespace()
	cfg := interrogationConfig(true, nil)
	r := newReconciler(configMap(ns, consts.OdigosConfigurationName, cfg), configMap(ns, consts.OdigosEffectiveConfigName, cfg), liteLLMSecret(ns, "helm-key"), liteLLMDeployment(ns))
	reconcile(t, r)
	statusName := types.NamespacedName{Namespace: ns, Name: k8sconsts.OdigosInterrogationStatusConfigMapName}
	var statusCM corev1.ConfigMap
	if err := r.Client.Get(context.Background(), statusName, &statusCM); err != nil {
		t.Fatal(err)
	}
	if err := r.Client.Delete(context.Background(), &statusCM); err != nil {
		t.Fatal(err)
	}
	reconcile(t, r)
	if err := r.Client.Get(context.Background(), statusName, &statusCM); err != nil {
		t.Fatalf("proxy status was not recreated: %v", err)
	}
	if statusCM.Data["proxy"] == "" {
		t.Fatal("recreated status ConfigMap has no proxy data")
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

func TestDeploymentRolledOut_RejectsOldSurgeReplica(t *testing.T) {
	wanted := int32(1)
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Generation: 3},
		Spec:       appsv1.DeploymentSpec{Replicas: &wanted},
		Status: appsv1.DeploymentStatus{
			ObservedGeneration: 3, Replicas: 2, UpdatedReplicas: 1,
			ReadyReplicas: 2, AvailableReplicas: 2,
		},
	}
	if deploymentRolledOut(dep) {
		t.Fatal("old surge replica is still serving")
	}
	dep.Status.Replicas = 1
	dep.Status.ReadyReplicas = 1
	dep.Status.AvailableReplicas = 1
	if !deploymentRolledOut(dep) {
		t.Fatal("single updated ready replica should be rolled out")
	}
}

func TestReconcile_MissingSelectedCredentialKeepsRuntimeRevision(t *testing.T) {
	ns := env.GetCurrentNamespace()
	cfg := interrogationConfig(true, nil)
	r := newReconciler(configMap(ns, consts.OdigosConfigurationName, cfg), configMap(ns, consts.OdigosEffectiveConfigName, cfg), liteLLMSecret(ns, "helm-key"), liteLLMDeployment(ns))
	reconcile(t, r)
	before := getRevision(t, r.Client, ns)
	cfg.Interrogation.CredentialSecretName = k8sconsts.OdigosLocalUIAICredentialSecretPrefix + "missing"
	cfg.Interrogation.UpdateID = "update-2"
	if err := r.Client.Update(context.Background(), configMap(ns, consts.OdigosEffectiveConfigName, cfg)); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Reconcile(context.Background(), ctrl.Request{}); err == nil {
		t.Fatal("missing selected credential must fail")
	}
	if got := getRevision(t, r.Client, ns); got != before {
		t.Errorf("revision changed on failed selection: %q -> %q", before, got)
	}
	var statusCM corev1.ConfigMap
	if err := r.Client.Get(context.Background(), types.NamespacedName{Namespace: ns, Name: k8sconsts.OdigosInterrogationStatusConfigMapName}, &statusCM); err != nil {
		t.Fatal(err)
	}
	var status proxyStatus
	if err := json.Unmarshal([]byte(statusCM.Data["proxy"]), &status); err != nil {
		t.Fatal(err)
	}
	if status.State != "ERROR" || status.UpdateID != "update-2" {
		t.Errorf("status = %+v, want selected update error", status)
	}
}

func TestReconcile_ReportsSanitizedFailureAfterApplying(t *testing.T) {
	ns := env.GetCurrentNamespace()
	cfg := interrogationConfig(true, nil)
	cfg.Interrogation.UpdateID = "update-error"
	r := newReconciler(configMap(ns, consts.OdigosConfigurationName, cfg), configMap(ns, consts.OdigosEffectiveConfigName, cfg), liteLLMSecret(ns, "sk-private"), liteLLMDeployment(ns))
	seenApplying := false
	r.Client = &failingRuntimeSecretClient{Client: r.Client, seenApplying: &seenApplying}
	if _, err := r.Reconcile(context.Background(), ctrl.Request{}); err == nil {
		t.Fatal("runtime Secret create should fail")
	}
	if !seenApplying {
		t.Error("APPLYING was not published before mutating the runtime Secret")
	}
	var statusCM corev1.ConfigMap
	if err := r.Client.Get(context.Background(), types.NamespacedName{Namespace: ns, Name: k8sconsts.OdigosInterrogationStatusConfigMapName}, &statusCM); err != nil {
		t.Fatal(err)
	}
	var status proxyStatus
	if err := json.Unmarshal([]byte(statusCM.Data["proxy"]), &status); err != nil {
		t.Fatal(err)
	}
	if status.State != "ERROR" || status.Error != "apply runtime credential" || status.ModelHash != common.InterrogationModelHash(nil) || strings.Contains(statusCM.Data["proxy"], "sk-private") {
		t.Errorf("unsafe or incomplete error status: %+v", status)
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

func TestReconcile_MissingDeploymentReportsError(t *testing.T) {
	ns := env.GetCurrentNamespace()
	cfg := interrogationConfig(true, nil)
	r := newReconciler(
		configMap(ns, consts.OdigosConfigurationName, cfg),
		configMap(ns, consts.OdigosEffectiveConfigName, cfg),
	)

	if _, err := r.Reconcile(context.Background(), ctrl.Request{}); err == nil {
		t.Fatal("missing proxy Deployment should be reported")
	}

	if _, found := getLiteLLMConfig(t, r.Client, ns); found {
		t.Errorf("litellm config map was created without a litellm deployment")
	}
	var statusCM corev1.ConfigMap
	if err := r.Client.Get(context.Background(), types.NamespacedName{Namespace: ns, Name: k8sconsts.OdigosInterrogationStatusConfigMapName}, &statusCM); err != nil {
		t.Fatal(err)
	}
	var status proxyStatus
	if err := json.Unmarshal([]byte(statusCM.Data["proxy"]), &status); err != nil {
		t.Fatal(err)
	}
	if status.State != "ERROR" || status.Error != "proxy Deployment is unavailable" {
		t.Errorf("status = %+v", status)
	}
}
