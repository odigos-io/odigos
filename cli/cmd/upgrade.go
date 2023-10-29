/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	"github.com/keyval-dev/odigos/cli/cmd/resources"
	"github.com/keyval-dev/odigos/cli/pkg/confirm"
	"github.com/keyval-dev/odigos/cli/pkg/kube"
	"github.com/keyval-dev/odigos/cli/pkg/log"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type VersionChangeType int

const (
	Upgrade VersionChangeType = iota
	Downgrade
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

		_, err = client.CoreV1().Secrets(ns).Get(ctx, resources.OdigosCloudSecretName, metav1.GetOptions{})
		notFound := errors.IsNotFound(err)
		if !notFound && err != nil {
			fmt.Println("Odigos upgrade failed - unable to check if odigos cloud is enabled")
			os.Exit(1)
		}
		isOdigosCloud := !notFound

		config, err := getConfig(ctx, client, ns)
		if err != nil {
			fmt.Println("Odigos upgrade failed - unable to read the current Odigos configuration.")
			os.Exit(1)
		}
		resourceManagers := resources.CreateResourceManagers(client, ns, versionFlag, isOdigosCloud, &config.Spec)

		for _, rm := range resourceManagers {
			l := log.Print(fmt.Sprintf("Upgrading Odigos %s", rm.Name()))
			err := rm.InstallFromScratch(ctx)
			if err != nil {
				l.Error(err)
				os.Exit(1)
			}
			l.Success()
		}

		resources := kube.GetManagedResources(ns)
		for _, resource := range resources {
			l := log.Print(fmt.Sprintf("Syncing %s", resource.Resource.Resource))
			err = client.DeleteOldOdigosSystemObjects(ctx, resource, versionFlag)
			if err != nil {
				l.Error(err)
				os.Exit(1)
			}
			l.Success()
		}
	},
}

func getConfig(ctx context.Context, client *kube.Client, ns string) (*v1alpha1.OdigosConfiguration, error) {
	return client.OdigosClient.OdigosConfigurations(ns).Get(ctx, resources.OdigosConfigName, metav1.GetOptions{})
}

func init() {
	rootCmd.AddCommand(upgradeCmd)
	if OdigosVersion != "" {
		versionFlag = OdigosVersion
	} else {
		installCmd.Flags().StringVar(&versionFlag, "version", OdigosVersion, "for development purposes only")
	}
}
