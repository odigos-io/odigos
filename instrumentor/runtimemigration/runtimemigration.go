package runtimemigration

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/common/envOverwrite"
	k8scontainer "github.com/odigos-io/odigos/k8sutils/pkg/container"
	"github.com/odigos-io/odigos/k8sutils/pkg/envoverwrite"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type MigrationRunnable struct {
	KubeClient client.Client
	Logger     logr.Logger
}

func (m *MigrationRunnable) NeedLeaderElection() bool {
	// make sure we run it only from one instance of an instrumentor
	return true
}

func (m *MigrationRunnable) Start(ctx context.Context) error {

	var instrumentationConfigs v1alpha1.InstrumentationConfigList
	err := m.KubeClient.List(ctx, &instrumentationConfigs, &client.ListOptions{})
	if err != nil {
		m.Logger.Error(err, "Failed to list InstrumentationConfigs")
		return nil
	}

	workloadNamespaces := map[string]map[string]map[string]*v1alpha1.InstrumentationConfig{
		"Deployment":  make(map[string]map[string]*v1alpha1.InstrumentationConfig),
		"StatefulSet": make(map[string]map[string]*v1alpha1.InstrumentationConfig),
		"DaemonSet":   make(map[string]map[string]*v1alpha1.InstrumentationConfig),
	}
	// Example structure:
	// {
	//   "Deployment": {
	//       "default": {
	//           "frontend": *<InstrumentationConfig>,
	//       },
	//   },

	for _, item := range instrumentationConfigs.Items {

		IcName := item.GetName()
		IcNamespace := item.GetNamespace()
		workloadName, workloadType, err := workload.ExtractWorkloadInfoFromRuntimeObjectName(IcName)

		if err != nil {
			m.Logger.Error(err, "Failed to extract workload info from runtime object name")
			continue
		}

		if _, exists := workloadNamespaces[string(workloadType)][IcNamespace]; !exists {
			workloadNamespaces[string(workloadType)][IcNamespace] = make(map[string]*v1alpha1.InstrumentationConfig)
		}

		// Save workloadName and the corresponding InstrumentationConfig reference
		workloadNamespaces[string(workloadType)][IcNamespace][workloadName] = &item
	}
	for workloadType, namespaces := range workloadNamespaces {
		switch workloadType {
		case "Deployment":
			if err := m.fetchAndProcessDeployments(ctx, m.KubeClient, namespaces); err != nil {
				m.Logger.Error(err, "Failed to fetch and process deployments")
				return nil
			}
		case "StatefulSet":
			if err := m.fetchAndProcessStatefulSets(ctx, m.KubeClient, namespaces); err != nil {
				m.Logger.Error(err, "Failed to fetch and process statefulsets")
				return nil
			}
		case "DaemonSet":
			if err := m.fetchAndProcessDaemonSets(ctx, m.KubeClient, namespaces); err != nil {
				m.Logger.Error(err, "Failed to fetch and process daemonsets")
				return nil
			}
		default:
			fmt.Printf("Unknown workload type: %s\n", workloadType)
		}
	}

	return nil
}

func odigosOriginalAnnotationFound(annotations map[string]string) bool {
	if annotations == nil {
		return false
	}
	_, annotationFound := annotations[consts.ManifestEnvOriginalValAnnotation]
	return annotationFound
}

func (m *MigrationRunnable) fetchAndProcessDeployments(ctx context.Context, kubeClient client.Client, namespaces map[string]map[string]*v1alpha1.InstrumentationConfig) error {
	for namespace, workloadNames := range namespaces {
		var deployments appsv1.DeploymentList
		err := kubeClient.List(ctx, &deployments, &client.ListOptions{Namespace: namespace})
		if err != nil {
			return fmt.Errorf("failed to list deployments in namespace %s: %v", namespace, err)
		}

		for _, dep := range deployments.Items {

			// Checking if the deployment is in the list of deployments that need to be processed
			if !contains(workloadNames, dep.Name) {
				continue
			}

			// Retry logic for handling conflicts
			err := wait.ExponentialBackoff(wait.Backoff{
				Duration: 100 * time.Millisecond, // Initial wait time
				Factor:   2.0,                    // Exponential factor
				Jitter:   0.1,                    // Add randomness
				Steps:    5,                      // Max retries
			}, func() (bool, error) {
				err := m.handleSingleWorkload(ctx, &dep)
				if err != nil {
					m.Logger.Info("Error during env migration, retrying", "Deployment", dep.Name)
					return false, nil // Retry
				}
				return true, nil // Success, stop retrying
			})

			if err != nil {
				m.Logger.Error(err, "Failed to handle deployment with retries", "Name", dep.GetName(), "Namespace", dep.GetNamespace())
			}
		}
	}
	return nil
}

