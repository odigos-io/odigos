package litellm

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	commonlogger "github.com/odigos-io/odigos/common/logger"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"github.com/odigos-io/odigos/k8sutils/pkg/utils"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"
)

const (
	defaultProvider                 = "openai"
	defaultModel                    = "gpt-4o-mini"
	credentialFingerprintAnnotation = "odigos.io/ai-settings-fingerprint"
	credentialRetention             = time.Hour
	// modelAlias is the model name the insights service requests from LiteLLM.
	modelAlias = "interrogation"
)

type proxyStatus struct {
	UpdateID         string `json:"updateId"`
	Revision         string `json:"revision"`
	ModelHash        string `json:"modelHash"`
	APIKeyConfigured bool   `json:"apiKeyConfigured"`
	APIKeySource     string `json:"apiKeySource"`
	State            string `json:"state"`
	Error            string `json:"error"`
	ObservedAt       string `json:"observedAt"`
}

type liteLLMConfig struct {
	ModelList       []liteLLMModel         `json:"model_list"`
	LiteLLMSettings liteLLMSettings        `json:"litellm_settings"`
	GeneralSettings liteLLMGeneralSettings `json:"general_settings"`
}

type liteLLMModel struct {
	ModelName     string        `json:"model_name"`
	LiteLLMParams liteLLMParams `json:"litellm_params"`
}

type liteLLMParams struct {
	Model   string `json:"model"`
	ApiKey  string `json:"api_key"`
	ApiBase string `json:"api_base,omitempty"`
}

type liteLLMSettings struct {
	DropParams bool `json:"drop_params"`
}

type liteLLMGeneralSettings struct {
	MasterKey string `json:"master_key"`
}

