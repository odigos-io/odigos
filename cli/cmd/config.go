package cmd

import (
	"context"
	"fmt"
	"os"

	cmdcontext "github.com/odigos-io/odigos/cli/pkg/cmd_context"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/cli/pkg/log"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage Odigos configuration",
	Long:  "Manage Odigos configuration settings, including central backend URL and other properties",
}

// `odigos config set <property> <value>`
var setConfigCmd = &cobra.Command{
	Use:   "set <property> <value>",
	Short: "Set a configuration property in Odigos",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		property := args[0]
		value := args[1]

		ctx := cmd.Context()
		client := cmdcontext.KubeClientFromContextOrExit(ctx)
		ns, _ := cmd.Flags().GetString("namespace")
		if ns == "" {
			ns = "odigos-system"
		}

		l := log.Print(fmt.Sprintf("Updating %s to %s...", property, value))
		err := updateConfigProperty(ctx, client, ns, property, value)
		if err != nil {
			l.Error(err)
			os.Exit(1)
		}
		l.Success()

		if property == "central-backend-url" {
			l = log.Print("Restarting UI pod to apply changes...")
			err = restartUIPod(ctx, client, ns)
			if err != nil {
				l.Error(err)
				os.Exit(1)
			}
			l.Success()
		}
	},
}

func updateConfigProperty(ctx context.Context, client *kube.Client, ns, property, value string) error {
	configMapName := "odigos-config"

	cm, err := client.CoreV1().ConfigMaps(ns).Get(ctx, configMapName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get ConfigMap: %w", err)
	}

	cm.Data[property] = value
	_, err = client.CoreV1().ConfigMaps(ns).Update(ctx, cm, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update ConfigMap: %w", err)
	}

	return nil
}

func restartUIPod(ctx context.Context, client *kube.Client, ns string) error {
	pods, err := client.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{LabelSelector: "app=odigos-ui"})
	if err != nil {
		return fmt.Errorf("failed to list UI pods: %w", err)
	}

	for _, pod := range pods.Items {
		err := client.CoreV1().Pods(ns).Delete(ctx, pod.Name, metav1.DeleteOptions{})
		if err != nil {
			return fmt.Errorf("failed to delete pod %s: %w", pod.Name, err)
		}
	}

	return nil
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(setConfigCmd)

	setConfigCmd.Flags().StringP("namespace", "n", "odigos-system", "Namespace where Odigos is installed")
}