func (m *MigrationRunnable) fetchAndProcessStatefulSets(ctx context.Context, kubeClient client.Client, namespaces map[string]map[string]*v1alpha1.InstrumentationConfig) error {
	for namespace, workloadNames := range namespaces {
		var statefulSets appsv1.StatefulSetList
		err := kubeClient.List(ctx, &statefulSets, &client.ListOptions{Namespace: namespace})
		if err != nil {
			return fmt.Errorf("failed to list statefulsets in namespace %s: %v", namespace, err)
		}

		for _, sts := range statefulSets.Items {

			// Checking if the statefulset is in the list of statefulsets that need to be processed
			if !contains(workloadNames, sts.Name) {
				continue
			}

			// Retry logic for handling conflicts
			err := wait.ExponentialBackoff(wait.Backoff{
				Duration: 100 * time.Millisecond, // Initial wait time
				Factor:   2.0,                    // Exponential factor
				Jitter:   0.1,                    // Add randomness
				Steps:    5,                      // Max retries
			}, func() (bool, error) {
				err := m.handleSingleWorkload(ctx, &sts)
				if err != nil {
					m.Logger.Info("Error during env migration, retrying", "Deployment", sts.Name)
					return false, nil // Retry
				}
				return true, nil // Success, stop retrying
			})

			if err != nil {
				m.Logger.Error(err, "Failed to handle deployment with retries", "Name", sts.GetName(), "Namespace", sts.GetNamespace())
			}

		}
	}
	return nil
}