// liteLLMController renders the odigos-litellm proxy config from the interrogation model in
// effective-config, and rolls the proxy when the config or its upstream API key changes.
type liteLLMController struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *liteLLMController) Reconcile(ctx context.Context, _ ctrl.Request) (ctrl.Result, error) {
	logger := commonlogger.FromContext(ctx)
	odigosNs := env.GetCurrentNamespace()

	odigosConfiguration, err := utils.GetCurrentOdigosConfiguration(ctx, r.Client)
	if err != nil {
		return utils.K8SNoEffectiveConfigErrorHandler(err)
	}
	if !odigosConfiguration.InterrogationEnabled() {
		return ctrl.Result{}, nil
	}
	cfg := odigosConfiguration.Interrogation
	modelHash := common.InterrogationModelHash(cfg.Model)
	apiKeySource := "NONE"
	apiKeyConfigured := false
	reportFailure := func(stage, revision string, cause error) (ctrl.Result, error) {
		// Kubernetes errors may include object bodies, so the status contains only
		// a fixed stage description. The original error remains in controller logs.
		if statusErr := r.writeProxyStatus(ctx, odigosNs, cfg.UpdateID, revision, modelHash, apiKeyConfigured, apiKeySource, "ERROR", stage); statusErr != nil {
			return ctrl.Result{}, fmt.Errorf("%s: %w; write proxy status: %v", stage, cause, statusErr)
		}
		return ctrl.Result{}, fmt.Errorf("%s: %w", stage, cause)
	}

	// LiteLLM is only deployed with an enterprise token, so there is nothing to configure without it.
	deployment := appsv1.Deployment{}
	err = r.Client.Get(ctx, types.NamespacedName{Namespace: odigosNs, Name: k8sconsts.OdigosLiteLLMName}, &deployment)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return reportFailure("proxy Deployment is unavailable", "", err)
		}
		return reportFailure("read proxy Deployment", "", err)
	}

	configYaml, err := renderLiteLLMConfig(cfg.Model)
	if err != nil {
		return reportFailure("invalid model configuration", "", err)
	}

	upstreamApiKey, err := r.upstreamAPIKey(ctx, odigosNs, cfg)
	if err != nil {
		return reportFailure("read selected credential", "", err)
	}
	apiKeyConfigured = len(upstreamApiKey) > 0
	if apiKeyConfigured {
		if cfg.CredentialSecretName != "" {
			apiKeySource = "LOCAL_UI"
		} else {
			apiKeySource = "HELM"
		}
	}

	configHash := sha256.Sum256(append(append([]byte(configYaml), 0), upstreamApiKey...))
	fingerprint := fmt.Sprintf("%x", configHash)
	revision, changed, err := r.selectRevision(ctx, odigosNs, upstreamApiKey, fingerprint, configYaml)
	if err != nil {
		return reportFailure("read runtime credential", "", err)
	}
	if changed || deployment.Spec.Template.Annotations[k8sconsts.OdigosAISettingsRevisionAnnotation] != revision {
		if err := r.writeProxyStatus(ctx, odigosNs, cfg.UpdateID, revision, modelHash, apiKeyConfigured, apiKeySource, "APPLYING", ""); err != nil {
			return ctrl.Result{}, err
		}
	}
	if err := r.applyRuntimeSecret(ctx, odigosNs, upstreamApiKey, fingerprint, revision); err != nil {
		return reportFailure("apply runtime credential", revision, err)
	}

	if err := r.applyConfigMap(ctx, odigosNs, configYaml, revision); err != nil {
		return reportFailure("apply proxy ConfigMap", revision, err)
	}

	if deployment.Spec.Template.Annotations[k8sconsts.OdigosAISettingsRevisionAnnotation] != revision {
		patch := client.MergeFrom(deployment.DeepCopy())
		if deployment.Spec.Template.Annotations == nil {
			deployment.Spec.Template.Annotations = make(map[string]string)
		}
		deployment.Spec.Template.Annotations[k8sconsts.OdigosAISettingsRevisionAnnotation] = revision
		if err := r.Client.Patch(ctx, &deployment, patch); err != nil {
			return reportFailure("update proxy Deployment", revision, err)
		}
		logger.Info("Rolled out LiteLLM with updated configuration", "revision", revision)
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}

	if err := r.cleanupOldCredentials(ctx, odigosNs, cfg); err != nil {
		return reportFailure("cleanup old credentials", revision, err)
	}
	state := "APPLYING"
	if deploymentRolledOut(&deployment) {
		state = "READY"
	}
	if err := r.writeProxyStatus(ctx, odigosNs, cfg.UpdateID, revision, modelHash, apiKeyConfigured, apiKeySource, state, ""); err != nil {
		return ctrl.Result{}, err
	}
	if state == "APPLYING" {
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}
	return ctrl.Result{RequeueAfter: credentialRetention}, nil
}

func renderLiteLLMConfig(model *common.InterrogationModelConfiguration) (string, error) {
	provider, name, apiBase := defaultProvider, defaultModel, ""
	if model != nil {
		if model.Provider != nil {
			provider = strings.TrimSpace(*model.Provider)
		}
		if model.Name != nil {
			name = strings.TrimSpace(*model.Name)
		}
		if model.ApiBase != nil {
			apiBase = strings.TrimSpace(*model.ApiBase)
		}
	}
	if provider == "" || name == "" || strings.Contains(provider, "/") {
		return "", fmt.Errorf("interrogation model requires a provider and name")
	}
	if apiBase != "" {
		parsed, err := url.Parse(apiBase)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil {
			return "", fmt.Errorf("interrogation model apiBase must be an HTTP(S) URL with a host and no credentials")
		}
	}

	config := liteLLMConfig{
		ModelList: []liteLLMModel{{
			ModelName: modelAlias,
			LiteLLMParams: liteLLMParams{
				Model:   fmt.Sprintf("%s/%s", provider, name),
				ApiKey:  "os.environ/LITELLM_UPSTREAM_API_KEY",
				ApiBase: apiBase,
			},
		}},
		LiteLLMSettings: liteLLMSettings{DropParams: true},
		GeneralSettings: liteLLMGeneralSettings{MasterKey: "os.environ/LITELLM_MASTER_KEY"},
	}

	configYaml, err := yaml.Marshal(config)
	if err != nil {
		return "", err
	}
	return string(configYaml), nil
}

