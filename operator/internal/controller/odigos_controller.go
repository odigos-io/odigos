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
	"os"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/cli/pkg/autodetect"
	cmdcontext "github.com/odigos-io/odigos/cli/pkg/cmd_context"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/common/consts"
	operatorv1alpha1 "github.com/odigos-io/odigos/operator/api/v1alpha1"
)

const (
	operatorFinalizer = "operator.odigos.io/odigos-finalizer"

	odigosInstalledCondition = "OdigosInstalled"
)

// OdigosReconciler reconciles a Odigos object
type OdigosReconciler struct {
	client.Client
	Scheme *runtime.Scheme
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
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;delete
// +kubebuilder:rbac:groups="",resources=pods/status,verbs=get
// +kubebuilder:rbac:groups="",resources=pods/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=apps,resources=deployments;replicasets;daemonsets;statefulsets,verbs=get;list;watch;create;update;patch;delete;deletecollection
// +kubebuilder:rbac:groups=apps,resources=deployments/finalizers;replicasets/finalizers;daemonsets/finalizers;statefulsets/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments/status;daemonsets/status;statefulsets/status,verbs=get
// +kubebuilder:rbac:groups=apps,resources=deployments/scale,verbs=get;update;patch
// +kubebuilder:rbac:groups=batch,resources=cronjobs,verbs=get;list;create;update;patch;watch;delete
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=create;list;watch;delete;get
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles;clusterrolebindings;roles;rolebindings,verbs=get;list;watch;create;update;patch;delete;deletecollection
// +kubebuilder:rbac:groups=security.openshift.io,resources=securitycontextconstraints,verbs=use
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=get;list;watch;create;update;patch;delete;deletecollection
// +kubebuilder:rbac:groups=admissionregistration.k8s.io,resources=mutatingwebhookconfigurations;validatingwebhookconfigurations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs=create;patch;update;delete;get
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
	kubeClient := &kube.Client{
		Interface: clientset,
		Clientset: clientset,
		Config:    k8sConfig,
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

	// Use Helm to uninstall Odigos
	err := helmUninstall(kubeClient.Config, ns, logger)
	if err != nil {
		logger.Error(err, "failed to uninstall Odigos via Helm")
		return ctrl.Result{}, err
	}

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

	// Check if Odigos already installed (another Odigos CR exists)
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

	// Get version from environment variable
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
			Message:            "could not determine Odigos version from ODIGOS_VERSION environment variable",
			ObservedGeneration: odigos.GetGeneration(),
		})
		logger.Info("could not determine Odigos version from ODIGOS_VERSION environment variable")
		return ctrl.Result{}, r.Status().Update(ctx, odigos)
	}

	// Detect cluster type (OpenShift, etc.)
	openshiftEnabled := false
	details := autodetect.GetK8SClusterDetails(ctx, "", "", kubeClient)
	if details.Kind == autodetect.KindOpenShift {
		logger.Info("Detected OpenShift cluster, enabling required configuration")
		openshiftEnabled = true
	}
	ctx = cmdcontext.ContextWithClusterDetails(ctx, details)

	// Check Kubernetes version
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
			logger.Info("odigos requires minimum Kubernetes version", "required", k8sconsts.MinK8SVersionForInstallation.String(), "found", k8sVersion.String())
			return ctrl.Result{}, r.Status().Update(ctx, odigos)
		}
		logger.Info("Detected Kubernetes version", "version", k8sVersion.String())
	}

	logger.Info("Installing Odigos", "version", version, "namespace", ns)

	// Use Helm to install/upgrade Odigos
	err = helmInstall(kubeClient.Config, ns, odigos, version, openshiftEnabled, logger)
	if err != nil {
		meta.SetStatusCondition(&odigos.Status.Conditions, metav1.Condition{
			Type:               odigosInstalledCondition,
			Status:             metav1.ConditionFalse,
			Reason:             "HelmInstallFailed",
			Message:            "failed to install Odigos via Helm: " + err.Error(),
			ObservedGeneration: odigos.GetGeneration(),
		})
		logger.Error(err, "failed to install Odigos via Helm")
		return ctrl.Result{}, r.Status().Update(ctx, odigos)
	}

	logger.Info("Odigos installed successfully")
	meta.SetStatusCondition(&odigos.Status.Conditions, metav1.Condition{
		Type:               odigosInstalledCondition,
		Status:             metav1.ConditionTrue,
		Reason:             "OdigosComponentsInstalled",
		Message:            "All Odigos components successfully installed via Helm",
		ObservedGeneration: odigos.GetGeneration(),
	})

	return ctrl.Result{}, r.Status().Update(ctx, odigos)
}