func (m *MigrationRunnable) fetchAndProcessDaemonSets(ctx context.Context, kubeClient client.Client, namespaces map[string]map[string]*v1alpha1.InstrumentationConfig) error {
	for namespace, workloadNames := range namespaces {
		var daemonSets appsv1.DaemonSetList
		err := kubeClient.List(ctx, &daemonSets, &client.ListOptions{Namespace: namespace})
		if err != nil {
			return fmt.Errorf("failed to list daemonsets in namespace %s: %v", namespace, err)
		}

		for _, ds := range daemonSets.Items {
			// Checking if the daemonset is in the list of daemonsets that need to be processed
			if !contains(workloadNames, ds.Name) {
				continue
			}

			// Retry logic for handling conflicts
			err := wait.ExponentialBackoff(wait.Backoff{
				Duration: 100 * time.Millisecond, // Initial wait time
				Factor:   2.0,                    // Exponential factor
				Jitter:   0.1,                    // Add randomness
				Steps:    5,                      // Max retries
			}, func() (bool, error) {
				err := m.handleSingleWorkload(ctx, &ds)
				if err != nil {
					m.Logger.Info("Error during env migration, retrying", "Deployment", ds.Name)
					return false, nil // Retry
				}
				return true, nil // Success, stop retrying
			})

			if err != nil {
				m.Logger.Error(err, "Failed to handle deployment with retries", "Name", ds.GetName(), "Namespace", ds.GetNamespace())
			}
		}
	}
	return nil
}
func handleContainerRuntimeDetailsUpdate(
	containerObject corev1.Container,
	originalWorkloadEnvVar *envoverwrite.OrigWorkloadEnvValues,
	runtimeDetailsByContainer []v1alpha1.RuntimeDetailsByContainer,
) (error, bool) {
	var needToUpdateWorkloadAnnotation bool

	for i := range runtimeDetailsByContainer {
		containerRuntimeDetails := &(runtimeDetailsByContainer)[i]

		// Find the relevant container in runtimeDetailsByContainer
		if containerRuntimeDetails.ContainerName != containerObject.Name {
			continue
		}

		annotationEnvVarsForContainer := originalWorkloadEnvVar.GetContainerStoredEnvs(containerObject.Name)

		// We identified a bug where Odigos-specific additions were included in the 'odigos.io/manifest-env-original-val',
		// leading to incorrect data in `instrumentationConfig`, particularly affecting the new runtime detection.
		// As a temporary workaround, we will remove the Odigos additions from the annotation and add the cleaned environment variables to `instrumentationConfig.EnvFromContainerRuntime`.
		// This approach resolves the issue as follows:
		// 1. If the environment variables originated from the Dockerfile, the webhook will use the value from `instrumentationConfig.EnvFromContainerRuntime`.
		// 2. If they originated from the manifest, the user's deployment will retain the original value, with the webhook appending any additional settings as needed.
		// This temporary fix addresses the current bug and will be removed once the `envOverwriter` logic is deprecated following the post-webhook injection release.
		shouldChangeUpdateStateToSucceeded := false
		for envKey, envValue := range annotationEnvVarsForContainer {

			if envValue == nil {
				continue
			}

			if strings.Contains(*envValue, "/var/odigos") {
				cleanedEnvValue := envOverwrite.CleanupEnvValueFromOdigosAdditions(envKey, *envValue)
				annotationEnvVarsForContainer[envKey] = &cleanedEnvValue
				needToUpdateWorkloadAnnotation = true
				isEnvVarAlreadyExists := isEnvVarPresent(containerRuntimeDetails.EnvFromContainerRuntime, envKey)
				if isEnvVarAlreadyExists {
					continue
				}

				containerRuntimeDetails.EnvFromContainerRuntime = append(containerRuntimeDetails.EnvFromContainerRuntime,
					v1alpha1.EnvVar{Name: envKey, Value: cleanedEnvValue})
				shouldChangeUpdateStateToSucceeded = true

			}
		}
		if shouldChangeUpdateStateToSucceeded { // ProcessingStateSucceeded is when values originally came from the Dockerfile
			state := v1alpha1.ProcessingStateSucceeded
			containerRuntimeDetails.RuntimeUpdateState = &state
		}

		// Determine if cleanup of Odigos values is needed in EnvFromContainerRuntime.
		// This addresses a bug in the runtime inspection for environment variables originating from the device.
		// If the "Generation runtime detection" failed EnvFromContainerRuntime will include odigos additions this code we clean it.
		if containerRuntimeDetails.RuntimeUpdateState != nil {

			if *containerRuntimeDetails.RuntimeUpdateState == v1alpha1.ProcessingStateSucceeded {
				filteredEnvVars := []v1alpha1.EnvVar{}

				for _, envVar := range containerRuntimeDetails.EnvFromContainerRuntime {
					if strings.Contains(envVar.Value, "/var/odigos") {
						// Skip the entry
						continue
					}
					// Keep the entry
					filteredEnvVars = append(filteredEnvVars, envVar)
				}
				containerRuntimeDetails.EnvFromContainerRuntime = filteredEnvVars
			}
			return nil, needToUpdateWorkloadAnnotation
		}

		// Mark as succeeded if no annotation set.
		// This occurs when the values were not originally present in the manifest, and the envOverwriter was skipped.
		if len(annotationEnvVarsForContainer) == 0 {
			state := v1alpha1.ProcessingStateSucceeded
			containerRuntimeDetails.RuntimeUpdateState = &state
		}

		for envKey, envValue := range annotationEnvVarsForContainer {

			// The containerRuntimeDetails might already include the EnvFromContainerRuntime if the runtime inspection was executed before the migration modified the environment variables.
			// In this case, we want to avoid overwriting the value set by Odiglet.
			// This check is only for safety, as we have already skipped processed containers.
			isEnvVarAlreadyExists := isEnvVarPresent(containerRuntimeDetails.EnvFromContainerRuntime, envKey)
			if isEnvVarAlreadyExists {
				continue
			}

			// if envValue is nil, it means that the value in the manifest is come from the runtime by the envOverwriter.
			// In his case, we mark as succeeded and set the EnvFromContainerRuntime as clean version of the manifest env values (without odigos additions).
			if envValue == nil {
				containerEnvFromManifestValue := k8scontainer.GetContainerEnvVarValue(&containerObject, envKey)
				if containerEnvFromManifestValue != nil {
					workloadEnvVarWithoutOdigosAdditions := envOverwrite.CleanupEnvValueFromOdigosAdditions(envKey, *containerEnvFromManifestValue)

					if workloadEnvVarWithoutOdigosAdditions != "" {
						envVarWithoutOdigosAddition := v1alpha1.EnvVar{Name: envKey, Value: workloadEnvVarWithoutOdigosAdditions}
						containerRuntimeDetails.EnvFromContainerRuntime = append(containerRuntimeDetails.EnvFromContainerRuntime, envVarWithoutOdigosAddition)
						state := v1alpha1.ProcessingStateSucceeded
						containerRuntimeDetails.RuntimeUpdateState = &state
					}

				}
			} else {
				// If envKey exists and != nil, it indicates that the environment variable originally came from the manifest.
				// In this case, we will set the RuntimeUpdateState to ProcessingStateSkipped.
				state := v1alpha1.ProcessingStateSkipped
				containerRuntimeDetails.RuntimeUpdateState = &state
			}
		}
	}
	return nil, needToUpdateWorkloadAnnotation
}

