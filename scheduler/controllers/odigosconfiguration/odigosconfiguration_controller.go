package odigosconfiguration

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

	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
	odigosConfiguration.IgnoredNamespaces = append(odigosConfiguration.IgnoredNamespaces, env.GetCurrentNamespace())

	// make sure the default ignored containers are always present
	odigosConfiguration.IgnoredContainers = mergeIgnoredItemLists(odigosConfiguration.IgnoredContainers, k8sconsts.DefaultIgnoredContainers)

	modifyConfigWithEffectiveProfiles(effectiveProfiles, &odigosConfiguration)
	odigosConfiguration.Profiles = effectiveProfiles

	// if none of the profiles set sizing for collectors, use size_s as default, so the values are never nil
	// if the values were already set (by user or profile) this is a no-op
	sizing.SizeSProfile.ModifyConfigFunc(&odigosConfiguration)

	// TODO: revisit doing this here, might be nicer to maintain in a more generic way
	// and have it on the config object itself.
	// I want to preserve that user input (specific request or empty), and persist the resolved value in effective config.
	resolveMountMethod(&odigosConfiguration)
	resolveEnvInjectionMethod(&odigosConfiguration)

	// Handle the Go offsets CronJob
	err = r.handleGoOffsetsCronJob(ctx, &odigosConfiguration, env.GetCurrentNamespace())
	if err != nil {
		return ctrl.Result{}, err
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

func (r *odigosConfigurationController) handleGoOffsetsCronJob(ctx context.Context, config *common.OdigosConfiguration, ns string) error {
	// Get the Kubernetes server version to determine the API version to use for the CronJob
	cfg := ctrl.GetConfigOrDie()
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return fmt.Errorf("failed to get Kubernetes server version: %v", err)
	}
	verInfo, err := discoveryClient.ServerVersion()
	if err != nil {
		return fmt.Errorf("failed to get Kubernetes server version: %v", err)
	}
	apiVersion := "batch/v1beta1"
	if verInfo.Minor >= "21" {
		apiVersion = "batch/v1"
	}

	if config.GoAutoOffsetsCron == "" {
		return deleteCronJob(ctx, r.Client, ns, apiVersion)
	}

	typeMeta := metav1.TypeMeta{
		Kind:       "CronJob",
		APIVersion: apiVersion,
	}
	objectMeta := metav1.ObjectMeta{
		Name:      k8sconsts.OffsetCronJobName,
		Namespace: ns,
		Labels: map[string]string{
			k8sconsts.OdigosSystemLabelKey: k8sconsts.OdigosSystemLabelValue,
		},
	}
	template := corev1.PodTemplateSpec{
		Spec: corev1.PodSpec{
			ServiceAccountName: k8sconsts.SchedulerServiceAccountName,
			Containers: []corev1.Container{
				{
					Name:  k8sconsts.CliImageName,
					Image: fmt.Sprintf("%s/%s:%s", config.ImagePrefix, k8sconsts.CliImageName, r.OdigosVersion),
					Args:  []string{"pro", "update-offsets"},
				},
			},
			RestartPolicy: corev1.RestartPolicyNever,
		},
	}

	if apiVersion == "batch/v1beta1" {
		cronJob := &batchv1beta1.CronJob{
			TypeMeta:   typeMeta,
			ObjectMeta: objectMeta,
			Spec: batchv1beta1.CronJobSpec{
				Schedule: config.GoAutoOffsetsCron,
				JobTemplate: batchv1beta1.JobTemplateSpec{
					Spec: batchv1.JobSpec{
						Template: template,
					},
				},
			},
		}
		return applyCronJob(ctx, r.Client, ns, cronJob, config)
	}
	cronJob := &batchv1.CronJob{
		TypeMeta:   typeMeta,
		ObjectMeta: objectMeta,
		Spec: batchv1.CronJobSpec{
			Schedule: config.GoAutoOffsetsCron,
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: template,
				},
			},
		},
	}
	return applyCronJob(ctx, r.Client, ns, cronJob, config)
}

func deleteCronJob(ctx context.Context, kubeClient client.Client, ns string, apiVersion string) error {
	var cronJob client.Object
	if apiVersion == "batch/v1" {
		cronJob = &batchv1.CronJob{}
	} else {
		cronJob = &batchv1beta1.CronJob{}
	}
	err := kubeClient.Get(ctx, types.NamespacedName{Namespace: ns, Name: k8sconsts.OffsetCronJobName}, cronJob)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	err = kubeClient.Delete(ctx, cronJob)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete go offsets CronJob: %v", err)
		}
		return nil
	}
	return nil
}

func applyCronJob(ctx context.Context, kubeClient client.Client, ns string, cronJob client.Object, config *common.OdigosConfiguration) error {
	// Apply the CronJob
	objApplyBytes, err := yaml.Marshal(cronJob)
	if err != nil {
		return err
	}

	err = kubeClient.Patch(ctx, cronJob, client.RawPatch(types.ApplyYAMLPatchType, objApplyBytes), client.ForceOwnership, client.FieldOwner("scheduler-odigosconfig"))
	if err != nil {
		return fmt.Errorf("failed to apply go offsets CronJob: %v", err)
	}

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