func (r *liteLLMController) applyConfigMap(ctx context.Context, odigosNs string, configYaml string, revision string) error {
	owner := corev1.ConfigMap{}
	err := r.Client.Get(ctx, types.NamespacedName{Namespace: odigosNs, Name: consts.OdigosConfigurationName}, &owner)
	if err != nil {
		return err
	}

	configMap := corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   odigosNs,
			Name:        k8sconsts.OdigosLiteLLMName,
			Annotations: map[string]string{k8sconsts.OdigosAISettingsRevisionAnnotation: revision},
			Labels: map[string]string{
				"app.kubernetes.io/name":       k8sconsts.OdigosLiteLLMName,
				k8sconsts.OdigosSystemLabelKey: k8sconsts.OdigosSystemLabelValue,
			},
		},
		Data: map[string]string{
			k8sconsts.OdigosLiteLLMConfigFileName: configYaml,
		},
	}
	// Preserve Helm adoption metadata even when this controller creates the
	// ConfigMap from scratch. A rollback to an older chart renders this same
	// resource and Helm must be able to recognize its release ownership.
	if managedBy := owner.Labels["app.kubernetes.io/managed-by"]; managedBy != "" {
		configMap.Labels["app.kubernetes.io/managed-by"] = managedBy
	}
	for _, key := range []string{"meta.helm.sh/release-name", "meta.helm.sh/release-namespace"} {
		if value := owner.Annotations[key]; value != "" {
			configMap.Annotations[key] = value
		}
	}

	// helm no longer renders this ConfigMap, so tie its lifetime to the user facing
	// odigos-configuration to have it removed on uninstall.
	err = ctrl.SetControllerReference(&owner, &configMap, r.Scheme)
	if err != nil {
		return err
	}

	objApplyBytes, err := yaml.Marshal(configMap)
	if err != nil {
		return err
	}

	return r.Client.Patch(ctx, &configMap, client.RawPatch(types.ApplyYAMLPatchType, objApplyBytes), client.ForceOwnership, client.FieldOwner("instrumentor-litellm"))
}

func (r *liteLLMController) upstreamAPIKey(ctx context.Context, ns string, cfg *common.InterrogationConfiguration) ([]byte, error) {
	if cfg.CredentialSecretName != "" {
		if !strings.HasPrefix(cfg.CredentialSecretName, k8sconsts.OdigosLocalUIAICredentialSecretPrefix) {
			return nil, fmt.Errorf("invalid interrogation credential reference")
		}
		var selected corev1.Secret
		if err := r.Client.Get(ctx, types.NamespacedName{Namespace: ns, Name: cfg.CredentialSecretName}, &selected); err != nil {
			return nil, fmt.Errorf("get selected interrogation credential: %w", err)
		}
		if selected.Labels[k8sconsts.OdigosLocalUIAIConfigSourceLabel] != k8sconsts.OdigosLocalUIAIConfigSourceValue {
			return nil, fmt.Errorf("selected interrogation credential is not owned by local UI")
		}
		key, exists := selected.Data[k8sconsts.OdigosLiteLLMUpstreamApiKeySecretKey]
		if !exists {
			return nil, fmt.Errorf("selected interrogation credential has no upstream API key")
		}
		return key, nil
	}

	var baseline corev1.Secret
	if err := r.Client.Get(ctx, types.NamespacedName{Namespace: ns, Name: k8sconsts.OdigosLiteLLMName}, &baseline); err != nil {
		return nil, fmt.Errorf("get Helm interrogation credential: %w", err)
	}
	return baseline.Data[k8sconsts.OdigosLiteLLMUpstreamApiKeySecretKey], nil
}

