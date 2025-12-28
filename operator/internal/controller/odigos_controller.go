/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"os"

	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/odigos-io/odigos/cli/cmd"
	"github.com/odigos-io/odigos/cli/cmd/resources"
	"github.com/odigos-io/odigos/cli/cmd/resources/resourcemanager"
	"github.com/odigos-io/odigos/cli/pkg/autodetect"
	cmdcontext "github.com/odigos-io/odigos/cli/pkg/cmd_context"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/installationmethod"

	"github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned/typed/odigos/v1alpha1"
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
	operatorv1alpha1 "github.com/odigos-io/odigos/operator/api/v1alpha1"
)

const (
	operatorFinalizer = "operator.odigos.io/odigos-finalizer"

	odigosInstalledCondition = "OdigosInstalled"
	odigosUpgradeCondition   = "OdigosUpgraded"
)

// OdigosReconciler reconciles a Odigos object
type OdigosReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// relatedImageEnvVars is a reverse-lookup map to quickly find the environment variable
// for component images when openshift is enabled
var relatedImageEnvVars = map[string]string{
	k8sconsts.AutoScalerImageName: "RELATED_IMAGE_AUTOSCALER",
	k8sconsts.AutoScalerImageUBI9: "RELATED_IMAGE_AUTOSCALER",

	k8sconsts.OdigosClusterCollectorImage:     "RELATED_IMAGE_COLLECTOR",
	k8sconsts.OdigosClusterCollectorImageUBI9: "RELATED_IMAGE_COLLECTOR",

	k8sconsts.InstrumentorImage:               "RELATED_IMAGE_INSTRUMENTOR",
	k8sconsts.InstrumentorImageUBI9:           "RELATED_IMAGE_INSTRUMENTOR",
	k8sconsts.InstrumentorEnterpriseImage:     "RELATED_IMAGE_ENTERPRISE_INSTRUMENTOR",
	k8sconsts.InstrumentorEnterpriseImageUBI9: "RELATED_IMAGE_ENTERPRISE_INSTRUMENTOR",

	k8sconsts.UIImage:     "RELATED_IMAGE_FRONTEND",
	k8sconsts.UIImageUBI9: "RELATED_IMAGE_FRONTEND",

	k8sconsts.OdigletImageName:           "RELATED_IMAGE_ODIGLET",
	k8sconsts.OdigletImageUBI9:           "RELATED_IMAGE_ODIGLET",
	k8sconsts.OdigletEnterpriseImageName: "RELATED_IMAGE_ENTERPRISE_ODIGLET",
	k8sconsts.OdigletEnterpriseImageUBI9: "RELATED_IMAGE_ENTERPRISE_ODIGLET",

	k8sconsts.SchedulerImage:     "RELATED_IMAGE_SCHEDULER",
	k8sconsts.SchedulerImageUBI9: "RELATED_IMAGE_SCHEDULER",
}

