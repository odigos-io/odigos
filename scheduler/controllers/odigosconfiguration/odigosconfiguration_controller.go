package odigosconfiguration

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"github.com/odigos-io/odigos/k8sutils/pkg/sizing"
	"github.com/odigos-io/odigos/profiles"
	"github.com/odigos-io/odigos/profiles/manifests"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
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
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/yaml"
)

type odigosConfigurationController struct {
	client.Client
	Scheme        *runtime.Scheme
	Tier          common.OdigosTier
	OdigosVersion string
	DynamicClient *dynamic.DynamicClient
}

func (r *odigosConfigurationController) Reconcile(ctx context.Context, _ ctrl.Request) (ctrl.Result, error) {
	logger := ctrl.LoggerFrom(ctx)

	odigosConfigMap, err := r.getOdigosConfigMap(ctx)
	if err != nil {
		return ctrl.Result{}, err
	}

	odigosDeployment := corev1.ConfigMap{}
	err = r.Client.Get(ctx, types.NamespacedName{Namespace: env.GetCurrentNamespace(), Name: k8sconsts.OdigosDeploymentConfigMapName}, &odigosDeployment)
	if err != nil {
		return ctrl.Result{}, err
	}
	odigosConfiguration := common.OdigosConfiguration{}
	err = yaml.Unmarshal([]byte(odigosConfigMap.Data[consts.OdigosConfigurationFileName]), &odigosConfiguration)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Read and merge remote config (from central-backend) if it exists
	// Remote config takes precedence over helm-managed config
	remoteConfig, err := r.getRemoteConfig(ctx)
	if err != nil {
		logger.Error(err, "Failed to get remote config, using only helm-managed config")
	} else if remoteConfig != nil {
		mergeRemoteConfig(&odigosConfiguration, remoteConfig)
		logger.V(1).Info("Merged remote config into effective config")
	}

	// effective profiles are what is actually used in the cluster (minus non existing profiles and plus dependencies)
	availableProfiles := profiles.GetAvailableProfilesForTier(r.Tier)

	allProfiles := make([]common.ProfileName, 0)
	allProfiles = append(allProfiles, odigosConfiguration.Profiles...)

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
	odigosConfiguration.IgnoredNamespaces = mergeIgnoredItemLists(odigosConfiguration.IgnoredNamespaces, k8sconsts.DefaultIgnoredNamespaces)
	currentNamespace := env.GetCurrentNamespace()
	// Only add the current namespace to ignored namespaces if ignoreOdigosNamespace is not explicitly set to false
	if odigosConfiguration.IgnoreOdigosNamespace == nil || *odigosConfiguration.IgnoreOdigosNamespace {
		odigosConfiguration.IgnoredNamespaces = append(odigosConfiguration.IgnoredNamespaces, currentNamespace)
	} else {
		// Remove the namespace from the list if ignoreOdigosNamespace is explicitly set to false
		odigosConfiguration.IgnoredNamespaces = removeItemFromList(odigosConfiguration.IgnoredNamespaces, currentNamespace)
	}

	// make sure the default ignored containers are always present
	odigosConfiguration.IgnoredContainers = mergeIgnoredItemLists(odigosConfiguration.IgnoredContainers, k8sconsts.DefaultIgnoredContainers)

	modifyConfigWithEffectiveProfiles(effectiveProfiles, &odigosConfiguration)
	odigosConfiguration.Profiles = effectiveProfiles

	// compute effective collector configurations that merge sizing presets with existing configurations
	// preserving all non-sizing attributes (ServiceGraphDisabled, CollectorOwnMetricsPort, etc.)
	odigosConfiguration.CollectorGateway, odigosConfiguration.CollectorNode = sizing.ComputeEffectiveCollectorConfig(&odigosConfiguration)

	// TODO: revisit doing this here, might be nicer to maintain in a more generic way
	// and have it on the config object itself.
	// I want to preserve that user input (specific request or empty), and persist the resolved value in effective config.
	resolveMountMethod(&odigosConfiguration)
	resolveEnvInjectionMethod(&odigosConfiguration)

	err = verifyMetricsConfig(&odigosConfiguration)
	if err != nil {
		return ctrl.Result{}, reconcile.TerminalError(err)
	}

	err = r.persistEffectiveConfig(ctx, &odigosConfiguration, odigosConfigMap)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *odigosConfigurationController) getOdigosConfigMap(ctx context.Context) (*corev1.ConfigMap, error) {
	var configMap corev1.ConfigMap
	odigosNs := env.GetCurrentNamespace()

	// read current content in odigos-configuration, which is the content supplied by the user.
	// this is the baseline for reconciling, without defaults and profiles applied.
	err := r.Client.Get(ctx, types.NamespacedName{Namespace: odigosNs, Name: consts.OdigosConfigurationName}, &configMap)
	if err != nil {
		return nil, err
	}

	return &configMap, nil
}

