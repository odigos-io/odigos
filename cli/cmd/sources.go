package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/cli/cmd/sources_utils"
	cmdcontext "github.com/odigos-io/odigos/cli/pkg/cmd_context"
	"github.com/odigos-io/odigos/cli/pkg/confirm"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	sourceFlags *pflag.FlagSet

	namespaceFlagName   = "namespace"
	sourceNamespaceFlag string

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

	sourceOtelServiceFlagName = "otel-service"
	sourceOtelServiceFlag     string
)

var sourcesCmd = &cobra.Command{
	Use:   "sources [command] [flags]",
	Short: "Manage Odigos Sources in a cluster",
	Long:  "This command can be used to create, delete, or update Sources to configure workload or namespace auto-instrumentation",
	Example: `# Create a Source "foo-source" for deployment "foo" in namespace "default"
odigos sources create foo-source --workload-kind=Deployment --workload-name=foo --workload-namespace=default -n default

# Update all existing Sources in namespace "default" to disable instrumentation
odigos sources update --disable-instrumentation -n default

# Delete all Sources in group "mygroup"
odigos sources delete --group mygroup --all-namespaces
	`,
}

var kindAliases = map[k8sconsts.WorkloadKind][]string{
	k8sconsts.WorkloadKindDeployment:  []string{"deploy", "deployments", "deploy.apps", "deployment.apps", "deployments.apps"},
	k8sconsts.WorkloadKindDaemonSet:   []string{"ds", "daemonsets", "ds.apps", "daemonset.apps", "daemonsets.apps"},
	k8sconsts.WorkloadKindStatefulSet: []string{"sts", "statefulsets", "sts.apps", "statefulset.apps", "statefulsets.apps"},
	k8sconsts.WorkloadKindNamespace:   []string{"ns", "namespaces"},
}

var sourceDisableCmd = &cobra.Command{
	Use:   "disable [workload type] [workload name] [flags]",
	Short: "Disable a source for Odigos instrumentation.",
	Long:  "This command disables the given workload for Odigos instrumentation. It will create a Source object (if one does not already exist)",
	Example: `
# Disable deployment "foo" in namespace "default"
odigos sources disable deployment foo

# Disable namespace "bar" in namespace "default"
odigos sources disable namespace bar

# Disable statefulset "foo" in namespace "bar"
odigos sources disable statefulset foo -n bar
`,
}

