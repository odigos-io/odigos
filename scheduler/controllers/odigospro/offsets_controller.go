package odigospro

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
)

type odigosproOffsetsController struct {
	client.Client
	OdigosVersion string
}

func (r *odigosproOffsetsController) Reconcile(ctx context.Context, _ ctrl.Request) (ctrl.Result, error) {
	logger := ctrl.LoggerFrom(ctx)
	var configMap corev1.ConfigMap
	odigosNs := env.GetCurrentNamespace()

	err := r.Client.Get(ctx, types.NamespacedName{Namespace: odigosNs, Name: consts.OdigosConfigurationName}, &configMap)
	if err != nil {
		return ctrl.Result{}, err
	}

	odigosConfiguration := &common.OdigosConfiguration{}
	err = yaml.Unmarshal([]byte(configMap.Data[consts.OdigosConfigurationFileName]), odigosConfiguration)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Get the Kubernetes server version to determine the API version to use for the CronJob
	cfg := ctrl.GetConfigOrDie()
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get Kubernetes server version: %v", err)
	}
	verInfo, err := discoveryClient.ServerVersion()
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get Kubernetes server version: %v", err)
	}
	apiVersion := "batch/v1beta1"
	if verInfo.Minor >= "21" {
		apiVersion = "batch/v1"
	}

	// Determine the mode to use (default to "direct" if not specified)
	mode := k8sconsts.OffsetCronJobMode(odigosConfiguration.GoAutoOffsetsMode)
	if mode == "" {
		mode = k8sconsts.OffsetCronJobModeDirect
	}

	// Validate the mode
	if !mode.IsValid() {
		return ctrl.Result{}, fmt.Errorf("invalid go-auto-offsets-mode: %s. Must be one of: %s, %s, %s",
			mode, k8sconsts.OffsetCronJobModeDirect, k8sconsts.OffsetCronJobModeImage, k8sconsts.OffsetCronJobModeOff)
	}

	if odigosConfiguration.GoAutoOffsetsCron == "" || mode == k8sconsts.OffsetCronJobModeOff {
		return ctrl.Result{}, deleteCronJob(ctx, r.Client, odigosNs, apiVersion)
	}

	tier, err := getCurrentOdigosTier(ctx, r.Client, odigosNs)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get current Odigos tier: %v", err)
	}
	if tier == common.CommunityOdigosTier {
		logger.Error(fmt.Errorf("custom offsets support is only available in Odigos pro tier"), "could not reconcile go offsets cron job")
		return ctrl.Result{}, nil
	}

	// Determine image name and command based on mode
	var imageName string
	var command []string

	switch mode {
	case k8sconsts.OffsetCronJobModeDirect:
		imageName = k8sconsts.CliImageName
		command = []string{"pro", "update-offsets"}
	case k8sconsts.OffsetCronJobModeImage:
		imageName = k8sconsts.CliOffsetsImageName
		command = []string{"pro", "update-offsets", "--from-file", "/odigos/offset_results_min.json"}
	}

	typeMeta := metav1.TypeMeta{
		Kind:       "CronJob",
		APIVersion: apiVersion,
	}
	objectMeta := metav1.ObjectMeta{
		Name:      k8sconsts.OffsetCronJobName,
		Namespace: odigosNs,
		Labels: map[string]string{
			k8sconsts.OdigosSystemLabelKey: k8sconsts.OdigosSystemLabelValue,
		},
	}
	template := corev1.PodTemplateSpec{
		Spec: corev1.PodSpec{
			ServiceAccountName: k8sconsts.SchedulerServiceAccountName,
			Containers: []corev1.Container{
				{
					Name:  imageName,
					Image: fmt.Sprintf("%s/%s:%s", odigosConfiguration.ImagePrefix, imageName, r.OdigosVersion),
					Args:  command,
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
				Schedule: odigosConfiguration.GoAutoOffsetsCron,
				JobTemplate: batchv1beta1.JobTemplateSpec{
					Spec: batchv1.JobSpec{
						Template: template,
					},
				},
			},
		}
		return ctrl.Result{}, applyCronJob(ctx, r.Client, odigosNs, cronJob, odigosConfiguration)
	}
	cronJob := &batchv1.CronJob{
		TypeMeta:   typeMeta,
		ObjectMeta: objectMeta,
		Spec: batchv1.CronJobSpec{
			Schedule: odigosConfiguration.GoAutoOffsetsCron,
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: template,
				},
			},
		},
	}
	return ctrl.Result{}, applyCronJob(ctx, r.Client, odigosNs, cronJob, odigosConfiguration)
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

func getCurrentOdigosTier(ctx context.Context, kubeClient client.Client, ns string) (common.OdigosTier, error) {
	secret := &corev1.Secret{}
	err := kubeClient.Get(ctx, client.ObjectKey{Namespace: ns, Name: k8sconsts.OdigosProSecretName}, secret)
	if apierrors.IsNotFound(err) {
		return common.CommunityOdigosTier, nil
	}
	if err != nil {
		return common.CommunityOdigosTier, err
	}

	if _, exists := secret.Data[k8sconsts.OdigosCloudApiKeySecretKey]; exists {
		return common.CloudOdigosTier, nil
	}
	if _, exists := secret.Data[k8sconsts.OdigosOnpremTokenSecretKey]; exists {
		return common.OnPremOdigosTier, nil
	}
	return common.CommunityOdigosTier, nil
}
