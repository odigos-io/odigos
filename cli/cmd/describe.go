package cmd

import (
	"fmt"

	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/describe"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	describeNamespaceFlag string
)

var describeCmd = &cobra.Command{
	Use:   "describe",
	Short: "Show details of a specific odigos entity",
	Long:  `Print detailed description of a specific odigos entity, which can be used to troubleshoot issues`,
}

var describeSourceCmd = &cobra.Command{
	Use:   "source",
	Short: "Show details of a specific odigos source",
	Long:  `Print detailed description of a specific odigos source, which can be used to troubleshoot issues`,
}

var describeSourceDeploymentCmd = &cobra.Command{
	Use:     "deployment <name>",
	Short:   "Show details of a specific odigos source of type deployment",
	Long:    `Print detailed description of a specific odigos source of type deployment, which can be used to troubleshoot issues`,
	Aliases: []string{"deploy", "deployments", "deploy.apps", "deployment.apps", "deployments.apps"},
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		client, err := kube.CreateClient(cmd)
		if err != nil {
			kube.PrintClientErrorAndExit(err)
		}

		ctx := cmd.Context()
		name := args[0]
		ns := cmd.Flag("namespace").Value.String()
		deployment, err := client.AppsV1().Deployments(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		workloadObj := &describe.K8sSourceObject{
			Kind:            "deployment",
			ObjectMeta:      deployment.ObjectMeta,
			PodTemplateSpec: &deployment.Spec.Template,
			LabelSelector:   deployment.Spec.Selector,
		}
		describeText := describe.PrintDescribeSource(ctx, client.Interface, client.OdigosClient, workloadObj)
		fmt.Println(describeText)
	},
}

var describeSourceDaemonSetCmd = &cobra.Command{
	Use:     "daemonset <name>",
	Short:   "Show details of a specific odigos source of type daemonset",
	Long:    `Print detailed description of a specific odigos source of type daemonset, which can be used to troubleshoot issues`,
	Aliases: []string{"ds", "daemonsets", "ds.apps", "daemonset.apps", "daemonsets.apps"},
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client, err := kube.CreateClient(cmd)
		if err != nil {
			kube.PrintClientErrorAndExit(err)
		}

		ctx := cmd.Context()
		name := args[0]
		ns := cmd.Flag("namespace").Value.String()
		ds, err := client.AppsV1().DaemonSets(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		workloadObj := &describe.K8sSourceObject{
			Kind:            "daemonset",
			ObjectMeta:      ds.ObjectMeta,
			PodTemplateSpec: &ds.Spec.Template,
			LabelSelector:   ds.Spec.Selector,
		}
		describeText := describe.PrintDescribeSource(ctx, client, client.OdigosClient, workloadObj)
		fmt.Println(describeText)
	},
}

var describeSourceStatefulSetCmd = &cobra.Command{
	Use:     "statefulset <name>",
	Short:   "Show details of a specific odigos source of type statefulset",
	Long:    `Print detailed description of a specific odigos source of type statefulset, which can be used to troubleshoot issues`,
	Aliases: []string{"sts", "statefulsets", "sts.apps", "statefulset.apps", "statefulsets.apps"},
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client, err := kube.CreateClient(cmd)
		if err != nil {
			kube.PrintClientErrorAndExit(err)
		}

		ctx := cmd.Context()
		name := args[0]
		ns := cmd.Flag("namespace").Value.String()
		sts, err := client.AppsV1().StatefulSets(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		workloadObj := &describe.K8sSourceObject{
			Kind:            "statefulset",
			ObjectMeta:      sts.ObjectMeta,
			PodTemplateSpec: &sts.Spec.Template,
			LabelSelector:   sts.Spec.Selector,
		}
		describeText := describe.PrintDescribeSource(ctx, client.Interface, client.OdigosClient, workloadObj)
		fmt.Println(describeText)
	},
}

func init() {

	// describe
	rootCmd.AddCommand(describeCmd)

	// source
	describeCmd.AddCommand(describeSourceCmd)
	describeSourceCmd.PersistentFlags().StringVarP(&describeNamespaceFlag, "namespace", "n", "default", "namespace of the source being described")

	// source kinds
	describeSourceCmd.AddCommand(describeSourceDeploymentCmd)
	describeSourceCmd.AddCommand(describeSourceDaemonSetCmd)
	describeSourceCmd.AddCommand(describeSourceStatefulSetCmd)
}