// +kubebuilder:rbac:groups=operator.odigos.io,resources=odigos,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=operator.odigos.io,resources=odigos/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=operator.odigos.io,resources=odigos/finalizers,verbs=update
// +kubebuilder:rbac:groups=actions.odigos.io,resources=*,verbs=get;list;watch;create;patch;update;delete;deletecollection
// +kubebuilder:rbac:groups=actions.odigos.io,resources=*/status,verbs=get;patch;update
// +kubebuilder:rbac:groups=odigos.io,resources=instrumentationrules/status,verbs=get;patch;update
// +kubebuilder:rbac:groups=odigos.io,resources=*,verbs=*
// +kubebuilder:rbac:groups=odigos.io,resources=destinations/status;instrumentationinstances/status;instrumentationconfigs/status;collectorsgroups/status,verbs=get;list;watch;patch;update
// +kubebuilder:rbac:groups=odigos.io,resources=sources/finalizers,verbs=update
// +kubebuilder:rbac:groups=odigos.io,resources=collectorsgroups/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete;deletecollection
// +kubebuilder:rbac:groups="",resources=configmaps;endpoints;secrets,verbs=get;list;watch;create;update;delete;patch;deletecollection
// +kubebuilder:rbac:groups="",resources=configmaps/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=create;get;list;watch;patch;delete
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;patch
// +kubebuilder:rbac:groups="",resources=nodes,verbs=get;list;watch;patch;update
// +kubebuilder:rbac:groups="",resources=nodes/proxy,verbs=get;list
// +kubebuilder:rbac:groups="",resources=nodes/stats,verbs=get;list
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;delete
// +kubebuilder:rbac:groups="",resources=pods/status,verbs=get
// +kubebuilder:rbac:groups="",resources=pods/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=apps,resources=deployments;replicasets;daemonsets;statefulsets,verbs=get;list;watch;create;update;patch;delete;deletecollection
// +kubebuilder:rbac:groups=apps,resources=deployments/finalizers;replicasets/finalizers;daemonsets/finalizers;statefulsets/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments/status;daemonsets/status;statefulsets/status,verbs=get
// +kubebuilder:rbac:groups=batch,resources=cronjobs,verbs=get;list;create;update;patch;watch;delete
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=create
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles;clusterrolebindings;roles;rolebindings,verbs=get;list;watch;create;update;patch;delete;deletecollection
// +kubebuilder:rbac:groups=security.openshift.io,resources=securitycontextconstraints,verbs=use
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=get;list;watch;create;update;patch;delete;deletecollection
// +kubebuilder:rbac:groups=admissionregistration.k8s.io,resources=mutatingwebhookconfigurations;validatingwebhookconfigurations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs=create;patch;update;delete
// +kubebuilder:rbac:groups=policy,resources=podsecuritypolicies,resourceNames=privileged,verbs=use
// +kubebuilder:rbac:groups=apps.openshift.io,resources=deploymentconfigs;deploymentconfigs/finalizers,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=apiregistration.k8s.io,resources=apiservices,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups=argoproj.io,resources=rollouts,verbs=get;list;watch;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *OdigosReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	odigos := &operatorv1alpha1.Odigos{}
	err := r.Client.Get(ctx, req.NamespacedName, odigos)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	k8sConfig, err := config.GetConfig()
	if err != nil {
		logger.Error(err, "unable to get k8s config", "controller", "Odigos")
		os.Exit(1)
	}
	clientset, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		logger.Error(err, "unable to get k8s clientset", "controller", "Odigos")
		os.Exit(1)
	}
	dynamicClient, err := dynamic.NewForConfig(k8sConfig)
	if err != nil {
		logger.Error(err, "unable to get k8s dynamic client", "controller", "Odigos")
		os.Exit(1)
	}
	extendClientset, err := apiextensionsclient.NewForConfig(k8sConfig)
	if err != nil {
		logger.Error(err, "unable to get k8s extendClientset", "controller", "Odigos")
		os.Exit(1)
	}

	odigosClient, err := v1alpha1.NewForConfig(k8sConfig)
	if err != nil {
		logger.Error(err, "unable to get Odigos client", "controller", "Odigos")
		os.Exit(1)
	}
	kubeClient := &kube.Client{
		Interface:     clientset,
		Clientset:     clientset,
		Dynamic:       dynamicClient,
		ApiExtensions: extendClientset,
		OdigosClient:  odigosClient,
		Config:        k8sConfig,
	}

	// Store original object for patch operations
	originalOdigos := odigos.DeepCopy()

	if odigos.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.install(ctx, kubeClient, odigos, originalOdigos)
	} else {
		return r.uninstall(ctx, kubeClient, odigos, originalOdigos)
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *OdigosReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.Odigos{}).
		WithEventFilter(&predicate.GenerationChangedPredicate{}).
		Complete(r)
}

