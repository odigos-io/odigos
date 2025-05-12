package odigosconfig

import (
	"context"
	"fmt"
	"strings"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"github.com/odigos-io/odigos/profiles"
	"github.com/odigos-io/odigos/profiles/manifests"
	"github.com/odigos-io/odigos/profiles/sizing"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

type odigosConfigController struct {
	client.Client
	Scheme        *runtime.Scheme
	Tier          common.OdigosTier
	OdigosVersion string
	DynamicClient *dynamic.DynamicClient
}

func (r *odigosConfigController) Reconcile(ctx context.Context, _ ctrl.Request) (ctrl.Result, error) {

	odigosConfig, err := r.getOdigosConfigUserObject(ctx)
	if err != nil {
		return ctrl.Result{}, err
	}

	odigosDeployment := corev1.ConfigMap{}
	err = r.Client.Get(ctx, types.NamespacedName{Namespace: env.GetCurrentNamespace(), Name: k8sconsts.OdigosDeploymentConfigMapName}, &odigosDeployment)
	if err != nil {
		return ctrl.Result{}, err
	}

	// effective profiles are what is actually used in the cluster (minus non existing profiles and plus dependencies)
	availableProfiles := profiles.GetAvailableProfilesForTier(r.Tier)

	allProfiles := make([]common.ProfileName, 0)
	allProfiles = append(allProfiles, odigosConfig.Profiles...)

	if tokenProfilesString, ok := odigosDeployment.Data[k8sconsts.OdigosDeploymentConfigMapOnPremClientProfilesKey]; ok {
		tokenProfiles := strings.Split(tokenProfilesString, ", ")
		// cast tokenProfiles to common.ProfileName
		for _, p := range tokenProfiles {
			allProfiles = append(allProfiles, common.ProfileName(p))
		}
	}

	effectiveProfiles := calculateEffectiveProfiles(allProfiles, availableProfiles)

	// apply the current profiles list to the cluster
	err = r.applyProfileManifests(ctx, effectiveProfiles)
	if err != nil {
		return ctrl.Result{}, err
	}

	// make sure the default ignored namespaces are always present
	odigosConfig.IgnoredNamespaces = mergeIgnoredItemLists(odigosConfig.IgnoredNamespaces, k8sconsts.DefaultIgnoredNamespaces)
	odigosConfig.IgnoredNamespaces = append(odigosConfig.IgnoredNamespaces, env.GetCurrentNamespace())

	// make sure the default ignored containers are always present
	odigosConfig.IgnoredContainers = mergeIgnoredItemLists(odigosConfig.IgnoredContainers, k8sconsts.DefaultIgnoredContainers)

	modifyConfigWithEffectiveProfiles(effectiveProfiles, odigosConfig)
	odigosConfig.Profiles = effectiveProfiles

	// if none of the profiles set sizing for collectors, use size_s as default, so the values are never nil
	// if the values were already set (by user or profile) this is a no-op
	sizing.SizeSProfile.ModifyConfigFunc(odigosConfig)

	// TODO: revisit doing this here, might be nicer to maintain in a more generic way
	// and have it on the config object itself.
	// I want to preserve that user input (specific request or empty), and persist the resolved value in effective config.
	resolveMountMethod(odigosConfig)
	resolveEnvInjectionMethod(odigosConfig)

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

func (r *odigosConfigController) applyProfileManifests(ctx context.Context, effectiveProfiles []common.ProfileName) error {

	profileDeploymentHash := calculateProfilesDeploymentHash(effectiveProfiles, r.OdigosVersion)

	for _, profileName := range effectiveProfiles {

		yamls, err := manifests.ReadProfileYamlManifests(profileName)
		if err != nil {
			return err
		}

		for _, yamlBytes := range yamls {
			err = r.applySingleProfileManifest(ctx, profileName, yamlBytes, profileDeploymentHash)
			if err != nil {
				return err
			}
		}
	}

	// after we applied all the current profiles, we need to deleted resources
	// which did not participate in the current deployment.
	// we will delete any resource with the "odigos.io/profiles-hash" label which is not the current hash.
	differentHashSelector, _ := labels.NewRequirement(k8sconsts.OdigosProfilesHashLabel, selection.NotEquals, []string{profileDeploymentHash})
	managedByProfileSelector, _ := labels.NewRequirement(k8sconsts.OdigosProfilesManagedByLabel, selection.Equals, []string{k8sconsts.OdigosProfilesManagedByValue})
	differentHashLabelSelector := labels.NewSelector().Add(*differentHashSelector, *managedByProfileSelector)
	listOptions := &client.ListOptions{LabelSelector: differentHashLabelSelector, Namespace: env.GetCurrentNamespace()}

	processesList := odigosv1alpha1.ProcessorList{}
	err := r.Client.List(ctx, &processesList, listOptions)
	if err != nil {
		return err
	}
	// TODO: migrate to DeleteAllOf once we drop support for old k8s versions
	for i := range processesList.Items {
		err = r.Client.Delete(ctx, &processesList.Items[i])
		if err != nil {
			return err
		}
	}

	instrumentationRulesList := odigosv1alpha1.InstrumentationRuleList{}
	err = r.Client.List(ctx, &instrumentationRulesList, listOptions)
	if err != nil {
		return err
	}
	for i := range instrumentationRulesList.Items {
		err = r.Client.Delete(ctx, &instrumentationRulesList.Items[i])
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *odigosConfigController) applySingleProfileManifest(ctx context.Context, profileName common.ProfileName, yamlBytes []byte, profileDeploymentHash string) error {

	obj := &unstructured.Unstructured{}
	err := yaml.Unmarshal(yamlBytes, obj)
	if err != nil {
		return err
	}

	labels := obj.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[k8sconsts.OdigosProfilesHashLabel] = profileDeploymentHash
	labels[k8sconsts.OdigosProfilesManagedByLabel] = k8sconsts.OdigosProfilesManagedByValue
	obj.SetLabels(labels)

	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations[k8sconsts.OdigosProfileAnnotation] = string(profileName)
	obj.SetAnnotations(annotations)

	gvk := obj.GroupVersionKind()
	resource, found := supportedKindToResource[gvk.Kind]
	if !found {
		return fmt.Errorf("unsupported kind for profile manifest %s", gvk.Kind)
	}
	gvr := schema.GroupVersionResource{
		Group:    gvk.Group,
		Version:  gvk.Version,
		Resource: resource,
	}

	resourceClient := r.DynamicClient.Resource(gvr).Namespace(env.GetCurrentNamespace())
	_, err = resourceClient.Apply(ctx, obj.GetName(), obj, metav1.ApplyOptions{
		FieldManager: "scheduler-odigosconfig",
	})
	if err != nil {
		return err
	}

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

func resolveMountMethod(odigosConfig *common.OdigosConfiguration) {
	defaultMountMethod := common.K8sVirtualDeviceMountMethod

	if odigosConfig.MountMethod == nil {
		odigosConfig.MountMethod = &defaultMountMethod
		return
	}

	switch *odigosConfig.MountMethod {
	case common.K8sHostPathMountMethod:
		return
	case common.K8sVirtualDeviceMountMethod:
		return
	default:
		// any illegal value will be defaulted to host-path
		// TODO: emit some error here and think how to handle it
		odigosConfig.MountMethod = &defaultMountMethod
	}
}

func resolveEnvInjectionMethod(odigosConfig *common.OdigosConfiguration) {
	defaultInjectionMethod := common.PodManifestEnvInjectionMethod

	if odigosConfig.AgentEnvVarsInjectionMethod == nil {
		odigosConfig.AgentEnvVarsInjectionMethod = &defaultInjectionMethod
		return
	}

	switch *odigosConfig.AgentEnvVarsInjectionMethod {
	case common.LoaderFallbackToPodManifestInjectionMethod:
		return
	case common.LoaderEnvInjectionMethod:
		return
	case common.PodManifestEnvInjectionMethod:
		return
	default:
		// any illegal value will be defaulted to pod-manifest
		odigosConfig.AgentEnvVarsInjectionMethod = &defaultInjectionMethod
	}
}
