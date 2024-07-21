package cmd

import (
	"fmt"
	"strings"

	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"github.com/spf13/cobra"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	describeNamespaceFlag string
)

func cmdKindToK8sGVR(kind string) (schema.GroupVersionResource, error) {
	kind = strings.ToLower(kind)
	if kind == "deployment" || kind == "deployments" || kind == "dep" {
		return schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}, nil
	}
	if kind == "statefulset" || kind == "statefulsets" || kind == "sts" {
		return schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "statefulsets"}, nil
	}
	if kind == "daemonset" || kind == "daemonsets" || kind == "ds" {
		return schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "daemonsets"}, nil
	}

	return schema.GroupVersionResource{}, fmt.Errorf("unsupported kind: %s", kind)
}

func extractPodTemplate(obj *unstructured.Unstructured) (*v1.PodTemplateSpec, error) {
	gvk := obj.GroupVersionKind()

	switch gvk.Kind {
	case "Deployment":
		var deployment appsv1.Deployment
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &deployment)
		if err != nil {
			return nil, fmt.Errorf("failed to cast to Deployment: %v", err)
		}
		return &deployment.Spec.Template, nil

	case "StatefulSet":
		var statefulSet appsv1.StatefulSet
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &statefulSet)
		if err != nil {
			return nil, fmt.Errorf("failed to cast to StatefulSet: %v", err)
		}
		return &statefulSet.Spec.Template, nil

	case "DaemonSet":
		var daemonSet appsv1.DaemonSet
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &daemonSet)
		if err != nil {
			return nil, fmt.Errorf("failed to cast to DaemonSet: %v", err)
		}
		return &daemonSet.Spec.Template, nil

	default:
		return nil, fmt.Errorf("unsupported kind: %s", gvk.Kind)
	}
}

func getInstrumentationLabelTexts(workload *unstructured.Unstructured, ns *v1.Namespace) (workloadText, nsText, decisionText string, instrumented bool) {
	odigosLabel, workloadFound := workload.GetLabels()[consts.OdigosInstrumentationLabel]
	nsLabel, nsFound := ns.GetLabels()[consts.OdigosInstrumentationLabel]

	if workloadFound {
		workloadText = consts.OdigosInstrumentationLabel + "=" + odigosLabel
	} else {
		workloadText = consts.OdigosInstrumentationLabel + " label not set"
	}

	if nsFound {
		nsText = consts.OdigosInstrumentationLabel + "=" + nsLabel
	} else {
		nsText = consts.OdigosInstrumentationLabel + " label not set"
	}

	if workloadFound {
		instrumented = odigosLabel == consts.InstrumentationEnabled
		if instrumented {
			decisionText = "Workload is instrumented because the " + workload.GetKind() + " contains the '" + consts.OdigosInstrumentationLabel + "' label with value '" + consts.InstrumentationEnabled + "'"
		} else {
			decisionText = "Workload is not instrumented because the " + workload.GetKind() + " contains the '" + consts.OdigosInstrumentationLabel + "' label with value '" + odigosLabel + "'"
		}
	} else {
		instrumented = nsText == consts.InstrumentationEnabled
		if instrumented {
			decisionText = workload.GetKind() + " is instrumented because the it's namespace " + consts.OdigosInstrumentationLabel + " is set to " + consts.InstrumentationEnabled
		} else {
			if nsFound {
				decisionText = workload.GetKind() + " is not instrumented because the it's namespace " + consts.OdigosInstrumentationLabel + " is set to " + nsLabel
			} else {
				decisionText = workload.GetKind() + " is not instrumented because neither the workload nor the namespace has the " + consts.OdigosInstrumentationLabel + " label set"
			}
		}
	}

	return
}

// installCmd represents the install command
var describeCmd = &cobra.Command{
	Use:   "describe",
	Short: "Show details of a specific odigos entity",
	Long:  `Print detailed description of a specific odigos entity, which can be used to troubleshoot issues`,
	Run: func(cmd *cobra.Command, args []string) {

		client, err := kube.CreateClient(cmd)
		if err != nil {
			kube.PrintClientErrorAndExit(err)
		}
		ctx := cmd.Context()
		ns := cmd.Flag("namespace").Value.String()

		if len(args) != 2 {
			fmt.Println("Usage: odigos describe <kind> <name>")
			return
		}

		kind := args[0]
		name := args[1]

		gvr, err := cmdKindToK8sGVR(kind)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		workloadObj, err := client.Dynamic.Resource(gvr).Namespace(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		namespace, err := client.CoreV1().Namespaces().Get(ctx, ns, metav1.GetOptions{})
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		fmt.Println("Showing details for Workload")
		fmt.Println("Name: ", workloadObj.GetName())
		fmt.Println("Kind: ", workloadObj.GetKind())
		fmt.Println("Namespace: ", workloadObj.GetNamespace())

		fmt.Println("")
		fmt.Println("Labels:")
		workloadText, nsText, decisionText, instrumented := getInstrumentationLabelTexts(workloadObj, namespace)
		fmt.Println("Workload: " + workloadText)
		fmt.Println("Namespace: " + nsText)
		fmt.Println("Instrumented: ", instrumented)
		fmt.Println("Decision: " + decisionText)

		runtimeObjectName := workload.GetRuntimeObjectName(workloadObj.GetName(), workloadObj.GetKind())
		instrumentationConfig, err := client.OdigosClient.InstrumentationConfigs(ns).Get(ctx, runtimeObjectName, metav1.GetOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			fmt.Printf("Error: %v\n", err)
			return
		}
		fmt.Println("")
		fmt.Println("Instrumentation Config:")
		if apierrors.IsNotFound(err) {
			fmt.Println("Not yet created")
		} else {
			fmt.Println("Created at " + instrumentationConfig.GetCreationTimestamp().String())
		}

		instrumentedApplication, err := client.OdigosClient.InstrumentedApplications(ns).Get(ctx, runtimeObjectName, metav1.GetOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			fmt.Printf("Error: %v\n", err)
			return
		}
		fmt.Println("")
		fmt.Println("Runtime inspection details:")
		if apierrors.IsNotFound(err) {
			fmt.Println("Not yet created")
		} else {
			fmt.Println("Created at " + instrumentedApplication.GetCreationTimestamp().String())
			fmt.Println("Detected Containers:")
			for _, container := range instrumentedApplication.Spec.RuntimeDetails {
				fmt.Println("    - Container Name:", container.ContainerName)
				fmt.Println("      Language:      ", container.Language)
			}
		}

		podTemplate, err := extractPodTemplate(workloadObj)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		fmt.Println("")
		fmt.Println("Instrumentation Device:")
		for _, container := range podTemplate.Spec.Containers {
			fmt.Println("    - Container Name:", container.Name)
			odigosDevices := make([]string, 0)
			for resourceName := range container.Resources.Limits {
				deviceName, found := strings.CutPrefix(resourceName.String(), common.OdigosResourceNamespace+"/")
				if found {
					odigosDevices = append(odigosDevices, deviceName)
				}
			}
			if len(odigosDevices) == 0 {
				fmt.Println("      No instrumentation devices")
			} else {
				fmt.Println("      Instrumentation Devices:", strings.Join(odigosDevices, ", "))
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(describeCmd)
	describeCmd.Flags().StringVarP(&describeNamespaceFlag, "namespace", "n", "default", "namespace of the resource being described")

}
