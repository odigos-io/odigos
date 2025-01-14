package runtimemigration

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/envOverwrite"
	k8scontainer "github.com/odigos-io/odigos/k8sutils/pkg/container"
	"github.com/odigos-io/odigos/k8sutils/pkg/envoverwrite"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type MigrationRunnable struct {
	KubeClient client.Client
	Logger     logr.Logger
}

// This code ensures that migrationRunnable is categorized as an `Other` Runnable.
func (m *MigrationRunnable) NeedLeaderElection() bool {
	return false
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

func (m *MigrationRunnable) fetchAndProcessDeployments(ctx context.Context, kubeClient client.Client, namespaces map[string]map[string]*v1alpha1.InstrumentationConfig) error {
	for namespace, workloadNames := range namespaces {
		var deployments appsv1.DeploymentList
		err := kubeClient.List(ctx, &deployments, &client.ListOptions{Namespace: namespace})
		if err != nil {
			return fmt.Errorf("failed to list deployments in namespace %s: %v", namespace, err)
		}

		for _, dep := range deployments.Items {

			// Checking if the deployment is in the list of deployments that need to be processed
			if contains(workloadNames, dep.Name) {

				originalWorkloadEnvVar, err := envoverwrite.NewOrigWorkloadEnvValues(dep.Annotations)
				if err != nil {
					m.Logger.Error(err, "Failed to get original workload environment variables")
					continue
				}

				workloadInstrumentationConfigReference := workloadNames[dep.Name]
				if workloadInstrumentationConfigReference == nil {
					m.Logger.Error(err, "Failed to get InstrumentationConfig reference")
					continue
				}

				// Fetching the latest state of the InstrumentationConfig resource from the Kubernetes API.
				// This is necessary to ensure we work with the most up-to-date version of the resource, as it may
				// have been modified by other processes or controllers in the cluster. Without this step, there is
				// a risk of encountering conflicts or using stale data during operations on the InstrumentationConfig object.
				err = m.KubeClient.Get(ctx, client.ObjectKey{
					Namespace: workloadInstrumentationConfigReference.Namespace,
					Name:      workloadInstrumentationConfigReference.Name,
				}, workloadInstrumentationConfigReference)

				if err != nil {
					m.Logger.Error(err, "Failed to get InstrumentationConfig", "Name", workloadInstrumentationConfigReference.Name,
						"Namespace", workloadInstrumentationConfigReference.Namespace)
					continue
				}

				runtimeDetailsByContainer := workloadInstrumentationConfigReference.Status.RuntimeDetailsByContainer

				for _, containerObject := range dep.Spec.Template.Spec.Containers {

					err := handleContainerRuntimeDetailsUpdate(
						containerObject,
						*originalWorkloadEnvVar,
						runtimeDetailsByContainer,
					)
					if err != nil {
						return fmt.Errorf("failed to process container %s in deployment %s: %v", containerObject.Name, dep.Name, err)
					}
				}
				err = kubeClient.Status().Update(
					ctx,
					workloadInstrumentationConfigReference,
				)
				if err != nil {
					m.Logger.Error(err, "Failed to update InstrumentationConfig status", "Name", dep.Name, "Namespace", dep.Namespace)
					continue
				}
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
			if contains(workloadNames, sts.Name) {

				originalWorkloadEnvVar, err := envoverwrite.NewOrigWorkloadEnvValues(sts.Annotations)
				if err != nil {
					m.Logger.Error(err, "Failed to get original workload environment variables")
					continue
				}

				workloadInstrumentationConfigReference := workloadNames[sts.Name]
				if workloadInstrumentationConfigReference == nil {
					m.Logger.Error(err, "Failed to get InstrumentationConfig reference")
					continue
				}

				// Fetching the latest state of the InstrumentationConfig resource from the Kubernetes API.
				// This is necessary to ensure we work with the most up-to-date version of the resource, as it may
				// have been modified by other processes or controllers in the cluster. Without this step, there is
				// a risk of encountering conflicts or using stale data during operations on the InstrumentationConfig object.
				err = m.KubeClient.Get(ctx, client.ObjectKey{
					Namespace: workloadInstrumentationConfigReference.Namespace,
					Name:      workloadInstrumentationConfigReference.Name,
				}, workloadInstrumentationConfigReference)

				if err != nil {
					m.Logger.Error(err, "Failed to get InstrumentationConfig", "Name", workloadInstrumentationConfigReference.Name,
						"Namespace", workloadInstrumentationConfigReference.Namespace)
					continue
				}

				runtimeDetailsByContainer := workloadInstrumentationConfigReference.Status.RuntimeDetailsByContainer

				for _, containerObject := range sts.Spec.Template.Spec.Containers {
					err := handleContainerRuntimeDetailsUpdate(
						containerObject,
						*originalWorkloadEnvVar,
						runtimeDetailsByContainer,
					)
					if err != nil {
						return fmt.Errorf("failed to process container %s in statefulset %s: %v", containerObject.Name, sts.Name, err)
					}
				}

				// Update the InstrumentationConfig status
				err = kubeClient.Status().Update(
					ctx,
					workloadInstrumentationConfigReference,
				)
				if err != nil {
					m.Logger.Error(err, "Failed to update InstrumentationConfig status", "Name", sts.Name, "Namespace", sts.Namespace)
					continue
				}
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
			if contains(workloadNames, ds.Name) {

				originalWorkloadEnvVar, err := envoverwrite.NewOrigWorkloadEnvValues(ds.Annotations)
				if err != nil {
					m.Logger.Error(err, "Failed to get original workload environment variables")
					continue
				}
				workloadInstrumentationConfigReference := workloadNames[ds.Name]
				if workloadInstrumentationConfigReference == nil {
					m.Logger.Error(err, "Failed to get InstrumentationConfig reference")
					continue
				}

				// Fetching the latest state of the InstrumentationConfig resource from the Kubernetes API.
				// This is necessary to ensure we work with the most up-to-date version of the resource, as it may
				// have been modified by other processes or controllers in the cluster. Without this step, there is
				// a risk of encountering conflicts or using stale data during operations on the InstrumentationConfig object.
				err = m.KubeClient.Get(ctx, client.ObjectKey{
					Namespace: workloadInstrumentationConfigReference.Namespace,
					Name:      workloadInstrumentationConfigReference.Name,
				}, workloadInstrumentationConfigReference)

				if err != nil {
					m.Logger.Error(err, "Failed to get InstrumentationConfig", "Name", workloadInstrumentationConfigReference.Name,
						"Namespace", workloadInstrumentationConfigReference.Namespace)
					continue
				}
				runtimeDetailsByContainer := workloadInstrumentationConfigReference.Status.RuntimeDetailsByContainer

				for _, containerObject := range ds.Spec.Template.Spec.Containers {
					err := handleContainerRuntimeDetailsUpdate(
						containerObject,
						*originalWorkloadEnvVar,
						runtimeDetailsByContainer)
					if err != nil {
						return fmt.Errorf("failed to process container %s in daemonset %s: %v", containerObject.Name, ds.Name, err)
					}
				}

				// Update the InstrumentationConfig status
				err = kubeClient.Status().Update(
					ctx,
					workloadInstrumentationConfigReference,
				)
				if err != nil {
					m.Logger.Error(err, "Failed to update InstrumentationConfig status", "Name", ds.Name, "Namespace", ds.Namespace)
					continue
				}
			}
		}
	}
	return nil
}
func handleContainerRuntimeDetailsUpdate(
	containerObject v1.Container,
	originalWorkloadEnvVar envoverwrite.OrigWorkloadEnvValues,
	runtimeDetailsByContainer []v1alpha1.RuntimeDetailsByContainer,
) error {
	for i := range runtimeDetailsByContainer {
		containerRuntimeDetails := &(runtimeDetailsByContainer)[i]

		// Find the relevant container in runtimeDetailsByContainer
		if containerRuntimeDetails.ContainerName != containerObject.Name {
			continue
		}
		// Skip if the container has already been processed
		if containerRuntimeDetails.RuntimeUpdateState != nil {
			return nil
		}

		annotationEnvVarsForContainer := originalWorkloadEnvVar.GetContainerStoredEnvs(containerObject.Name)

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
					workloadEnvVarWithoutOdigosAdditions := cleanUpManifestValueFromOdigosAdditions(envKey, *containerEnvFromManifestValue)

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
	return nil
}

func contains(workloadNames map[string]*v1alpha1.InstrumentationConfig, workloadName string) bool {
	_, exists := workloadNames[workloadName]
	return exists
}

func cleanUpManifestValueFromOdigosAdditions(manifestEnvVarKey string, manifestEnvVarValue string) string {
	_, exists := envOverwrite.EnvValuesMap[manifestEnvVarKey]
	if exists {
		// clean up the value from all possible odigos additions
		for _, value := range envOverwrite.GetPossibleValuesPerEnv(manifestEnvVarKey) {
			manifestEnvVarValue = strings.ReplaceAll(manifestEnvVarValue, value, "")
		}
		withoutTrailingColon := cleanTrailingChar(manifestEnvVarValue, ":")
		withoutTrailingAndLeadingSpace := strings.TrimSpace(withoutTrailingColon)
		return withoutTrailingAndLeadingSpace
	} else {
		// manifestEnvVarKey does not exist in the EnvValuesMap
		return ""
	}
}

// In case we remove OdigosAdditions to PythonPath we need to remove this also.
func cleanTrailingChar(input string, char string) string {
	if len(input) > 0 && input[len(input)-1:] == char {
		return input[:len(input)-1]
	}
	return input
}

func isEnvVarPresent(envVars []v1alpha1.EnvVar, envVarName string) bool {
	for _, envVar := range envVars {
		if envVar.Name == envVarName {
			return true
		}
	}
	return false
}
