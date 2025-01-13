package odigosconfig

import (
	"context"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	k8sconsts "github.com/odigos-io/odigos/k8sutils/pkg/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"github.com/odigos-io/odigos/profiles"
	"github.com/odigos-io/odigos/profiles/profile"
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
	Tier   common.OdigosTier
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

	// effective profiles are what is actually used in the cluster
	availableProfiles := profiles.GetAvailableProfilesForTier(r.Tier)
	effectiveProfiles := calculateEffectiveProfiles(odigosConfig.Profiles, availableProfiles)
	odigosConfig.Profiles = effectiveProfiles
	modifyConfigWithEffectiveProfiles(effectiveProfiles, odigosConfig)

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

func modifyConfigWithEffectiveProfiles(effectiveProfiles []common.ProfileName, odigosConfig *common.OdigosConfiguration) {
	for _, profileName := range effectiveProfiles {
		p := profiles.ProfilesByName[profileName]
		if p.ModifyConfigFunc != nil {
			p.ModifyConfigFunc(odigosConfig)
		}
	}
}

// from the list of input profiles, calculate the effective profiles:
// - check the dependencies of each profile and add them to the list
// - remove profiles which are not present in the profiles list
func calculateEffectiveProfiles(configProfiles []common.ProfileName, availableProfiles []profile.Profile) []common.ProfileName {

	effectiveProfiles := []common.ProfileName{}
	for _, profileName := range configProfiles {

		// ignored missing profiles (either not available for tier or typos)
		p, found := findProfileNameInAvailableList(profileName, availableProfiles)
		if !found {
			continue
		}

		effectiveProfiles = append(effectiveProfiles, profileName)

		// if this profile has dependencies, add them to the list
		if p.Dependencies != nil {
			effectiveProfiles = append(effectiveProfiles, calculateEffectiveProfiles(p.Dependencies, availableProfiles)...)
		}
	}
	return effectiveProfiles
}

func findProfileNameInAvailableList(profileName common.ProfileName, availableProfiles []profile.Profile) (profile.Profile, bool) {
	// there aren't many profiles, so a linear search is fine
	for _, p := range availableProfiles {
		if p.ProfileName == profileName {
			return p, true
		}
	}
	return profile.Profile{}, false
}
