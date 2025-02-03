package cmd

import (
	"fmt"
	"os"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	cmdcontext "github.com/odigos-io/odigos/cli/pkg/cmd_context"
	"github.com/odigos-io/odigos/cli/pkg/confirm"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	sourceFlags *pflag.FlagSet

	namespaceFlagName = "namespace"

	allNamespacesFlagName = "all-namespaces"
	allNamespaceFlag      bool

	workloadKindFlagName = "workload-kind"
	workloadKindFlag     string

	workloadNameFlagName = "workload-name"
	workloadNameFlag     string

	workloadNamespaceFlagName = "workload-namespace"
	workloadNamespaceFlag     string

	disableInstrumentationFlagName = "disable-instrumentation"
	disableInstrumentationFlag     bool

	sourceGroupFlagName = "group"
	sourceGroupFlag     string

	sourceSetGroupFlagName = "set-group"
	sourceSetGroupFlag     string

	sourceRemoveGroupFlagName = "remove-group"
	sourceRemoveGroupFlag     string
)

var sourcesCmd = &cobra.Command{
	Use:   "sources [command] [flags]",
	Short: "Manage Odigos Sources in a cluster",
	Long:  "This command can be used to create, delete, or update Sources to configure workload or namespace auto-instrumentation",
	Example: `
# Create a Source "foo-source" for deployment "foo" in namespace "default"
odigos sources create foo-source --workload-kind=Deployment --workload-name=foo --workload-namespace=default -n default

# Update all existing Sources in namespace "default" to disable instrumentation
odigos sources update --disable-instrumentation -n default

# Delete all Sources in group "mygroup"
odigos sources delete --group mygroup --all-namespaces
	`,
}

var sourceCreateCmd = &cobra.Command{
	Use:   "create [name] [flags]",
	Short: "Create an Odigos Source",
	Long:  "This command will create the named Source object for the provided workload.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		client := cmdcontext.KubeClientFromContextOrExit(ctx)
		disableInstrumentation := disableInstrumentationFlag
		sourceName := args[0]

		source := &v1alpha1.Source{
			ObjectMeta: v1.ObjectMeta{
				Name:      sourceName,
				Namespace: namespaceFlag,
			},
			Spec: v1alpha1.SourceSpec{
				Workload: k8sconsts.PodWorkload{
					Kind:      k8sconsts.WorkloadKind(workloadKindFlag),
					Name:      workloadNameFlag,
					Namespace: workloadNamespaceFlag,
				},
				DisableInstrumentation: disableInstrumentation,
			},
		}

		if len(sourceGroupFlag) > 0 {
			source.Labels = make(map[string]string)
			source.Labels[k8sconsts.SourceGroupLabelPrefix+sourceGroupFlag] = "true"
		}

		_, err := client.OdigosClient.Sources(namespaceFlag).Create(ctx, source, v1.CreateOptions{})
		if err != nil {
			fmt.Printf("\033[31mERROR\033[0m Cannot create Source: %+v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Created Source %s\n", sourceName)
	},
}

var sourceDeleteCmd = &cobra.Command{
	Use:   "delete [flags]",
	Short: "Delete Odigos Sources",
	Long:  "This command will delete any Source objects that match the provided Workload info.",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		client := cmdcontext.KubeClientFromContextOrExit(ctx)

		namespaceText, providedWorkloadFlags, namespaceList, labelSet := parseSourceLabelFlags()

		if !cmd.Flag("yes").Changed {
			fmt.Printf("About to delete all Sources in %s that match:\n%s", namespaceText, providedWorkloadFlags)
			confirmed, err := confirm.Ask("Are you sure?")
			if err != nil || !confirmed {
				fmt.Println("Aborting delete")
				return
			}
		}

		sources, err := client.OdigosClient.Sources(namespaceList).List(ctx, v1.ListOptions{LabelSelector: labelSet.AsSelector().String()})
		if err != nil {
			fmt.Printf("\033[31mERROR\033[0m Cannot list Sources: %+v\n", err)
			os.Exit(1)
		}

		deletedCount := 0
		for _, source := range sources.Items {
			err := client.OdigosClient.Sources(source.GetNamespace()).Delete(ctx, source.GetName(), v1.DeleteOptions{})
			if err != nil {
				fmt.Printf("\033[31mERROR\033[0m Cannot delete Sources %s/%s: %+v\n", source.GetNamespace(), source.GetName(), err)
				os.Exit(1)
			}
			fmt.Printf("Deleted Source %s/%s\n", source.GetNamespace(), source.GetName())
			deletedCount++
		}
		fmt.Printf("Deleted %d Sources\n", deletedCount)
	},
}