func (r *OdigosReconciler) uninstall(ctx context.Context, kubeClient *kube.Client, odigos *operatorv1alpha1.Odigos, originalOdigos *operatorv1alpha1.Odigos) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	ns := odigos.GetNamespace()
	cmd.UninstallOdigosResources(ctx, kubeClient, ns)
	cmd.UninstallClusterResources(ctx, kubeClient, ns)

	if controllerutil.ContainsFinalizer(odigos, operatorFinalizer) {
		controllerutil.RemoveFinalizer(odigos, operatorFinalizer)
		if err := r.Patch(ctx, odigos, client.MergeFrom(originalOdigos)); err != nil {
			return ctrl.Result{}, err
		}
	}

	logger.Info("Odigos uninstalled")
	return ctrl.Result{}, nil
}

// install Odigos based on the config passed in odigos
func (r *OdigosReconciler) install(ctx context.Context, kubeClient *kube.Client, odigos *operatorv1alpha1.Odigos, originalOdigos *operatorv1alpha1.Odigos) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if !controllerutil.ContainsFinalizer(odigos, operatorFinalizer) {
		controllerutil.AddFinalizer(odigos, operatorFinalizer)
		if err := r.Patch(ctx, odigos, client.MergeFrom(originalOdigos)); err != nil {
			return ctrl.Result{}, err
		}
	}

	ns := odigos.GetNamespace()
	// Check if Odigos already installed
	odigosList := &operatorv1alpha1.OdigosList{}
	err := r.Client.List(ctx, odigosList, client.InNamespace(ns))
	if err != nil {
		return ctrl.Result{}, err
	}
	for _, o := range odigosList.Items {
		if o.GetName() != odigos.GetName() {
			meta.SetStatusCondition(&odigos.Status.Conditions, metav1.Condition{
				Type:               odigosInstalledCondition,
				Status:             metav1.ConditionFalse,
				Reason:             "AlreadyInstalled",
				Message:            "odigos is already installed in namespace " + ns,
				ObservedGeneration: odigos.GetGeneration(),
			})
			logger.Info("odigos is already installed in namespace", "namespace", ns)
			return ctrl.Result{}, r.Status().Update(ctx, odigos)
		}
	}

	version := os.Getenv(consts.OdigosVersionEnvVarName)
	if len(version) > 0 {
		if string(version[0]) != "v" {
			version = "v" + version
		}
	} else {
		meta.SetStatusCondition(&odigos.Status.Conditions, metav1.Condition{
			Type:               odigosInstalledCondition,
			Status:             metav1.ConditionFalse,
			Reason:             "InvalidOdigosVersion",
			Message:            "could not determine Odigos version from odigos-version configmap or ODIGOS_VERSION environment variable",
			ObservedGeneration: odigos.GetGeneration(),
		})
		logger.Info("could not determine Odigos version from odigos-version configmap or ODIGOS_VERSION environment variable")
		return ctrl.Result{}, r.Status().Update(ctx, odigos)
	}

	openshiftEnabled := false
	details := autodetect.GetK8SClusterDetails(ctx, "", "", kubeClient)
	if details.Kind == autodetect.KindOpenShift {
		logger.Info("Detected OpenShift cluster, enabling required configuration")
		openshiftEnabled = true
	}
	ctx = cmdcontext.ContextWithClusterDetails(ctx, details)

	k8sVersion := cmdcontext.K8SVersionFromContext(ctx)
	if k8sVersion != nil {
		if k8sVersion.LessThan(k8sconsts.MinK8SVersionForInstallation) {
			meta.SetStatusCondition(&odigos.Status.Conditions, metav1.Condition{
				Type:               odigosInstalledCondition,
				Status:             metav1.ConditionFalse,
				Reason:             "InvalidKubernetesVersion",
				Message:            "odigos requires Kubernetes version " + k8sconsts.MinK8SVersionForInstallation.String() + " but found " + k8sVersion.String(),
				ObservedGeneration: odigos.GetGeneration(),
			})
			logger.Info("odigos requires Kubernetes version " + k8sconsts.MinK8SVersionForInstallation.String() + " but found " + k8sVersion.String())
			return ctrl.Result{}, r.Status().Update(ctx, odigos)
		}
		logger.Info(fmt.Sprintf("Detected cluster: Kubernetes version: %s\n", k8sVersion.String()))
	}

	var odigosProToken string
	odigosTier := common.CommunityOdigosTier
	if odigos.Spec.OnPremToken != "" {
		odigosTier = common.OnPremOdigosTier
		odigosProToken = odigos.Spec.OnPremToken
	}

	// validate user input profiles against available profiles
	err = cmd.ValidateUserInputProfiles(odigosTier)
	if err != nil {
		meta.SetStatusCondition(&odigos.Status.Conditions, metav1.Condition{
			Type:               odigosInstalledCondition,
			Status:             metav1.ConditionFalse,
			Reason:             "InvalidProfile",
			Message:            err.Error(),
			ObservedGeneration: odigos.GetGeneration(),
		})
		logger.Error(err, "unable to validate input profile")
		return ctrl.Result{}, r.Status().Update(ctx, odigos)
	}

	selectedProfiles := []common.ProfileName{}
	for _, profile := range odigos.Spec.Profiles {
		selectedProfiles = append(selectedProfiles, common.ProfileName(profile))
	}

	odigosConfiguration := common.OdigosConfiguration{}
	upgrade := false
	config, err := resources.GetCurrentConfig(ctx, kubeClient, ns)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			meta.SetStatusCondition(&odigos.Status.Conditions, metav1.Condition{
				Type:               odigosInstalledCondition,
				Status:             metav1.ConditionFalse,
				Reason:             "OdigosConfigErr",
				Message:            "Error getting current Odigos config",
				ObservedGeneration: odigos.GetGeneration(),
			})
			logger.Error(err, "error getting current Odigos config")
			return ctrl.Result{}, r.Status().Update(ctx, odigos)
		}

		odigosConfiguration = common.OdigosConfiguration{
			ConfigVersion: 1,
		}
	} else {
		odigosConfiguration = *config
		odigosConfiguration.ConfigVersion = odigosConfiguration.ConfigVersion + 1
		upgrade = true
	}

	nodeSelector := make(map[string]string)
	if odigos.Spec.NodeSelector != nil {
		nodeSelector = odigos.Spec.NodeSelector
	}

	odigosConfiguration.TelemetryEnabled = odigos.Spec.TelemetryEnabled
	odigosConfiguration.OpenshiftEnabled = openshiftEnabled
	odigosConfiguration.IgnoredNamespaces = odigos.Spec.IgnoredNamespaces
	odigosConfiguration.IgnoredContainers = odigos.Spec.IgnoredContainers
	odigosConfiguration.SkipWebhookIssuerCreation = odigos.Spec.SkipWebhookIssuerCreation
	odigosConfiguration.Psp = odigos.Spec.PodSecurityPolicy
	odigosConfiguration.ImagePrefix = odigos.Spec.ImagePrefix
	odigosConfiguration.Profiles = odigos.Spec.Profiles

	// "normal" is deprecated. Kept here for backwards compatibility with operator CRD.
	if odigos.Spec.UIMode == "normal" {
		odigosConfiguration.UiMode = common.UiModeDefault
	} else {
		odigosConfiguration.UiMode = common.UiMode(odigos.Spec.UIMode)
	}
	odigosConfiguration.NodeSelector = nodeSelector
	agentEnvVarsInjectionMethod := odigos.Spec.AgentEnvVarsInjectionMethod
	if agentEnvVarsInjectionMethod == "" {
		agentEnvVarsInjectionMethod = common.PodManifestEnvInjectionMethod
	}
	odigosConfiguration.AgentEnvVarsInjectionMethod = &agentEnvVarsInjectionMethod

	ownerReference := metav1.OwnerReference{
		APIVersion: odigos.APIVersion,
		Kind:       odigos.Kind,
		Name:       odigos.GetName(),
		UID:        odigos.GetUID(),
	}
	managerOpts := resourcemanager.ManagerOpts{
		OwnerReferences: []metav1.OwnerReference{ownerReference},
	}
	imageReferences := cmd.GetImageReferences(odigosTier, openshiftEnabled)
	if openshiftEnabled {
		imageReferences.AutoscalerImage = os.Getenv(relatedImageEnvVars[imageReferences.AutoscalerImage])
		imageReferences.CollectorImage = os.Getenv(relatedImageEnvVars[imageReferences.CollectorImage])
		imageReferences.UIImage = os.Getenv(relatedImageEnvVars[imageReferences.UIImage])
		imageReferences.InstrumentorImage = os.Getenv(relatedImageEnvVars[imageReferences.InstrumentorImage])
		imageReferences.OdigletImage = os.Getenv(relatedImageEnvVars[imageReferences.OdigletImage])
		imageReferences.SchedulerImage = os.Getenv(relatedImageEnvVars[imageReferences.SchedulerImage])
	}
	managerOpts.ImageReferences = imageReferences

	defaultMountMethod := common.K8sVirtualDeviceMountMethod
	if len(odigos.Spec.MountMethod) == 0 {
		odigosConfiguration.MountMethod = &defaultMountMethod
	} else {
		switch odigos.Spec.MountMethod {
		case common.K8sHostPathMountMethod:
		case common.K8sVirtualDeviceMountMethod:
		default:
			meta.SetStatusCondition(&odigos.Status.Conditions, metav1.Condition{
				Type:               odigosInstalledCondition,
				Status:             metav1.ConditionFalse,
				Reason:             "OdigosConfigErr",
				Message:            "Invalid mount method " + string(odigos.Spec.MountMethod),
				ObservedGeneration: odigos.GetGeneration(),
			})
			logger.Error(fmt.Errorf("invalid mount method (valid values: %s, %s)", common.K8sHostPathMountMethod, common.K8sVirtualDeviceMountMethod), "mountMethod", odigos.Spec.MountMethod)
			return ctrl.Result{}, r.Status().Update(ctx, odigos)
		}
		odigosConfiguration.MountMethod = &odigos.Spec.MountMethod
	}

	if !openshiftEnabled {
		if odigos.Spec.ImagePrefix == "" {
			odigosConfiguration.ImagePrefix = k8sconsts.OdigosImagePrefix
		}
	}

	logger.Info("Installing Odigos version " + version + " in namespace " + ns)

	resourceManagers := resources.CreateResourceManagers(kubeClient, ns, odigosTier, &odigosProToken, &odigosConfiguration, version, installationmethod.K8sInstallationMethodOdigosOperator, managerOpts)
	err = resources.ApplyResourceManagers(ctx, kubeClient, resourceManagers, "Creating")
	if err != nil {
		return ctrl.Result{}, err
	}

	logger.Info("Odigos installed")
	meta.SetStatusCondition(&odigos.Status.Conditions, metav1.Condition{
		Type:               odigosInstalledCondition,
		Status:             metav1.ConditionTrue,
		Reason:             "OdigosComponentsInstalled",
		Message:            "All Odigos components successfully installed",
		ObservedGeneration: odigos.GetGeneration(),
	})

	if upgrade {
		err = resources.DeleteOldOdigosSystemObjects(ctx, kubeClient, ns, &odigosConfiguration)
		if err != nil {
			meta.SetStatusCondition(&odigos.Status.Conditions, metav1.Condition{
				Type:               odigosUpgradeCondition,
				Status:             metav1.ConditionFalse,
				Reason:             "OdigosUpgradeFailed",
				Message:            "error deleting old Odigos system objects: " + err.Error(),
				ObservedGeneration: odigos.GetGeneration(),
			})
			logger.Error(err, "error deleting old Odigos system objects")
			return ctrl.Result{}, r.Status().Update(ctx, odigos)
		}
		meta.SetStatusCondition(&odigos.Status.Conditions, metav1.Condition{
			Type:               odigosUpgradeCondition,
			Status:             metav1.ConditionTrue,
			Reason:             "OdigosUpgradeSucceeded",
			Message:            "successfully upgraded Odigos",
			ObservedGeneration: odigos.GetGeneration(),
		})
	}

	return ctrl.Result{}, r.Status().Update(ctx, odigos)
}