func contains(workloadNames map[string]*v1alpha1.InstrumentationConfig, workloadName string) bool {
	_, exists := workloadNames[workloadName]
	return exists
}

func isEnvVarPresent(envVars []v1alpha1.EnvVar, envVarName string) bool {
	for _, envVar := range envVars {
		if envVar.Name == envVarName {
			return true
		}
	}
	return false
}

func deleteOriginalEnvAnnotationInPlace(obj client.Object) {
	annotations := obj.GetAnnotations()
	if annotations == nil {
		return
	}
	delete(annotations, consts.ManifestEnvOriginalValAnnotation)
	obj.SetAnnotations(annotations)
}

func getPodManifestContainerByName(podSpec *corev1.PodSpec, containerName string) *corev1.Container {
	for i := range podSpec.Containers {
		if podSpec.Containers[i].Name == containerName {
			return &podSpec.Containers[i]
		}
	}
	return nil
}

func revertOriginalEnvAnnotationInPlace(originalWorkloadEnvVar *envoverwrite.OrigWorkloadEnvValues, podSpec *corev1.PodSpec) {
	for containerName, envVars := range originalWorkloadEnvVar.OrigManifestValues {

		// find the container to update in the pod manifest
		containerManifest := getPodManifestContainerByName(podSpec, containerName)
		if containerManifest == nil {
			// container found in original value annotation but not in spec.
			// should not happen, ignore it silently.
			continue
		}

		// iterate on all the existing envs in the manifest and either remove or update them
		newContainerEnvs := []corev1.EnvVar{}
		for _, manifestEnvVar := range containerManifest.Env {

			if !strings.Contains(manifestEnvVar.Value, "/var/odigos") {
				// we only revert values that odigos overwrote,
				// if the value is not odigos value, keep it as is
				newContainerEnvs = append(newContainerEnvs, manifestEnvVar)
				continue
			}

			originalEnvValue, found := envVars[manifestEnvVar.Name]
			if !found {
				// env var not found in the original value annotation, keep it as is
				newContainerEnvs = append(newContainerEnvs, manifestEnvVar)
				continue
			}

			if originalEnvValue == nil {
				// original value is nil, drop it from the updated manifest list
				continue
			} else {
				// just in case, make sure we introduce a clean value back to the manifest
				sanitizedEnvValue := envOverwrite.CleanupEnvValueFromOdigosAdditions(manifestEnvVar.Name, *originalEnvValue)
				newContainerEnvs = append(newContainerEnvs, corev1.EnvVar{Name: manifestEnvVar.Name, Value: sanitizedEnvValue})
			}
		}
		containerManifest.Env = newContainerEnvs
	}
}