var sourceUpdateCmd = &cobra.Command{
	Use:   "update [flags]",
	Short: "Update Odigos Sources",
	Long:  "This command will update any Source objects that match the provided Workload info.",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		client := cmdcontext.KubeClientFromContextOrExit(ctx)

		namespaceText, providedWorkloadFlags, namespaceList, labelSet := parseSourceLabelFlags()

		if !cmd.Flag("yes").Changed {
			fmt.Printf("About to update all Sources in %s that match:\n%s", namespaceText, providedWorkloadFlags)
			confirmed, err := confirm.Ask("Are you sure?")
			if err != nil || !confirmed {
				fmt.Println("Aborting delete")
				return
			}
		}

		sources, err := client.OdigosClient.Sources(namespaceList).List(ctx, v1.ListOptions{LabelSelector: labelSet.AsSelector().String()})
		if err != nil {
			fmt.Printf("\033[31mERROR\033[0m Cannot list Sources: %+v\n", err)
			os.Exit(1)
		}

		for _, source := range sources.Items {
			source.Spec.DisableInstrumentation = disableInstrumentationFlag
			if len(sourceRemoveGroupFlag) > 0 {
				for label, value := range source.Labels {
					if label == k8sconsts.SourceGroupLabelPrefix+sourceRemoveGroupFlag && value == "true" {
						delete(source.Labels, k8sconsts.SourceGroupLabelPrefix+sourceRemoveGroupFlag)
					}
				}
			}
			if len(sourceSetGroupFlag) > 0 {
				if source.Labels == nil {
					source.Labels = make(map[string]string)
				}
				source.Labels[k8sconsts.SourceGroupLabelPrefix+sourceSetGroupFlag] = "true"
			}
			_, err := client.OdigosClient.Sources(namespaceList).Update(ctx, &source, v1.UpdateOptions{})
			if err != nil {
				fmt.Printf("\033[31mERROR\033[0m Cannot update Sources %s/%s: %+v\n", source.GetNamespace(), source.GetName(), err)
				os.Exit(1)
			}
			fmt.Printf("Updated Source %s/%s\n", source.GetNamespace(), source.GetName())
		}
	},
}

func parseSourceLabelFlags() (string, string, string, labels.Set) {
	labelSet := labels.Set{}
	providedWorkloadFlags := ""
	if len(workloadKindFlag) > 0 {
		providedWorkloadFlags = fmt.Sprintf("%s Workload Kind: %s\n", providedWorkloadFlags, workloadKindFlag)
		labelSet[k8sconsts.WorkloadKindLabel] = workloadKindFlag
	}
	if len(workloadNameFlag) > 0 {
		providedWorkloadFlags = fmt.Sprintf("%s Workload Name: %s\n", providedWorkloadFlags, workloadNameFlag)
		labelSet[k8sconsts.WorkloadNameLabel] = workloadNameFlag
	}
	if len(workloadNamespaceFlag) > 0 {
		providedWorkloadFlags = fmt.Sprintf("%s Workload Namespace: %s\n", providedWorkloadFlags, workloadNamespaceFlag)
		labelSet[k8sconsts.WorkloadNamespaceLabel] = workloadNamespaceFlag
	}
	if len(sourceGroupFlag) > 0 {
		providedWorkloadFlags = fmt.Sprintf("%s Source Group: %s\n", providedWorkloadFlags, sourceGroupFlag)
		labelSet[k8sconsts.SourceGroupLabelPrefix+sourceGroupFlag] = "true"
	}
	namespaceList := namespaceFlag
	namespaceText := fmt.Sprintf("namespace %s", namespaceFlag)
	if allNamespaceFlag {
		namespaceText = "all namespaces"
		namespaceList = ""
	}
	return namespaceText, providedWorkloadFlags, namespaceList, labelSet
}

func init() {
	sourceFlags = pflag.NewFlagSet("sourceFlags", pflag.ContinueOnError)
	sourceFlags.StringVarP(&namespaceFlag, namespaceFlagName, "n", "default", "Kubernetes Namespace for Source")
	sourceFlags.StringVar(&workloadKindFlag, workloadKindFlagName, "", "Kubernetes Kind for entity (one of: Deployment, DaemonSet, StatefulSet, Namespace)")
	sourceFlags.StringVar(&workloadNameFlag, workloadNameFlagName, "", "Name of entity for Source")
	sourceFlags.StringVar(&workloadNamespaceFlag, workloadNamespaceFlagName, "", "Namespace of entity for Source")
	sourceFlags.StringVar(&sourceGroupFlag, sourceGroupFlagName, "", "Name of Source group to use")

	rootCmd.AddCommand(sourcesCmd)
	sourcesCmd.AddCommand(sourceCreateCmd)
	sourcesCmd.AddCommand(sourceDeleteCmd)
	sourcesCmd.AddCommand(sourceUpdateCmd)

	sourceCreateCmd.Flags().AddFlagSet(sourceFlags)
	sourceCreateCmd.Flags().BoolVar(&disableInstrumentationFlag, disableInstrumentationFlagName, false, "Disable instrumentation for Source")

	sourceDeleteCmd.Flags().AddFlagSet(sourceFlags)
	sourceDeleteCmd.Flags().Bool("yes", false, "skip the confirmation prompt")
	sourceDeleteCmd.Flags().Bool(allNamespacesFlagName, false, "apply to all Kubernetes namespaces")

	sourceUpdateCmd.Flags().AddFlagSet(sourceFlags)
	sourceUpdateCmd.Flags().BoolVar(&disableInstrumentationFlag, disableInstrumentationFlagName, false, "Disable instrumentation for Source")
	sourceUpdateCmd.Flags().StringVar(&sourceSetGroupFlag, sourceSetGroupFlagName, "", "Group name to be applied to the Source")
	sourceUpdateCmd.Flags().StringVar(&sourceRemoveGroupFlag, sourceRemoveGroupFlagName, "", "Group name to be removed from the Source (if set)")
	sourceUpdateCmd.Flags().Bool("yes", false, "skip the confirmation prompt")
	sourceUpdateCmd.Flags().Bool(allNamespacesFlagName, false, "apply to all Kubernetes namespaces")
}