var sourceEnableCmd = &cobra.Command{
	Use:   "enable [workload type] [workload name] [flags]",
	Short: "Enable a source for Odigos instrumentation.",
	Long:  "This command enables the given workload for Odigos instrumentation. It will create a Source object (if one does not already exist)",
	Example: `
# Enable deployment "foo" in namespace "default"
odigos sources enable deployment foo

# Enable namespace "bar" in namespace "default"
odigos sources enable namespace bar

# Enable statefulset "foo" in namespace "bar"
odigos sources enable statefulset foo -n bar
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
				Namespace: sourceNamespaceFlag,
			},
			Spec: v1alpha1.SourceSpec{
				Workload: k8sconsts.PodWorkload{
					Kind:      k8sconsts.WorkloadKind(workloadKindFlag),
					Name:      workloadNameFlag,
					Namespace: workloadNamespaceFlag,
				},
				DisableInstrumentation: disableInstrumentation,
				OtelServiceName:        sourceOtelServiceFlag,
			},
		}

		if len(sourceGroupFlag) > 0 {
			source.Labels = make(map[string]string)
			source.Labels[k8sconsts.SourceGroupLabelPrefix+sourceGroupFlag] = "true"
		}

		_, err := client.OdigosClient.Sources(sourceNamespaceFlag).Create(ctx, source, v1.CreateOptions{})
		if err != nil {
			fmt.Printf("\033[31mERROR\033[0m Cannot create Source: %+v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Created Source %s\n", sourceName)
	},
}

var sourceDeleteCmd = &cobra.Command{
	Use:   "delete [name] [flags]",
	Short: "Delete Odigos Sources",
	Long: `This command will delete the named Source object or any Source objects that match the provided Workload info.
If a [name] is provided, that Source object will be deleted in the given namespace using the --namespace (-n) flag.`,
	Example: `For example, to delete the Source named "mysource-abc123" in namespace "myapp", run:

$ odigos sources delete mysource-abc123 -n myapp

Multiple Source objects can be deleted at once using the --workload-name, --workload-kind, and --workload-namespace flags.
These flags are AND-ed so that if any of these flags are provided, all Sources that match the given flags will be deleted.

For example, to delete all Sources for StatefulSet workloads in the cluster, run:

$ odigos sources delete --workload-kind=StatefulSet --all-namespaces

To delete all Deployment Sources in namespace Foobar, run:

$ odigos sources delete --workload-kind=Deployment --workload-namespace=Foobar

or

$ odigos sources delete --workload-kind=Deployment -n Foobar

These flags can be used to batch delete Sources, or as an alternative to deleting a Source by name (for instance, when
the name of the Source might not be known, but the Workload information is known). For example:

$ odigos sources delete --workload-kind=Deployment --workload-name=myapp -n myapp-namespace

This command will delete any Sources in the namespace "myapp-namespace" that instrument a Deployment named "myapp"

It is important to note that if a Source [name] is provided, all --workload-* flags will be ignored to delete only the named Source.
`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		client := cmdcontext.KubeClientFromContextOrExit(ctx)

		if len(args) > 0 {
			sourceName := args[0]
			fmt.Printf("Deleting Source %s in namespace %s\n", sourceName, sourceNamespaceFlag)
			err := client.OdigosClient.Sources(sourceNamespaceFlag).Delete(ctx, sourceName, v1.DeleteOptions{})
			if err != nil {
				fmt.Printf("\033[31mERROR\033[0m Cannot delete source %s in namespace %s: %+v\n", sourceName, sourceNamespaceFlag, err)
				os.Exit(1)
			} else {
				fmt.Printf("Deleted source %s in namespace %s\n", sourceName, sourceNamespaceFlag)
			}
		} else {
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
		}
	},
}

var sourceUpdateCmd = &cobra.Command{
	Use:   "update [name] [flags]",
	Short: "Update Odigos Sources",
	Long: `This command will update the named Source object or any Source objects that match the provided Workload info.
If a [name] is provided, that Source object will be updated in the given namespace using the --namespace (-n) flag.`,
	Example: `For example, to update the Source named "mysource-abc123" in namespace "myapp", run:

$ odigos sources update mysource-abc123 -n myapp <flags>

Multiple Source objects can be updated at once using the --workload-name, --workload-kind, and --workload-namespace flags.
These flags are AND-ed so that if any of these flags are provided, all Sources that match the given flags will be updated.

For example, to update all Sources for StatefulSet workloads in the cluster, run:

$ odigos sources update --workload-kind=StatefulSet --all-namespaces <flags>

To update all Deployment Sources in namespace Foobar, run:

$ odigos sources update --workload-kind=Deployment --workload-namespace=Foobar <flags>

or

$ odigos sources update --workload-kind=Deployment -n Foobar <flags>

These flags can be used to batch update Sources, or as an alternative to updating a Source by name (for instance, when
the name of the Source might not be known, but the Workload information is known). For example:

$ odigos sources update --workload-kind=Deployment --workload-name=myapp -n myapp-namespace <flags>

This command will update any Sources in the namespace "myapp-namespace" that instrument a Deployment named "myapp"

It is important to note that if a Source [name] is provided, all --workload-* flags will be ignored to update only the named Source.
`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		client := cmdcontext.KubeClientFromContextOrExit(ctx)

		sourceList := &v1alpha1.SourceList{}
		if len(args) > 0 {
			sourceName := args[0]
			sources, err := client.OdigosClient.Sources(sourceNamespaceFlag).List(ctx, v1.ListOptions{FieldSelector: "metadata.name=" + sourceName})
			if err != nil {
				fmt.Printf("\033[31mERROR\033[0m Cannot list Source %s: %+v\n", sourceName, err)
				os.Exit(1)
			}
			sourceList = sources
		} else {
			namespaceText, providedWorkloadFlags, namespaceList, labelSet := parseSourceLabelFlags()

			if !cmd.Flag("yes").Changed {
				fmt.Printf("About to update all Sources in %s that match:\n%s", namespaceText, providedWorkloadFlags)
				confirmed, err := confirm.Ask("Are you sure?")
				if err != nil || !confirmed {
					fmt.Println("Aborting update")
					return
				}
			}

			sources, err := client.OdigosClient.Sources(namespaceList).List(ctx, v1.ListOptions{LabelSelector: labelSet.AsSelector().String()})
			if err != nil {
				fmt.Printf("\033[31mERROR\033[0m Cannot list Sources: %+v\n", err)
				os.Exit(1)
			}
			sourceList = sources
		}

		for _, source := range sourceList.Items {
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

			if len(sourceOtelServiceFlag) > 0 {
				source.Spec.OtelServiceName = sourceOtelServiceFlag
			}

			_, err := client.OdigosClient.Sources(source.GetNamespace()).Update(ctx, &source, v1.UpdateOptions{})
			if err != nil {
				fmt.Printf("\033[31mERROR\033[0m Cannot update Sources %s/%s: %+v\n", source.GetNamespace(), source.GetName(), err)
				os.Exit(1)
			}
			fmt.Printf("Updated Source %s/%s\n", source.GetNamespace(), source.GetName())
		}
	},
}

var errorOnly bool

var sourceStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the status of all Odigos Sources",
	Long:  "Lists all InstrumentationConfigs and InstrumentationInstances with their current status. Use --error to filter only failed sources.",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		statuses, err := sources_utils.SourcesStatus(ctx)
		if err != nil {
			fmt.Printf("\033[31mERROR\033[0m Failed to retrieve source statuses: %+v\n", err)
			return
		}

		if errorOnly {
			var filteredStatuses []sources_utils.SourceStatus
			for _, s := range statuses {
				if s.IsError {
					filteredStatuses = append(filteredStatuses, s)
				}
			}
			statuses = filteredStatuses
		}

		fmt.Println("\n\033[33mOdigos Source Status:\033[0m")
		w := tabwriter.NewWriter(os.Stdout, 20, 4, 2, ' ', tabwriter.TabIndent)

		fmt.Fprintln(w, "NAMESPACE\tNAME\tSTATUS\tMESSAGE")

		for _, status := range statuses {
			color := "\033[32m"
			if status.IsError {
				color = "\033[31m"
			}

			fmt.Fprintf(w, "%s%s\t%s\t%s\t%s\033[0m\n",
				color, status.Namespace, status.Name, status.Status, status.Message)
		}

		w.Flush()
	},
}

func enableOrDisableSource(cmd *cobra.Command, args []string, workloadKind k8sconsts.WorkloadKind, disableInstrumentation bool) {
	msg := "enable"
	if disableInstrumentation {
		msg = "disable"
	}

	ctx := cmd.Context()
	client := cmdcontext.KubeClientFromContextOrExit(ctx)
	source, err := updateOrCreateSourceForObject(ctx, client, workloadKind, args[0], disableInstrumentation)
	if err != nil {
		fmt.Printf("\033[31mERROR\033[0m Cannot %s Source: %+v\n", msg, err)
		os.Exit(1)
	}
	fmt.Printf("%sd Source %s for %s %s\n", msg, source.GetName(), source.Spec.Workload.Kind, source.Spec.Workload.Name)
}

func enableOrDisableSourceCmd(workloadKind k8sconsts.WorkloadKind, disableInstrumentation bool) *cobra.Command {
	msg := "enable"
	if disableInstrumentation {
		msg = "disable"
	}

	return &cobra.Command{
		Use:     fmt.Sprintf("%s [name]", workload.WorkloadKindLowerCaseFromKind(workloadKind)),
		Short:   fmt.Sprintf("%s a %s for Odigos instrumentation", msg, workloadKind),
		Long:    fmt.Sprintf("This command %ss the provided %s for Odigos instrumentatin. It will create a Source object if one does not already exists, or update the existing one if it does.", msg, workloadKind),
		Args:    cobra.ExactArgs(1),
		Aliases: kindAliases[workloadKind],
		Run: func(cmd *cobra.Command, args []string) {
			enableOrDisableSource(cmd, args, workloadKind, disableInstrumentation)
		},
	}
}

func updateOrCreateSourceForObject(ctx context.Context, client *kube.Client, workloadKind k8sconsts.WorkloadKind, argName string, disableInstrumentation bool) (*v1alpha1.Source, error) {
	var err error
	obj := workload.ClientObjectFromWorkloadKind(workloadKind)
	var objName, objNamespace, sourceNamespace string
	switch workloadKind {
	case k8sconsts.WorkloadKindDaemonSet:
		obj, err = client.Clientset.AppsV1().DaemonSets(sourceNamespaceFlag).Get(ctx, argName, metav1.GetOptions{})
		objName = obj.GetName()
		objNamespace = obj.GetNamespace()
		sourceNamespace = sourceNamespaceFlag
	case k8sconsts.WorkloadKindDeployment:
		obj, err = client.Clientset.AppsV1().Deployments(sourceNamespaceFlag).Get(ctx, argName, metav1.GetOptions{})
		objName = obj.GetName()
		objNamespace = obj.GetNamespace()
		sourceNamespace = sourceNamespaceFlag
	case k8sconsts.WorkloadKindStatefulSet:
		obj, err = client.Clientset.AppsV1().StatefulSets(sourceNamespaceFlag).Get(ctx, argName, metav1.GetOptions{})
		objName = obj.GetName()
		objNamespace = obj.GetNamespace()
		sourceNamespace = sourceNamespaceFlag
	case k8sconsts.WorkloadKindNamespace:
		obj, err = client.Clientset.CoreV1().Namespaces().Get(ctx, argName, metav1.GetOptions{})
		objName = obj.GetName()
		objNamespace = obj.GetName()
		sourceNamespace = obj.GetName()
	}
	if err != nil {
		return nil, err
	}

	var source *v1alpha1.Source
	selector := labels.SelectorFromSet(labels.Set{
		k8sconsts.WorkloadNameLabel:      obj.GetName(),
		k8sconsts.WorkloadNamespaceLabel: sourceNamespace,
		k8sconsts.WorkloadKindLabel:      string(workloadKind),
	})
	sources, err := client.OdigosClient.Sources(sourceNamespace).List(ctx, metav1.ListOptions{LabelSelector: selector.String()})
	if len(sources.Items) > 0 {
		source = &sources.Items[0]
	} else {
		source = &v1alpha1.Source{
			ObjectMeta: v1.ObjectMeta{
				GenerateName: workload.CalculateWorkloadRuntimeObjectName(objName, workloadKind),
				Namespace:    sourceNamespace,
			},
			Spec: v1alpha1.SourceSpec{
				Workload: k8sconsts.PodWorkload{
					Kind:      workloadKind,
					Name:      objName,
					Namespace: objNamespace,
				},
			},
		}

	}

	source.Spec.DisableInstrumentation = disableInstrumentation

	if len(sources.Items) > 0 {
		source, err = client.OdigosClient.Sources(sourceNamespace).Update(ctx, source, v1.UpdateOptions{})
	} else {
		source, err = client.OdigosClient.Sources(sourceNamespace).Create(ctx, source, v1.CreateOptions{})
	}
	if err != nil {
		return nil, err
	}

	if workloadKind == k8sconsts.WorkloadKindNamespace {
		// if toggling a namespace, check for individually instrumented workloads
		// alert the user that these workloads won't be affected by the command
		selector := labels.NewSelector()
		workloadKindRequirement, err := labels.NewRequirement(k8sconsts.WorkloadKindLabel, selection.NotIn, []string{string(k8sconsts.WorkloadKindNamespace)})
		if err != nil {
			return nil, err
		}
		selector.Add(*workloadKindRequirement)
		sources, err := client.OdigosClient.Sources(sourceNamespace).List(ctx, metav1.ListOptions{LabelSelector: selector.String()})
		if len(sources.Items) > 0 {
			fmt.Printf("NOTE: Configured Namespace Source, but the following Workload Sources will not be affected:\n")
			for _, source := range sources.Items {
				fmt.Printf("Source: %s (Workload=%s, Kind=%s)\n", source.GetName(), source.Spec.Workload.Name, source.Spec.Workload.Kind)
			}
		}

	}
	return source, nil
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
	namespaceList := sourceNamespaceFlag
	namespaceText := fmt.Sprintf("namespace %s", sourceNamespaceFlag)
	if allNamespaceFlag {
		namespaceText = "all namespaces"
		namespaceList = ""
	}
	return namespaceText, providedWorkloadFlags, namespaceList, labelSet
}

func init() {
	sourceFlags = pflag.NewFlagSet("sourceFlags", pflag.ContinueOnError)
	sourceFlags.StringVarP(&sourceNamespaceFlag, namespaceFlagName, "n", "default", "Kubernetes Namespace for Source")
	sourceFlags.StringVar(&workloadKindFlag, workloadKindFlagName, "", "Kubernetes Kind for entity (one of: Deployment, DaemonSet, StatefulSet, Namespace)")
	sourceFlags.StringVar(&workloadNameFlag, workloadNameFlagName, "", "Name of entity for Source")
	sourceFlags.StringVar(&workloadNamespaceFlag, workloadNamespaceFlagName, "", "Namespace of entity for Source")
	sourceFlags.StringVar(&sourceGroupFlag, sourceGroupFlagName, "", "Name of Source group to use")

	rootCmd.AddCommand(sourcesCmd)
	sourcesCmd.AddCommand(sourceCreateCmd)
	sourcesCmd.AddCommand(sourceDeleteCmd)
	sourcesCmd.AddCommand(sourceUpdateCmd)
	sourcesCmd.AddCommand(sourceStatusCmd)

	sourcesCmd.AddCommand(sourceEnableCmd)
	sourceEnableCmd.PersistentFlags().StringVarP(&sourceNamespaceFlag, namespaceFlagName, "n", "default", "Kubernetes Namespace for Source")
	sourceEnableCmd.AddCommand(enableOrDisableSourceCmd(k8sconsts.WorkloadKindDeployment, false))
	sourceEnableCmd.AddCommand(enableOrDisableSourceCmd(k8sconsts.WorkloadKindDaemonSet, false))
	sourceEnableCmd.AddCommand(enableOrDisableSourceCmd(k8sconsts.WorkloadKindStatefulSet, false))
	sourceEnableCmd.AddCommand(enableOrDisableSourceCmd(k8sconsts.WorkloadKindNamespace, false))
	sourcesCmd.AddCommand(sourceDisableCmd)
	sourceDisableCmd.PersistentFlags().StringVarP(&sourceNamespaceFlag, namespaceFlagName, "n", "default", "Kubernetes Namespace for Source")
	sourceDisableCmd.AddCommand(enableOrDisableSourceCmd(k8sconsts.WorkloadKindDeployment, true))
	sourceDisableCmd.AddCommand(enableOrDisableSourceCmd(k8sconsts.WorkloadKindDaemonSet, true))
	sourceDisableCmd.AddCommand(enableOrDisableSourceCmd(k8sconsts.WorkloadKindStatefulSet, true))
	sourceDisableCmd.AddCommand(enableOrDisableSourceCmd(k8sconsts.WorkloadKindNamespace, true))

	sourceCreateCmd.Flags().AddFlagSet(sourceFlags)
	sourceCreateCmd.Flags().BoolVar(&disableInstrumentationFlag, disableInstrumentationFlagName, false, "Disable instrumentation for Source")
	sourceCreateCmd.Flags().StringVar(&sourceOtelServiceFlag, sourceOtelServiceFlagName, "", "OpenTelemetry service name to use for the Source")

	sourceDeleteCmd.Flags().AddFlagSet(sourceFlags)
	sourceDeleteCmd.Flags().Bool("yes", false, "skip the confirmation prompt")
	sourceDeleteCmd.Flags().Bool(allNamespacesFlagName, false, "apply to all Kubernetes namespaces")

	sourceUpdateCmd.Flags().AddFlagSet(sourceFlags)
	sourceUpdateCmd.Flags().BoolVar(&disableInstrumentationFlag, disableInstrumentationFlagName, false, "Disable instrumentation for Source")
	sourceUpdateCmd.Flags().StringVar(&sourceSetGroupFlag, sourceSetGroupFlagName, "", "Group name to be applied to the Source")
	sourceUpdateCmd.Flags().StringVar(&sourceRemoveGroupFlag, sourceRemoveGroupFlagName, "", "Group name to be removed from the Source (if set)")
	sourceUpdateCmd.Flags().Bool("yes", false, "skip the confirmation prompt")
	sourceUpdateCmd.Flags().Bool(allNamespacesFlagName, false, "apply to all Kubernetes namespaces")
	sourceUpdateCmd.Flags().StringVar(&sourceOtelServiceFlag, sourceOtelServiceFlagName, "", "OpenTelemetry service name to use for the Source")

	sourceStatusCmd.Flags().BoolVar(&errorOnly, "error", false, "Show only sources with errors")

}
