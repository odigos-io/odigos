package odigosconfig

import (
	"context"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	k8sconsts "github.com/odigos-io/odigos/k8sutils/pkg/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"github.com/odigos-io/odigos/profiles"
	"github.com/odigos-io/odigos/profiles/sizing"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

type odigosConfigController struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *odigosConfigController) Reconcile(ctx context.Context, _ ctrl.Request) (ctrl.Result, error) {

	odigosConfig, err := r.getOdigosConfigUserObject(ctx)
	if err != nil {
		return ctrl.Result{}, err
	}

	// make sure the default ignored namespaces are always present
	odigosConfig.IgnoredNamespaces = mergeIgnoredItemLists(odigosConfig.IgnoredNamespaces, k8sconsts.DefaultIgnoredNamespaces)
	odigosConfig.IgnoredNamespaces = append(odigosConfig.IgnoredNamespaces, env.GetCurrentNamespace())

	// make sure the default ignored containers are always present
	odigosConfig.IgnoredContainers = mergeIgnoredItemLists(odigosConfig.IgnoredContainers, k8sconsts.DefaultIgnoredContainers)

	applyProfilesToOdigosConfig(odigosConfig)

	// if none of the profiles set sizing for collectors, use size_s as default, so the values are never nil
	// if the values were already set (by user or profile) this is a no-op
	sizing.SizeSProfile.ModifyConfigFunc(odigosConfig)

	err = r.persistEffectiveConfig(ctx, odigosConfig)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *odigosConfigController) getOdigosConfigUserObject(ctx context.Context) (*common.OdigosConfiguration, error) {
	var configMap corev1.ConfigMap
	var odigosConfig common.OdigosConfiguration
	odigosNs := env.GetCurrentNamespace()

	// read current content in odigos-config, which is the content supplied by the user.
	// this is the baseline for reconciling, without defaults and profiles applied.
	err := r.Client.Get(ctx, types.NamespacedName{Namespace: odigosNs, Name: consts.OdigosConfigurationName}, &configMap)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal([]byte(configMap.Data[consts.OdigosConfigurationFileName]), &odigosConfig)
	if err != nil {
		return nil, err
	}

	return &odigosConfig, nil
}

func (r *odigosConfigController) persistEffectiveConfig(ctx context.Context, effectiveConfig *common.OdigosConfiguration) error {
	odigosNs := env.GetCurrentNamespace()

	// apply patch the OdigosEffectiveConfigName configmap with the effective configuration
	// this is the configuration after applying defaults and profiles.

	effectiveConfigYamlText, err := yaml.Marshal(effectiveConfig)
	if err != nil {
		return err
	}

	effectiveConfigMap := corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: odigosNs,
			Name:      consts.OdigosEffectiveConfigName,
		},
		Data: map[string]string{
			consts.OdigosConfigurationFileName: string(effectiveConfigYamlText),
		},
	}

	objApplyBytes, err := yaml.Marshal(effectiveConfigMap)
	if err != nil {
		return err
	}

	err = r.Client.Patch(ctx, &effectiveConfigMap, client.RawPatch(types.ApplyYAMLPatchType, objApplyBytes), client.ForceOwnership, client.FieldOwner("scheduler-odigosconfig"))
	if err != nil {
		return err
	}

	logger := ctrl.LoggerFrom(ctx)
	logger.Info("Successfully persisted effective configuration")

	return nil
}

func applySingleProfile(profile common.ProfileName, odigosConfig *common.OdigosConfiguration) {
	profileConfig, found := profiles.ProfilesByName[profile]
	if !found {
		return
	}

	if profileConfig.ModifyConfigFunc != nil {
		profileConfig.ModifyConfigFunc(odigosConfig)
	}

	for _, dependency := range profileConfig.Dependencies {
		applySingleProfile(dependency, odigosConfig)
	}
}

func applyProfilesToOdigosConfig(odigosConfig *common.OdigosConfiguration) {
	for _, profile := range odigosConfig.Profiles {
		applySingleProfile(profile, odigosConfig)
	}
}