func (r *liteLLMController) selectRevision(ctx context.Context, ns string, key []byte, fingerprint, configYaml string) (string, bool, error) {
	var runtime corev1.Secret
	err := r.Client.Get(ctx, types.NamespacedName{Namespace: ns, Name: k8sconsts.OdigosLiteLLMRuntimeSecretName}, &runtime)
	if apierrors.IsNotFound(err) {
		return uuid.NewString(), true, nil
	}
	if err != nil {
		return "", false, err
	}
	revision := runtime.Annotations[k8sconsts.OdigosAISettingsRevisionAnnotation]
	if runtime.Annotations[credentialFingerprintAnnotation] == fingerprint &&
		revision != "" && bytes.Equal(runtime.Data[k8sconsts.OdigosLiteLLMUpstreamApiKeySecretKey], key) {
		var proxyCM corev1.ConfigMap
		if err := r.Client.Get(ctx, types.NamespacedName{Namespace: ns, Name: k8sconsts.OdigosLiteLLMName}, &proxyCM); err != nil {
			if apierrors.IsNotFound(err) {
				return uuid.NewString(), true, nil
			}
			return "", false, err
		}
		if proxyCM.Annotations[k8sconsts.OdigosAISettingsRevisionAnnotation] == revision &&
			proxyCM.Data[k8sconsts.OdigosLiteLLMConfigFileName] == configYaml {
			return revision, false, nil
		}
	}
	return uuid.NewString(), true, nil
}

func (r *liteLLMController) applyRuntimeSecret(ctx context.Context, ns string, key []byte, fingerprint, revision string) error {
	var owner corev1.ConfigMap
	if err := r.Client.Get(ctx, types.NamespacedName{Namespace: ns, Name: consts.OdigosConfigurationName}, &owner); err != nil {
		return err
	}

	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		var runtime corev1.Secret
		err := r.Client.Get(ctx, types.NamespacedName{Namespace: ns, Name: k8sconsts.OdigosLiteLLMRuntimeSecretName}, &runtime)
		if apierrors.IsNotFound(err) {
			runtime = corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: ns,
					Name:      k8sconsts.OdigosLiteLLMRuntimeSecretName,
					Labels: map[string]string{
						k8sconsts.OdigosSystemLabelKey: k8sconsts.OdigosSystemLabelValue,
					},
					Annotations: map[string]string{
						credentialFingerprintAnnotation:              fingerprint,
						k8sconsts.OdigosAISettingsRevisionAnnotation: revision,
					},
				},
				Type: corev1.SecretTypeOpaque,
				Data: map[string][]byte{k8sconsts.OdigosLiteLLMUpstreamApiKeySecretKey: append([]byte{}, key...)},
			}
			if err := ctrl.SetControllerReference(&owner, &runtime, r.Scheme); err != nil {
				return err
			}
			return r.Client.Create(ctx, &runtime)
		}
		if err != nil {
			return err
		}
		if runtime.Annotations[credentialFingerprintAnnotation] == fingerprint &&
			runtime.Annotations[k8sconsts.OdigosAISettingsRevisionAnnotation] == revision &&
			bytes.Equal(runtime.Data[k8sconsts.OdigosLiteLLMUpstreamApiKeySecretKey], key) {
			return nil
		}
		if runtime.Annotations == nil {
			runtime.Annotations = map[string]string{}
		}
		runtime.Annotations[credentialFingerprintAnnotation] = fingerprint
		runtime.Annotations[k8sconsts.OdigosAISettingsRevisionAnnotation] = revision
		if runtime.Data == nil {
			runtime.Data = map[string][]byte{}
		}
		runtime.Data[k8sconsts.OdigosLiteLLMUpstreamApiKeySecretKey] = append([]byte{}, key...)
		return r.Client.Update(ctx, &runtime)
	})
}

func deploymentRolledOut(deployment *appsv1.Deployment) bool {
	wanted := int32(1)
	if deployment.Spec.Replicas != nil {
		wanted = *deployment.Spec.Replicas
	}
	return wanted > 0 && deployment.Status.ObservedGeneration >= deployment.Generation &&
		deployment.Status.Replicas == wanted &&
		deployment.Status.UpdatedReplicas == wanted &&
		deployment.Status.ReadyReplicas == wanted &&
		deployment.Status.AvailableReplicas == wanted &&
		deployment.Status.UnavailableReplicas == 0
}