// getRemoteConfig reads the odigos-remote-config ConfigMap which contains
// configuration managed by the central-backend. This config takes precedence
// over helm-managed configuration for supported fields.
func (r *odigosConfigurationController) getRemoteConfig(ctx context.Context) (*common.OdigosConfiguration, error) {
	var configMap corev1.ConfigMap
	odigosNs := env.GetCurrentNamespace()

	err := r.Client.Get(ctx, types.NamespacedName{Namespace: odigosNs, Name: consts.OdigosRemoteConfigName}, &configMap)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Remote config doesn't exist - this is expected for most deployments
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get remote config ConfigMap: %w", err)
	}

	if configMap.Data == nil || configMap.Data[consts.OdigosConfigurationFileName] == "" {
		// ConfigMap exists but is empty - treat as no remote config
		return nil, nil
	}

	remoteConfig := &common.OdigosConfiguration{}
	err = yaml.Unmarshal([]byte(configMap.Data[consts.OdigosConfigurationFileName]), remoteConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to parse remote config: %w", err)
	}

	return remoteConfig, nil
}

// mergeRemoteConfig merges the remote configuration (from central-backend) into the base configuration.
// Remote config values take precedence over helm-managed values for supported fields.
func mergeRemoteConfig(baseConfig *common.OdigosConfiguration, remoteConfig *common.OdigosConfiguration) {
	if remoteConfig == nil {
		return
	}

	if remoteConfig.Rollout != nil {
		if baseConfig.Rollout == nil {
			baseConfig.Rollout = &common.RolloutConfiguration{}
		}
		if remoteConfig.Rollout.AutomaticRolloutDisabled != nil {
			baseConfig.Rollout.AutomaticRolloutDisabled = remoteConfig.Rollout.AutomaticRolloutDisabled
		}
	}

	// Future fields can be added here following the same pattern:
	// - ignoredNamespaces, ignoredContainers
	// - profiles
	// - ...
}

func (r *odigosConfigurationController) persistEffectiveConfig(ctx context.Context, effectiveConfig *common.OdigosConfiguration, owner *corev1.ConfigMap) error {
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

	// the effective configuration is owned by the odigos config.
	// odigos config is a user facing object and is created upon installation.
	// the effective config is managed by this controller.
	// setting this owner reference makes sure the effective config is cleaned up
	// when the odigos config is deleted.
	err = ctrl.SetControllerReference(owner, &effectiveConfigMap, r.Scheme)
	if err != nil {
		return err
	}

	objApplyBytes, err := yaml.Marshal(effectiveConfigMap)
	if err != nil {
		return err
	}

	err = r.Client.Patch(ctx, &effectiveConfigMap, client.RawPatch(types.ApplyYAMLPatchType, objApplyBytes), client.ForceOwnership, client.FieldOwner("scheduler-odigosconfiguration"))
	if err != nil {
		return err
	}

	logger := ctrl.LoggerFrom(ctx)
	logger.Info("Successfully persisted effective configuration")

	return nil
}

