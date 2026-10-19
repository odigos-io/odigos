package litellm

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"net/url"
	"strings"
	"time"

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

	// LiteLLM is only deployed with an enterprise token, so there is nothing to configure without it.
	deployment := appsv1.Deployment{}
	err = r.Client.Get(ctx, types.NamespacedName{Namespace: odigosNs, Name: k8sconsts.OdigosLiteLLMName}, &deployment)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	configYaml, err := renderLiteLLMConfig(cfg.Model)
	if err != nil {
		return ctrl.Result{}, err
	}

	upstreamApiKey, err := r.upstreamAPIKey(ctx, odigosNs, cfg)
	if err != nil {
		return ctrl.Result{}, err
	}

	configHash := sha256.Sum256(append(append([]byte(configYaml), 0), upstreamApiKey...))
	fingerprint := fmt.Sprintf("%x", configHash)
	if err := r.applyRuntimeSecret(ctx, odigosNs, upstreamApiKey, fingerprint); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.applyConfigMap(ctx, odigosNs, configYaml); err != nil {
		return ctrl.Result{}, err
	}

	if deployment.Spec.Template.Annotations[k8sconsts.OdigosLiteLLMConfigHashAnnotation] != fingerprint {
		patch := client.MergeFrom(deployment.DeepCopy())
		if deployment.Spec.Template.Annotations == nil {
			deployment.Spec.Template.Annotations = make(map[string]string)
		}
		deployment.Spec.Template.Annotations[k8sconsts.OdigosLiteLLMConfigHashAnnotation] = fingerprint
		if err := r.Client.Patch(ctx, &deployment, patch); err != nil {
			return ctrl.Result{}, err
		}
		logger.Info("Rolled out LiteLLM with updated configuration")
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}

	if err := r.cleanupOldCredentials(ctx, odigosNs, cfg); err != nil {
		return ctrl.Result{}, err
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

func (r *liteLLMController) applyConfigMap(ctx context.Context, odigosNs string, configYaml string) error {
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
			Annotations: map[string]string{},
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

func (r *liteLLMController) applyRuntimeSecret(ctx context.Context, ns string, key []byte, fingerprint string) error {
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
						credentialFingerprintAnnotation: fingerprint,
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
			bytes.Equal(runtime.Data[k8sconsts.OdigosLiteLLMUpstreamApiKeySecretKey], key) {
			return nil
		}
		if runtime.Annotations == nil {
			runtime.Annotations = map[string]string{}
		}
		runtime.Annotations[credentialFingerprintAnnotation] = fingerprint
		if runtime.Data == nil {
			runtime.Data = map[string][]byte{}
		}
		runtime.Data[k8sconsts.OdigosLiteLLMUpstreamApiKeySecretKey] = append([]byte{}, key...)
		return r.Client.Update(ctx, &runtime)
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
