/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/keyval-dev/odigos/cli/cmd/resources"
	"github.com/keyval-dev/odigos/cli/pkg/confirm"
	"github.com/keyval-dev/odigos/cli/pkg/kube"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// upgradeCmd represents the upgrade command
var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade Odigos version",
	Long: `Use a specific version of Odigos.

This command will upgrade the Odigos version in the cluster to the specified version
and apply any required migrations.`,
	Run: func(cmd *cobra.Command, args []string) {

		client, err := kube.CreateClient(cmd)
		if err != nil {
			kube.PrintClientErrorAndExit(err)
		}
		ctx := cmd.Context()

		ns, err := resources.GetOdigosNamespace(client, ctx)
		if err != nil {
			fmt.Println("No Odigos installation found in cluster to upgrade")
			os.Exit(1)
		}

		cm, err := client.CoreV1().ConfigMaps(ns).Get(ctx, resources.OdigosDeploymentConfigMapName, metav1.GetOptions{})
		if err != nil {
			fmt.Println("Odigos upgrade failed - unable to read the current Odigos version for migration")
			os.Exit(1)
		}

		odigosVersion := cm.Data["ODIGOS_VERSION"]
		if odigosVersion == "" {
			fmt.Println("Odigos upgrade failed - unable to read the current Odigos version for migration")
			os.Exit(1)
		}

		fmt.Printf("About to upgrade Odigos version from '%s' (current) to '%s' (target)\n", odigosVersion, versionFlag)
		confirmed, err := confirm.Ask("Are you sure?")
		if err != nil || !confirmed {
			fmt.Println("Aborting upgrade")
			return
		}

		err = upgradeImageTags(ctx, client, ns, versionFlag)
		if err != nil {
			fmt.Printf("Odigos upgrade failed - unable to upgrade Odigos version: %s\n", err)
			os.Exit(1)
		}

	},
}

func upgradeImageTags(ctx context.Context, client *kube.Client, ns string, targetTag string) error {

	resourceManagers := []resources.ResourceManager{
		resources.NewAutoScalerResourceManager(client, ns),
		resources.NewSchedulerResourceManager(client, ns),
		resources.NewInstrumentorResourceManager(client, ns),
		resources.NewOdigletResourceManager(client, ns),
	}

	for _, rm := range resourceManagers {
		err := rm.PatchOdigosVersionToTarget(ctx, targetTag)
		if err != nil {
			fmt.Println("failed to upgrade Odigos version to target version")
			return err
		}
	}

	fmt.Println("Odigos upgrade complete - Odigos version is now", targetTag)

	return nil
}

func init() {
	rootCmd.AddCommand(upgradeCmd)
	upgradeCmd.Flags().StringVar(&versionFlag, "version", OdigosVersion, "target version for Odigos upgrade")
}
