package runtime_details_migration

import (
	"context"
	"fmt"
	"strings"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/common/envOverwrite"
	k8scontainer "github.com/odigos-io/odigos/k8sutils/pkg/container"
	"github.com/odigos-io/odigos/k8sutils/pkg/envoverwrite"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type MigrateRuntimeDetails struct {
	Client *kube.Client
}

func (m *MigrateRuntimeDetails) Name() string {
	return "migrate-runtime-details"
}

func (m *MigrateRuntimeDetails) Description() string {
	return "Migrate old RuntimeDetailsByContainer structure to the new format"
}

func (m *MigrateRuntimeDetails) TriggerVersion() string {
	return "v1.0.139"
}

func (m *MigrateRuntimeDetails) Execute() error {

	instrumentationConfigs, err := m.Client.OdigosClient.InstrumentationConfigs("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}

	workloadNamespaces := make(map[string]map[string]map[string]*v1alpha1.InstrumentationConfig)
	// Example structure:
	// {
	//   "deployment": {
	//       "default": {
	//           "frontend": *<InstrumentationConfig>,
	//       },
	//   },

	for _, item := range instrumentationConfigs.Items {
		IcName := item.GetName()
		IcNamespace := item.GetNamespace()
		parts := strings.Split(IcName, "-")
		if len(parts) < 2 {
			continue
		}

		workloadType := parts[0] // deployment/statefulset/daemonset
		workloadName := strings.Join(parts[1:], "-")
		if _, exists := workloadNamespaces[workloadType]; !exists {
			workloadNamespaces[workloadType] = make(map[string]map[string]*v1alpha1.InstrumentationConfig)
		}
		if _, exists := workloadNamespaces[workloadType][IcNamespace]; !exists {
			workloadNamespaces[workloadType][IcNamespace] = make(map[string]*v1alpha1.InstrumentationConfig)
		}

		// Save workloadName and the corresponding InstrumentationConfig reference
		workloadNamespaces[workloadType][IcNamespace][workloadName] = &item
	}
	for workloadType, namespaces := range workloadNamespaces {
		switch workloadType {
		case "deployment":
			if err := fetchAndProcessDeployments(m.Client, namespaces); err != nil {
				return err
			}
		case "statefulset":
			if err := fetchAndProcessStatefulSets(m.Client, namespaces); err != nil {
				return err
			}
		case "daemonset":
			if err := fetchAndProcessDaemonSets(m.Client, namespaces); err != nil {
				return err
			}
		default:
			fmt.Printf("Unknown workload type: %s\n", workloadType)
		}
	}

	return nil
}

func fetchAndProcessDeployments(clientset *kube.Client, namespaces map[string]map[string]*v1alpha1.InstrumentationConfig) error {
	for namespace, workloadNames := range namespaces {
		deployments, err := clientset.AppsV1().Deployments(namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return fmt.Errorf("failed to list deployments in namespace %s: %v", namespace, err)
		}

		for _, dep := range deployments.Items {

			// Checking if the deployment is in the list of deployments that need to be processed
			if contains(workloadNames, dep.Name) {

				originalWorkloadEnvVar, _ := envoverwrite.NewOrigWorkloadEnvValues(dep.Annotations)
				workloadInstrumentationConfigReference := workloadNames[dep.Name]
				runtimeDetailsByContainer := workloadInstrumentationConfigReference.Status.RuntimeDetailsByContainer

				for _, containerObject := range dep.Spec.Template.Spec.Containers {

					err := handleContainerRuntimeDetailsUpdate(
						containerObject,
						*originalWorkloadEnvVar,
						&runtimeDetailsByContainer,
					)
					if err != nil {
						return fmt.Errorf("failed to process container %s in deployment %s: %v", containerObject.Name, dep.Name, err)
					}
				}
				_, err = clientset.OdigosClient.InstrumentationConfigs(dep.Namespace).UpdateStatus(
					context.TODO(),
					workloadInstrumentationConfigReference,
					metav1.UpdateOptions{})
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func fetchAndProcessStatefulSets(clientset *kube.Client, namespaces map[string]map[string]*v1alpha1.InstrumentationConfig) error {
	for namespace, workloadNames := range namespaces {
		statefulSets, err := clientset.AppsV1().StatefulSets(namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return fmt.Errorf("failed to list statefulsets in namespace %s: %v", namespace, err)
		}

		for _, sts := range statefulSets.Items {
			// Checking if the statefulset is in the list of statefulsets that need to be processed
			if contains(workloadNames, sts.Name) {

				originalWorkloadEnvVar, _ := envoverwrite.NewOrigWorkloadEnvValues(sts.Annotations)
				workloadInstrumentationConfigReference := workloadNames[sts.Name]
				runtimeDetailsByContainer := workloadInstrumentationConfigReference.Status.RuntimeDetailsByContainer

				for _, containerObject := range sts.Spec.Template.Spec.Containers {
					err := handleContainerRuntimeDetailsUpdate(
						containerObject,
						*originalWorkloadEnvVar,
						&runtimeDetailsByContainer,
					)
					if err != nil {
						return fmt.Errorf("failed to process container %s in statefulset %s: %v", containerObject.Name, sts.Name, err)
					}
				}

				// Update runtimeDetailsByContainer in workloadInstrumentationConfigReference
				workloadInstrumentationConfigReference.Status.RuntimeDetailsByContainer = runtimeDetailsByContainer

				// Update the InstrumentationConfig status
				_, err = clientset.OdigosClient.InstrumentationConfigs(sts.Namespace).UpdateStatus(
					context.TODO(),
					workloadInstrumentationConfigReference,
					metav1.UpdateOptions{},
				)
				if err != nil {
					return fmt.Errorf("failed to update status for statefulset %s in namespace %s: %v", sts.Name, sts.Namespace, err)
				}
			}
		}
	}
	return nil
}

func fetchAndProcessDaemonSets(clientset *kube.Client, namespaces map[string]map[string]*v1alpha1.InstrumentationConfig) error {
	for namespace, workloadNames := range namespaces {
		daemonSets, err := clientset.AppsV1().DaemonSets(namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return fmt.Errorf("failed to list daemonsets in namespace %s: %v", namespace, err)
		}

		for _, ds := range daemonSets.Items {
			// Checking if the daemonset is in the list of daemonsets that need to be processed
			if contains(workloadNames, ds.Name) {

				originalWorkloadEnvVar, _ := envoverwrite.NewOrigWorkloadEnvValues(ds.Annotations)
				workloadInstrumentationConfigReference := workloadNames[ds.Name]
				runtimeDetailsByContainer := workloadInstrumentationConfigReference.Status.RuntimeDetailsByContainer

				for _, containerObject := range ds.Spec.Template.Spec.Containers {
					err := handleContainerRuntimeDetailsUpdate(
						containerObject,
						*originalWorkloadEnvVar,
						&runtimeDetailsByContainer)
					if err != nil {
						return fmt.Errorf("failed to process container %s in daemonset %s: %v", containerObject.Name, ds.Name, err)
					}
				}

				// Update runtimeDetailsByContainer in workloadInstrumentationConfigReference
				workloadInstrumentationConfigReference.Status.RuntimeDetailsByContainer = runtimeDetailsByContainer

				// Update the InstrumentationConfig status
				_, err = clientset.OdigosClient.InstrumentationConfigs(ds.Namespace).UpdateStatus(
					context.TODO(),
					workloadInstrumentationConfigReference,
					metav1.UpdateOptions{},
				)
				if err != nil {
					return fmt.Errorf("failed to update status for daemonset %s in namespace %s: %v", ds.Name, ds.Namespace, err)
				}
			}
		}
	}
	return nil
}
func handleContainerRuntimeDetailsUpdate(
	containerObject v1.Container,
	originalWorkloadEnvVar envoverwrite.OrigWorkloadEnvValues,
	runtimeDetailsByContainer *[]v1alpha1.RuntimeDetailsByContainer,
) error {
	for i := range *runtimeDetailsByContainer {
		containerRuntimeDetails := &(*runtimeDetailsByContainer)[i]

		// Find the relevant container in runtimeDetailsByContainer
		if containerRuntimeDetails.ContainerName != containerObject.Name {
			continue
		}

		// Process environment variables for the container
		annotationEnvVarsForContainer := originalWorkloadEnvVar.GetContainerStoredEnvs(containerObject.Name)
		for envKey, envValue := range annotationEnvVarsForContainer {
			isEnvVarAlreadyExists := isEnvVarPresent(containerRuntimeDetails.EnvVarsFromDockerFile, envKey)
			if isEnvVarAlreadyExists {
				continue
			}

			// Handle runtime-originated environment variables
			if envValue == nil {
				containerEnvFromManifestValue := k8scontainer.GetContainerEnvVarValue(&containerObject, envKey)
				if containerEnvFromManifestValue != nil {
					workloadEnvVarWithoutOdigosAdditions := cleanUpManifestValueFromOdigosAdditions(envKey, *containerEnvFromManifestValue)
					envVarWithoutOdigosAddition := v1alpha1.EnvVar{Name: envKey, Value: workloadEnvVarWithoutOdigosAdditions}
					containerRuntimeDetails.EnvVarsFromDockerFile = append(containerRuntimeDetails.EnvVarsFromDockerFile, envVarWithoutOdigosAddition)
					state := v1alpha1.ProcessingStateSucceeded
					containerRuntimeDetails.RuntimeUpdateState = &state
				}
			}
		}

		// Mark container as skipped if no runtime environment variables exist
		if len(containerRuntimeDetails.EnvVarsFromDockerFile) == 0 {
			state := v1alpha1.ProcessingStateSkipped
			containerRuntimeDetails.RuntimeUpdateState = &state
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
		withoutTrailingSpace := cleanTrailingChar(withoutTrailingColon, " ")
		return withoutTrailingSpace
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
