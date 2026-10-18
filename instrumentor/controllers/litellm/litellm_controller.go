package litellm

import (
	"context"
	"crypto/sha256"
	"fmt"

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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

const (
	defaultProvider = "openai"
	defaultModel    = "gpt-4o-mini"
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

	// LiteLLM is only deployed with an enterprise token, so there is nothing to configure without it.
	deployment := appsv1.Deployment{}
	err = r.Client.Get(ctx, types.NamespacedName{Namespace: odigosNs, Name: k8sconsts.OdigosLiteLLMName}, &deployment)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	configYaml, err := renderLiteLLMConfig(odigosConfiguration.Interrogation.Model)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.applyConfigMap(ctx, odigosNs, configYaml)
	if err != nil {
		return ctrl.Result{}, err
	}

	secret := corev1.Secret{}
	err = r.Client.Get(ctx, types.NamespacedName{Namespace: odigosNs, Name: k8sconsts.OdigosLiteLLMName}, &secret)
	if err != nil && !apierrors.IsNotFound(err) {
		return ctrl.Result{}, err
	}
	upstreamApiKey := secret.Data[k8sconsts.OdigosLiteLLMUpstreamApiKeySecretKey]

	configHash := fmt.Sprintf("%x", sha256.Sum256(append([]byte(configYaml), upstreamApiKey...)))
	if deployment.Spec.Template.Annotations[k8sconsts.OdigosLiteLLMConfigHashAnnotation] == configHash {
		return ctrl.Result{}, nil
	}

	patch := client.MergeFrom(deployment.DeepCopy())
	if deployment.Spec.Template.Annotations == nil {
		deployment.Spec.Template.Annotations = make(map[string]string)
	}
	deployment.Spec.Template.Annotations[k8sconsts.OdigosLiteLLMConfigHashAnnotation] = configHash
	err = r.Client.Patch(ctx, &deployment, patch)
	if err != nil {
		return ctrl.Result{}, err
	}

	logger.Info("Rolled out LiteLLM with updated configuration")
	return ctrl.Result{}, nil
}

func renderLiteLLMConfig(model *common.InterrogationModelConfiguration) (string, error) {
	provider, name, apiBase := defaultProvider, defaultModel, ""
	if model != nil {
		if model.Provider != "" {
			provider = model.Provider
		}
		if model.Name != "" {
			name = model.Name
		}
		apiBase = model.ApiBase
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
			Namespace: odigosNs,
			Name:      k8sconsts.OdigosLiteLLMName,
			Labels: map[string]string{
				"app.kubernetes.io/name":       k8sconsts.OdigosLiteLLMName,
				k8sconsts.OdigosSystemLabelKey: k8sconsts.OdigosSystemLabelValue,
			},
		},
		Data: map[string]string{
			k8sconsts.OdigosLiteLLMConfigFileName: configYaml,
		},
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