func (r *odigosConfigurationController) applyProfileManifests(ctx context.Context, effectiveProfiles []common.ProfileName) error {

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

	ActionsList := odigosv1alpha1.ActionList{}
	err = r.Client.List(ctx, &ActionsList, listOptions)
	if err != nil {
		return err
	}

	for i := range ActionsList.Items {
		err = r.Client.Delete(ctx, &ActionsList.Items[i])
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *odigosConfigurationController) applySingleProfileManifest(ctx context.Context, profileName common.ProfileName, yamlBytes []byte, profileDeploymentHash string) error {
	obj := &unstructured.Unstructured{}
	err := yaml.Unmarshal(yamlBytes, obj)
	if err != nil {
		return err
	}

	odigosNs := env.GetCurrentNamespace()

	// profiles are read without namespace, and we need to add it ourselves.
	// the namespace is usually odigos-system, but user can set it to anything,
	// which is why we need to address it here.
	// the namespace is set on the object itself, but not in the yamlbytes for the apply,
	// which is ok and works (the applied object takes the namespace from the object)
	obj.SetNamespace(odigosNs)

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

	resourceClient := r.DynamicClient.Resource(gvr).Namespace(odigosNs)
	_, err = resourceClient.Apply(ctx, obj.GetName(), obj, metav1.ApplyOptions{
		FieldManager: "scheduler-odigosconfiguration",
		Force:        true,
	})
	if err != nil {
		return err
	}

	return nil
}

func modifyConfigWithEffectiveProfiles(effectiveProfiles []common.ProfileName, odigosConfiguration *common.OdigosConfiguration) {
	for _, profileName := range effectiveProfiles {
		p := profiles.ProfilesByName[profileName]
		if p.ModifyConfigFunc != nil {
			p.ModifyConfigFunc(odigosConfiguration)
		}
	}
}

func getInitContainerResources(config *common.OdigosConfiguration) *common.AgentsInitContainerResources {
	const (
		defaultAgentsInitContainerRequestCPUm      = 300
		defaultAgentsInitContainerLimitCPUm        = 300
		defaultAgentsInitContainerRequestMemoryMiB = 300
		defaultAgentsInitContainerLimitMemoryMiB   = 300
	)
	logger := log.FromContext(context.Background())

	cpuRequest := defaultAgentsInitContainerRequestCPUm
	cpuLimit := defaultAgentsInitContainerLimitCPUm

	if config.AgentsInitContainerResources != nil {
		if config.AgentsInitContainerResources.RequestCPUm > 0 {
			cpuRequest = config.AgentsInitContainerResources.RequestCPUm
		}
		if config.AgentsInitContainerResources.LimitCPUm > 0 {
			cpuLimit = config.AgentsInitContainerResources.LimitCPUm
		}
	}
	// validate the CPU request value or default to the default value if it is not valid
	_, err := resource.ParseQuantity(fmt.Sprintf("%dm", cpuRequest))
	if err != nil {
		logger.Error(err, "failed to parse CPU request for init container", "cpuRequest", cpuRequest)
		cpuRequest = defaultAgentsInitContainerRequestCPUm
	}
	// validate the CPU limit value or default to the default value if it is not valid
	_, err = resource.ParseQuantity(fmt.Sprintf("%dm", cpuLimit))
	if err != nil {
		logger.Error(err, "failed to parse CPU limit for init container", "cpuLimit", cpuLimit)
		cpuLimit = defaultAgentsInitContainerLimitCPUm
	}

	memoryRequest := defaultAgentsInitContainerRequestMemoryMiB
	memoryLimit := defaultAgentsInitContainerLimitMemoryMiB
	if config.AgentsInitContainerResources != nil {
		if config.AgentsInitContainerResources.RequestMemoryMiB > 0 {
			memoryRequest = config.AgentsInitContainerResources.RequestMemoryMiB
		}
		if config.AgentsInitContainerResources.LimitMemoryMiB > 0 {
			memoryLimit = config.AgentsInitContainerResources.LimitMemoryMiB
		}
	}

	// validate the memory request value or default to the default value if it is not valid
	_, err = resource.ParseQuantity(fmt.Sprintf("%dMi", memoryRequest))
	// Fallback to default value if the value is not valid
	if err != nil {
		logger.Error(err, "failed to parse memory request for init container", "memoryRequest", memoryRequest)
		memoryRequest = defaultAgentsInitContainerRequestMemoryMiB
	}
	// validate the memory limit value or default to the default value if it is not valid
	_, err = resource.ParseQuantity(fmt.Sprintf("%dMi", memoryLimit))
	// Fallback to default value if the value is not valid
	if err != nil {
		logger.Error(err, "failed to parse memory limit for init container", "memoryLimit", memoryLimit)
		memoryLimit = defaultAgentsInitContainerLimitMemoryMiB
	}

	return &common.AgentsInitContainerResources{
		RequestCPUm:      cpuRequest,
		LimitCPUm:        cpuLimit,
		RequestMemoryMiB: memoryRequest,
		LimitMemoryMiB:   memoryLimit,
	}
}

func resolveMountMethod(odigosConfiguration *common.OdigosConfiguration) {
	defaultMountMethod := common.K8sVirtualDeviceMountMethod

	if odigosConfiguration.MountMethod == nil {
		odigosConfiguration.MountMethod = &defaultMountMethod
		return
	}

	switch *odigosConfiguration.MountMethod {
	case common.K8sHostPathMountMethod:
		return
	case common.K8sVirtualDeviceMountMethod:
		return
	case common.K8sInitContainerMountMethod:
		odigosConfiguration.AgentsInitContainerResources = getInitContainerResources(odigosConfiguration)
		return
	default:
		// any illegal value will be defaulted to host-path
		// TODO: emit some error here and think how to handle it
		odigosConfiguration.MountMethod = &defaultMountMethod
	}
}

func resolveEnvInjectionMethod(odigosConfig *common.OdigosConfiguration) {
	defaultInjectionMethod := common.LoaderFallbackToPodManifestInjectionMethod

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
		odigosConfig.AgentEnvVarsInjectionMethod = &defaultInjectionMethod
	}
}