func (r *liteLLMController) writeProxyStatus(ctx context.Context, ns, updateID, revision, modelHash string, apiKeyConfigured bool, apiKeySource, state, message string) error {
	var owner corev1.ConfigMap
	if err := r.Client.Get(ctx, types.NamespacedName{Namespace: ns, Name: consts.OdigosConfigurationName}, &owner); err != nil {
		return err
	}
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		var statusCM corev1.ConfigMap
		getErr := r.Client.Get(ctx, types.NamespacedName{Namespace: ns, Name: k8sconsts.OdigosInterrogationStatusConfigMapName}, &statusCM)
		if getErr != nil && !apierrors.IsNotFound(getErr) {
			return getErr
		}
		var current proxyStatus
		if statusCM.Data != nil {
			_ = json.Unmarshal([]byte(statusCM.Data["proxy"]), &current)
		}
		ownerRefExists := false
		for _, ref := range statusCM.OwnerReferences {
			if ref.UID == owner.UID && ref.Name == owner.Name && ref.Kind == "ConfigMap" {
				ownerRefExists = true
				break
			}
		}
		if !apierrors.IsNotFound(getErr) {
			if err := controllerutil.SetOwnerReference(&owner, &statusCM, r.Scheme); err != nil {
				return err
			}
		}
		if current.UpdateID == updateID && current.Revision == revision && current.ModelHash == modelHash &&
			current.APIKeyConfigured == apiKeyConfigured && current.APIKeySource == apiKeySource &&
			current.State == state && current.Error == message && ownerRefExists {
			return nil
		}
		payload, err := json.Marshal(proxyStatus{
			UpdateID: updateID, Revision: revision, ModelHash: modelHash,
			APIKeyConfigured: apiKeyConfigured, APIKeySource: apiKeySource,
			State: state, Error: message,
			ObservedAt: time.Now().UTC().Format(time.RFC3339Nano),
		})
		if err != nil {
			return err
		}
		if apierrors.IsNotFound(getErr) {
			statusCM = corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
				Namespace: ns, Name: k8sconsts.OdigosInterrogationStatusConfigMapName,
				Labels: map[string]string{k8sconsts.OdigosSystemLabelKey: k8sconsts.OdigosSystemLabelValue},
			}, Data: map[string]string{"proxy": string(payload)}}
			if err := ctrl.SetControllerReference(&owner, &statusCM, r.Scheme); err != nil {
				return err
			}
			return r.Client.Create(ctx, &statusCM)
		}
		if statusCM.Data == nil {
			statusCM.Data = make(map[string]string)
		}
		statusCM.Data["proxy"] = string(payload)
		return r.Client.Update(ctx, &statusCM)
	})
}

func (r *liteLLMController) cleanupOldCredentials(ctx context.Context, ns string, cfg *common.InterrogationConfiguration) error {
	var secrets corev1.SecretList
	if err := r.Client.List(ctx, &secrets, client.InNamespace(ns), client.MatchingLabels{
		k8sconsts.OdigosLocalUIAIConfigSourceLabel: k8sconsts.OdigosLocalUIAIConfigSourceValue,
	}); err != nil {
		return err
	}
	cutoff := time.Now().Add(-credentialRetention)
	for i := range secrets.Items {
		secret := &secrets.Items[i]
		if !strings.HasPrefix(secret.Name, k8sconsts.OdigosLocalUIAICredentialSecretPrefix) ||
			secret.Name == cfg.CredentialSecretName || secret.Name == cfg.PreviousCredentialSecretName ||
			secret.CreationTimestamp.IsZero() || secret.CreationTimestamp.Time.After(cutoff) {
			continue
		}
		if err := r.Client.Delete(ctx, secret); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}
	return nil
}