func (m *MigrationRunnable) handleSingleWorkload(ctx context.Context, workloadObject client.Object) error {
	// Fetch fresh copy of the workload
	freshWorkload := workloadObject.DeepCopyObject().(client.Object)
	err := m.KubeClient.Get(ctx, client.ObjectKey{
		Namespace: workloadObject.GetNamespace(),
		Name:      workloadObject.GetName(),
	}, freshWorkload)
	if err != nil {
		m.Logger.Error(err, "Failed to fetch fresh copy of workload", "Name", workloadObject.GetName(), "Namespace", workloadObject.GetNamespace())
		return client.IgnoreNotFound(err)
	}

	// Fetch InstrumentationConfig
	workloadKindType := getWorkloadKindByWorkloadType(freshWorkload)
	if workloadKindType == nil {
		return fmt.Errorf("failed to get workload type for workload %s", workloadObject.GetName())
	}

	instConfigName := workload.CalculateWorkloadRuntimeObjectName(workloadObject.GetName(), *workloadKindType)

	// Fetching the latest state of the InstrumentationConfig resource from the Kubernetes API.
	// This is necessary to ensure we work with the most up-to-date version of the resource, as it may
	// have been modified by other processes or controllers in the cluster. Without this step, there is
	// a risk of encountering conflicts or using stale data during operations on the InstrumentationConfig object.
	freshInstConfig := v1alpha1.InstrumentationConfig{}
	err = m.KubeClient.Get(ctx, client.ObjectKey{
		Namespace: workloadObject.GetNamespace(),
		Name:      instConfigName,
	}, &freshInstConfig)
	if err != nil {
		m.Logger.Error(err, "Failed to get InstrumentationConfig", "Name", instConfigName, "Namespace", workloadObject.GetNamespace())
		return err
	}

	// Extract original environment variables
	originalWorkloadEnvVar, err := envoverwrite.NewOrigWorkloadEnvValues(freshWorkload.GetAnnotations())
	if err != nil {
		m.Logger.Error(err, "Failed to get original workload environment variables")
		return err
	}

	// Process container runtime details
	runtimeDetailsByContainer := freshInstConfig.Status.RuntimeDetailsByContainer
	podTemplateSpec := getPodTemplateSpecByWorkloadType(freshWorkload)

	for _, containerObject := range podTemplateSpec.Spec.Containers {
		err, _ = handleContainerRuntimeDetailsUpdate(
			containerObject,
			originalWorkloadEnvVar,
			runtimeDetailsByContainer,
		)
		if err != nil {
			return fmt.Errorf("failed to process container %s in workload %s: %v", containerObject.Name, workloadObject.GetName(), err)
		}
	}

	// Handle annotation, label, and device revert operations
	envReverted := false
	if odigosOriginalAnnotationFound(freshWorkload.GetAnnotations()) {
		deleteOriginalEnvAnnotationInPlace(freshWorkload)
		revertOriginalEnvAnnotationInPlace(originalWorkloadEnvVar, &podTemplateSpec.Spec)
		envReverted = true
	}
	labelRemoved := removeInjectInstrumentationLabel(podTemplateSpec)
	devicesRemoved := revertInstrumentationDevices(podTemplateSpec)

	// Update workload if changes were made
	if envReverted || labelRemoved || devicesRemoved {
		err = m.KubeClient.Update(ctx, freshWorkload)
		if err != nil {
			m.Logger.Error(err, "Failed to revert workload", "Name", freshWorkload.GetName(), "Namespace", freshWorkload.GetNamespace())
		}
	}

	// Update InstrumentationConfig status
	err = m.KubeClient.Status().Update(ctx, &freshInstConfig)
	if err != nil {
		m.Logger.Error(err, "Failed to update InstrumentationConfig status", "Name", freshWorkload.GetName(), "Namespace", freshWorkload.GetNamespace())
		return err
	}

	return nil
}

func getPodTemplateSpecByWorkloadType(workload client.Object) *corev1.PodTemplateSpec {
	switch w := workload.(type) {
	case *appsv1.Deployment:
		return &w.Spec.Template
	case *appsv1.StatefulSet:
		return &w.Spec.Template
	case *appsv1.DaemonSet:
		return &w.Spec.Template
	default:
		return nil
	}
}

func getWorkloadKindByWorkloadType(workload client.Object) *k8sconsts.WorkloadKind {
	switch workload.(type) {
	case *appsv1.Deployment:
		value := k8sconsts.WorkloadKindDeployment
		return &value
	case *appsv1.StatefulSet:
		value := k8sconsts.WorkloadKindStatefulSet
		return &value
	case *appsv1.DaemonSet:
		value := k8sconsts.WorkloadKindDaemonSet
		return &value
	}
	return nil
}
