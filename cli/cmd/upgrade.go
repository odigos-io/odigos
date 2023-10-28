/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/keyval-dev/odigos/cli/cmd/resources"
	"github.com/keyval-dev/odigos/cli/pkg/confirm"
	"github.com/keyval-dev/odigos/cli/pkg/kube"
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

		// TODO: either read this from the cluster or from cli arguments
		telemetryEnabled := true
		sidecarInstrumentation := false
		ignoredNamespaces := []string{}

		resourceManagers := resources.CreateResourceManagers(client, ns, versionFlag, isOdigosCloud, telemetryEnabled, sidecarInstrumentation, ignoredNamespaces, psp)

		for _, rm := range resourceManagers {
			err := rm.InstallFromScratch(ctx)
			if err != nil {
				fmt.Printf("Odigos upgrade failed - unable to install Odigos: %s\n", err)
				os.Exit(1)
			}
		}

		// resourceManagers := []resources.ResourceManager{
		// 	resources.NewAutoScalerResourceManager(client, ns),
		// 	resources.NewSchedulerResourceManager(client, ns),
		// 	resources.NewInstrumentorResourceManager(client, ns),
		// 	resources.NewOdigletResourceManager(client, ns),
		// 	resources.NewOdigosDeploymentResourceManager(client, ns),
		// }

		// semverRange, versionChangeType, err := parseSourceAndTarget(odigosVersion, versionFlag)
		// if err != nil {
		// 	fmt.Printf("Odigos upgrade failed - unable to parse Odigos version: %s\n", err)
		// 	os.Exit(1)
		// }

		// err = applyRelevantMigrationSteps(ctx, resourceManagers, semverRange, versionChangeType)
		// if err != nil {
		// 	fmt.Printf("Odigos upgrade failed - error during applying migration steps: %s\n", err)
		// 	os.Exit(1)
		// }

		// err = upgradeImageTags(ctx, resourceManagers, versionFlag)
		// if err != nil {
		// 	fmt.Printf("Odigos upgrade failed - unable to upgrade Odigos version: %s\n", err)
		// 	os.Exit(1)
		// }

	},
}

// func applyRelevantMigrationSteps(ctx context.Context, resourceManagers []resources.ResourceManager, versionRange semver.Range, versionChangeType VersionChangeType) error {
// 	migrationsByVersion := make(map[string][]resources.MigrationStep)

// 	for _, rm := range resourceManagers {
// 		resourceMigrationSteps := rm.GetMigrationSteps()
// 		for _, migrationStep := range resourceMigrationSteps {
// 			sourceVersion := migrationStep.SourceVersion[1:]
// 			sourceVersionSemver := semver.MustParse(sourceVersion)

// 			// filter out just those versions which are relevant to us
// 			if versionRange(sourceVersionSemver) {
// 				migrationsByVersion[sourceVersion] = append(migrationsByVersion[sourceVersion], migrationStep)
// 			}
// 		}
// 	}

// 	// create a list of all the migration versions, and sort by version
// 	var migrationVersions semver.Versions
// 	for version := range migrationsByVersion {
// 		migrationVersions = append(migrationVersions, semver.MustParse(version))
// 	}
// 	semver.Sort(migrationVersions)

// 	if versionChangeType == Upgrade {
// 		for _, version := range migrationVersions {
// 			versionStr := version.String()
// 			migrationSteps := migrationsByVersion[versionStr]
// 			fmt.Printf("Applying %d migration steps for version %s:\n", len(migrationSteps), version)
// 			for _, migrationStep := range migrationSteps {
// 				for _, patcher := range migrationStep.Patchers {
// 					err := patcher.Patch(ctx)
// 					if err != nil {
// 						return err
// 					}
// 				}
// 			}
// 		}
// 	} else {
// 		for i := len(migrationVersions) - 1; i >= 0; i-- {
// 			version := migrationVersions[i]
// 			versionStr := version.String()
// 			migrationSteps := migrationsByVersion[versionStr]
// 			fmt.Printf("Rolling back %d migration steps for version %s:\n", len(migrationSteps), version)
// 			for j := len(migrationSteps) - 1; j >= 0; j-- {
// 				migrationStep := migrationSteps[j]
// 				for k := len(migrationStep.Patchers) - 1; k >= 0; k-- {
// 					patcher := migrationStep.Patchers[k]
// 					err := patcher.UnPatch(ctx)
// 					if err != nil {
// 						return err
// 					}
// 				}
// 			}
// 		}
// 	}

// 	return nil
// }

// func upgradeImageTags(ctx context.Context, resourceManagers []resources.ResourceManager, targetTag string) error {

// 	for _, rm := range resourceManagers {
// 		err := rm.PatchOdigosVersionToTarget(ctx, targetTag)
// 		if err != nil {
// 			fmt.Println("failed to upgrade Odigos version to target version")
// 			return err
// 		}
// 	}

// 	fmt.Println("Odigos upgrade complete - Odigos version is now", targetTag)

// 	return nil
// }

func init() {
	rootCmd.AddCommand(upgradeCmd)
	upgradeCmd.Flags().StringVar(&versionFlag, "version", OdigosVersion, "target version for Odigos upgrade")
}

// func parseSourceAndTarget(sourceVersion string, targetVersion string) (semver.Range, VersionChangeType, error) {
// 	// remove the trailing "v" from the odigosVersion
// 	sourceVersionBare := sourceVersion[1:]
// 	targetVersionBare := targetVersion[1:]
// 	isUpgrade := semver.MustParse(sourceVersionBare).LT(semver.MustParse(targetVersionBare))
// 	if isUpgrade {
// 		semverRange, err := semver.ParseRange(fmt.Sprintf(">=%s <%s", sourceVersionBare, targetVersionBare))
// 		return semverRange, Upgrade, err
// 	} else {
// 		semverRange, err := semver.ParseRange(fmt.Sprintf(">=%s <=%s", targetVersionBare, sourceVersionBare))
// 		return semverRange, Downgrade, err
// 	}
// }
