package cmd

import (
	"fmt"
	"os"

	"github.com/odigos-io/odigos/api/k8sconsts"

	"github.com/odigos-io/odigos/cli/cmd/resources"
	cmdcontext "github.com/odigos-io/odigos/cli/pkg/cmd_context"
	"github.com/odigos-io/odigos/cli/pkg/cmdutil"
	"github.com/odigos-io/odigos/cli/pkg/confirm"
	"github.com/odigos-io/odigos/common/consts"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/spf13/cobra"
)

// cleanupCmd represents the cleanup command
var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: `Remove Odigos Sources created by the user.`,
	Long: `This command removes all Odigos Source resources that were added by the user.
It runs as part of the cleanup job triggered during 'odigos uninstall'.
All other Odigos components and system resources are deleted automatically by Helm.`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		client := cmdcontext.KubeClientFromContextOrExit(ctx)

		nsFlag, err := cmd.Flags().GetString("namespace")
		if err != nil {
			fmt.Printf("\033[31mERROR\033[0m Failed to read namespace flag: %s\n", err)
			os.Exit(1)
		}
		var ns string
		if nsFlag != "" {
			ns = nsFlag
		} else {
			ns, err = resources.GetOdigosNamespace(client, ctx)
			if err != nil && !resources.IsErrNoOdigosNamespaceFound(err) {
				fmt.Printf("\033[31mERROR\033[0m Failed to check if Odigos is already cleaned up: %s\n", err)
				os.Exit(1)
			}
		}

		if err != nil && !resources.IsErrNoOdigosNamespaceFound(err) {
			fmt.Printf("\033[31mERROR\033[0m Failed to check if Odigos is already cleaned up: %s\n", err)
			os.Exit(1)
		}
		odigosNsFound := !resources.IsErrNoOdigosNamespaceFound(err)

		if odigosNsFound {
			if !cmd.Flag("yes").Changed {
				fmt.Printf("About to cleanup Odigos from namespace %s\n", ns)
				confirmed, err := confirm.Ask("Are you sure?")
				if err != nil || !confirmed {
					fmt.Println("Aborting cleanup")
					return
				}
			}

			config, err := resources.GetCurrentConfig(ctx, client, ns)
			if err != nil {
				fmt.Println("Failed to get current Odigos configuration, assuming default values for cleanup...")
			}

			autoRolloutDisabled := false
			if config != nil {
				autoRolloutDisabled = config.Rollout != nil &&
					config.Rollout.AutomaticRolloutDisabled != nil &&
					*config.Rollout.AutomaticRolloutDisabled
			}

			// delete all sources, and wait for the pods to rollout without instrumentation
			// this is done before the instrumentor is removed, to ensure that the instrumentation is removed

			err = removeAllSources(ctx, client)
			if err != nil {
				fmt.Printf("\033[31mERROR\033[0m Failed to remove all sources for cleanup: %s\n", err)
				os.Exit(1)
			}
			if autoRolloutDisabled {
				fmt.Println("Odigos is configured to NOT rollout workloads automatically; existing pods will remain instrumented until a manual rollout is triggered.")
			} else if !cmd.Flag("no-wait").Changed {
				err = waitForPodsToRolloutWithoutInstrumentation(ctx, client)
				if err != nil {
					fmt.Printf("\033[31mERROR\033[0m Failed to wait for pods to rollout without instrumentation: %s\n", err)
					os.Exit(1)
				}
			}

			// If the user only wants to uninstall instrumentation, we exit here.
			// This flag being used by users who want to remove instrumentation without removing the entire Odigos setup,
			// And by cleanup jobs that runs as helm pre-uninstall hook before helm uninstall command.
			if cmd.Flag("instrumentation-only").Changed {
				fmt.Println("Cleaning up Odigos instrumentation resources... new approeach")
				// Node labels are added by the Odiglet, and since it's not managed by Helm, we need to clean them up here.
				// In CLI logic, this is done in UninstallClusterResources after the Odiglet is deleted.
				cmdutil.CreateKubeResourceWithLogging(ctx, "Cleaning up Odigos node labels",
					client, ns, k8sconsts.OdigosSystemLabelKey, cleanupNodeOdigosLabels)
				// MIGRATION: In older versions of Odigos, a legacy ConfigMap named "odigos-config" was used.
				// It has since been replaced by "odigos-configuration", which is Helm-managed and does not include hook annotations.
				// As part of the migration, we explicitly delete the legacy ConfigMap if it still exists.
				config, err := client.CoreV1().ConfigMaps(ns).Get(ctx, consts.OdigosLegacyConfigName, metav1.GetOptions{})
				if err != nil && apierrors.IsNotFound(err) {
					// If the ConfigMap does not exist, we can safely exit.
					fmt.Printf("\n\u001B[32mSUCCESS:\u001B[0m Odigos uninstalled instrumentation resources successfuly\n")
					return
				} else if err != nil {
					fmt.Printf("\033[31mERROR\033[0m Failed to get legacy Odigos config ConfigMap %s in namespace %s: %v\n", consts.OdigosLegacyConfigName, ns, err)
					os.Exit(1)
				}
				if val, ok := config.Labels[k8sconsts.AppManagedByHelmLabel]; ok && val == k8sconsts.AppManagedByHelmValue {
					err := client.CoreV1().ConfigMaps(ns).Delete(ctx, consts.OdigosLegacyConfigName, metav1.DeleteOptions{})
					if err != nil {
						fmt.Printf("\033[31mERROR\033[0m Failed to delete legacy Odigos config ConfigMap %s in namespace %s: %v\n", consts.OdigosLegacyConfigName, ns, err)
						os.Exit(1)
					} else {
						fmt.Printf("Deleted legacy Odigos config ConfigMap %s in namespace %s\n", consts.OdigosLegacyConfigName, ns)
					}
				}
				fmt.Printf("\n\u001B[32mSUCCESS:\u001B[0m Odigos uninstalled instrumentation resources successfuly\n")
				return
			}

		} else {
			fmt.Println("Odigos is not installed in any namespace. cleaning up any other Odigos resources that might be left in the cluster...")
		}

	},
	Example: `
# Cleanup Odigos open-source or cloud from the cluster in your kubeconfig active context.
odigos cleanup

# Cleanup Odigos without confirmation
odigos cleanup --yes

# Cleanup Odigos without waiting for pods to rollout without instrumentation
odigos cleanup --no-wait

`,
}

func init() {
	rootCmd.AddCommand(cleanupCmd)
	cleanupCmd.Flags().Bool("yes", false, "skip the confirmation prompt")
	cleanupCmd.Flags().Bool("no-wait", false, "skip waiting for pods to rollout without instrumentation")
	cleanupCmd.Flags().Bool("instrumentation-only", false, "only remove instrumentation from workloads, without removing the entire Odigos setup")
	cleanupCmd.Flags().StringP("namespace", "n", "", "namespace to uninstall Odigos from (overrides auto-detection)")

}