func verifyMetricsConfig(odigosConfiguration *common.OdigosConfiguration) error {
	if odigosConfiguration.MetricsSources == nil {
		return nil
	}

	if odigosConfiguration.MetricsSources.OdigosOwnMetrics != nil && odigosConfiguration.MetricsSources.OdigosOwnMetrics.Interval != "" {
		_, err := time.ParseDuration(odigosConfiguration.MetricsSources.OdigosOwnMetrics.Interval)
		if err != nil {
			return fmt.Errorf("failed to parse odigos own metrics interval: %w", err)
		}
	}

	if odigosConfiguration.MetricsSources.KubeletStats != nil && odigosConfiguration.MetricsSources.KubeletStats.Interval != "" {
		_, err := time.ParseDuration(odigosConfiguration.MetricsSources.KubeletStats.Interval)
		if err != nil {
			return fmt.Errorf("failed to parse kubelet stats interval: %w", err)
		}
	}

	if odigosConfiguration.MetricsSources.HostMetrics != nil && odigosConfiguration.MetricsSources.HostMetrics.Interval != "" {
		_, err := time.ParseDuration(odigosConfiguration.MetricsSources.HostMetrics.Interval)
		if err != nil {
			return fmt.Errorf("failed to parse host metrics interval: %w", err)
		}
	}

	if odigosConfiguration.MetricsSources.SpanMetrics != nil {
		if odigosConfiguration.MetricsSources.SpanMetrics.Interval == "" {
			odigosConfiguration.MetricsSources.SpanMetrics.Interval = "60s"
		} else {
			_, err := time.ParseDuration(odigosConfiguration.MetricsSources.SpanMetrics.Interval)
			if err != nil {
				return fmt.Errorf("failed to parse span metrics interval: %w", err)
			}
		}
	}

	return nil
}
