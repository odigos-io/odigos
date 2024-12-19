package runtime_details_migration

import (
	"context"
	"fmt"
	"strings"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/envoverwrite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
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
	fmt.Println("Migrating RuntimeDetailsByContainer....")

	gvr := schema.GroupVersionResource{Group: "odigos.io", Version: "v1alpha1", Resource: "instrumentationconfigs"}

	ctx := context.TODO()
	instrumentationConfigs, err := m.Client.Dynamic.Resource(gvr).List(ctx, metav1.ListOptions{})
	if err != nil {
		panic(fmt.Errorf("failed to list InstrumentationConfigs: %v", err))
	}

	workloadNamespaces := make(map[string]map[string][]string) // workloadType -> namespace -> []workloadNames
	for _, item := range instrumentationConfigs.Items {
		fmt.Printf("Found InstrumentationConfig: %s in namespace: %s\n", item.GetName(), item.GetNamespace())
		IcName := item.GetName()
		IcNamespace := item.GetNamespace()
		parts := strings.Split(IcName, "-")
		if len(parts) < 2 {
			fmt.Printf("Skipping invalid InstrumentationConfig name: %s\n", IcName)
			continue
		}

		workloadType := parts[0] // deployment/statefulset/aemonset
		workloadName := strings.Join(parts[1:], "-")
		if _, exists := workloadNamespaces[workloadType]; !exists {
			workloadNamespaces[workloadType] = make(map[string][]string)
		}
		workloadNamespaces[workloadType][IcNamespace] = append(workloadNamespaces[workloadType][IcNamespace], workloadName)
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

func fetchAndProcessDeployments(clientset *kube.Client, namespaces map[string][]string) error {
	for namespace, workloadNames := range namespaces {
		fmt.Printf("Processing Deployments in namespace: %s\n", namespace)
		deployments, err := clientset.AppsV1().Deployments(namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return fmt.Errorf("failed to list deployments in namespace %s: %v", namespace, err)
		}

		for _, dep := range deployments.Items {
			if contains(workloadNames, dep.Name) {
				fmt.Printf("Processing Deployment: %s in namespace: %s\n", dep.Name, dep.Namespace)
				originalEnvVar, _ := envoverwrite.NewOrigWorkloadEnvValues(dep.Annotations)
				allNil := originalEnvVar.AreAllEnvValuesNil()
				if allNil {
					// update instrumentationConfig object
				}
				instrumentationConfig, err := clientset.OdigosClient.InstrumentationConfigs(dep.Namespace).Get(context.TODO(), dep.Name, metav1.GetOptions{})
				if err != nil {
					fmt.Printf("Failed to get InstrumentationConfig: %v\n", err)
				}
				// TODO: Need to move the general operations to top level object and not per iteration.
				// updating runtimeDetailsByContainer as Skipped - executed but nothing happen
				for _, runtimeDetails := range instrumentationConfig.Status.RuntimeDetailsByContainer {
					value := v1alpha1.ProcessingStateSkipped
					runtimeDetails.RuntimeUpdateState = &value
				}
				for _, container := range dep.Spec.Template.Spec.Containers {
					fmt.Printf("Processing container: %s\n", container.Name)
					// Add your deployment-specific logic here
				}
			}
		}
	}
	return nil
}

func fetchAndProcessStatefulSets(clientset *kube.Client, namespaces map[string][]string) error {
	for namespace, workloadNames := range namespaces {
		statefulSets, err := clientset.AppsV1().StatefulSets(namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return fmt.Errorf("failed to list statefulsets in namespace %s: %v", namespace, err)
		}

		for _, sts := range statefulSets.Items {
			if contains(workloadNames, sts.Name) {
				fmt.Printf("Processing StatefulSet: %s in namespace: %s\n", sts.Name, sts.Namespace)
				// Add your statefulset-specific logic here
			}
		}
	}
	return nil
}

func fetchAndProcessDaemonSets(clientset *kube.Client, namespaces map[string][]string) error {
	for namespace, workloadNames := range namespaces {
		daemonSets, err := clientset.AppsV1().DaemonSets(namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return fmt.Errorf("failed to list daemonsets in namespace %s: %v", namespace, err)
		}

		for _, ds := range daemonSets.Items {
			if contains(workloadNames, ds.Name) {
				fmt.Printf("Processing DaemonSet: %s in namespace: %s\n", ds.Name, ds.Namespace)
				// Add your daemonset-specific logic here
			}
		}
	}
	return nil
}

func contains(slice []string, item string) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}
