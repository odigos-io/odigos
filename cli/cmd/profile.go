package cmd

import (
	"fmt"
	"os"
	"slices"

	"github.com/odigos-io/odigos/cli/cmd/resources"
	"github.com/odigos-io/odigos/cli/cmd/resources/odigospro"
	cmdcontext "github.com/odigos-io/odigos/cli/pkg/cmd_context"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/k8sutils/pkg/getters"
	"github.com/odigos-io/odigos/k8sutils/pkg/installationmethod"
	"github.com/odigos-io/odigos/profiles"
	"github.com/odigos-io/odigos/profiles/profile"
	"github.com/spf13/cobra"
)

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage presets of applied profiles to your odigos installation",
	Long:  `This command can be used to interact with the applied profiles in your odigos installation.`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		client := cmdcontext.KubeClientFromContextOrExit(ctx)

		ns, err := resources.GetOdigosNamespace(client, ctx)
		if resources.IsErrNoOdigosNamespaceFound(err) {
			fmt.Println("\033[31mERROR\033[0m no odigos installation found in the current cluster. use \"odigos install\" to install odigos in the cluster or check that kubeconfig is pointing to the correct cluster.")
			os.Exit(1)
		} else if err != nil {
			fmt.Printf("\033[31mERROR\033[0m Failed to check if Odigos is already installed: %s\n", err)
			os.Exit(1)
		}

		currentTier, err := odigospro.GetCurrentOdigosTier(ctx, client, ns)
		if err != nil {
			fmt.Println("Error reading current odigos tier")
			os.Exit(1)
		}

		availableFlag, err := cmd.Flags().GetBool("available")
		if err != nil {
			fmt.Println("\033[31mERROR\033[0m Unable to parse available flag:", err)
			os.Exit(1)
		}

		if availableFlag {
			fmt.Println("Listing available profiles for", currentTier, "tier:")
			profiles := profiles.GetAvailableProfilesForTier(currentTier)
			if len(profiles) == 0 {
				fmt.Println("No profiles are available for the current tier")
				os.Exit(0)
			}
			for _, profile := range profiles {
				fmt.Println("-", profile.ProfileName, " - ", profile.ShortDescription)
			}
			return
		}

		config, err := resources.GetCurrentConfig(ctx, client, ns)
		if err != nil {
			fmt.Println("Odigos profile unavailable - no configuration found")
			os.Exit(1)
		}

		if len(config.Profiles) == 0 {
			fmt.Println("No profiles are currently applied")
			os.Exit(0)
		}

		fmt.Println("Currently applied profiles:", config.Profiles)
	},
	Example: `
# Enable payload collection for all supported workloads and instrumentation libraries in the cluster
odigos profile add full-payload-collection

# Remove the full-payload-collection profile from the cluster
odigos profile remove full-payload-collection
`,
}

var addProfileCmd = &cobra.Command{
	Use:   "add <profile_name>",
	Short: "Add a profile to the current Odigos installation",
	Long:  `Add a profile by its name to the current Odigos installation.`,
	Args:  cobra.ExactArgs(1), // Ensure exactly one argument is passed (the profile name)
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		client := cmdcontext.KubeClientFromContextOrExit(ctx)

		ns, err := resources.GetOdigosNamespace(client, ctx)
		if resources.IsErrNoOdigosNamespaceFound(err) {
			fmt.Println("\033[31mERROR\033[0m No odigos installation found in the current cluster. Use \"odigos install\" to install odigos in the cluster or check that kubeconfig is pointing to the correct cluster.")
			os.Exit(1)
		} else if err != nil {
			fmt.Printf("\033[31mERROR\033[0m Failed to check if Odigos is already installed: %s\n", err)
			os.Exit(1)
		}

		currentOdigosVersion, err := getters.GetOdigosVersionInClusterFromConfigMap(ctx, client.Clientset, ns)
		if err != nil {
			fmt.Println("Odigos cloud login failed - unable to read the current Odigos version.")
			os.Exit(1)
		}

		profileName := args[0]
		currentTier, err := odigospro.GetCurrentOdigosTier(ctx, client, ns)
		if err != nil {
			fmt.Println("\033[31mERROR\033[0m Failed to retrieve current Odigos tier:", err)
			os.Exit(1)
		}

		profiles := profiles.GetAvailableProfilesForTier(currentTier)
		selectedProfile := profile.FindProfileByName(common.ProfileName(profileName), profiles)

		if selectedProfile == nil {
			fmt.Printf("\033[31mERROR\033[0m Profile '%s' not available.\n", profileName)
			os.Exit(1)
		}

		config, err := resources.GetCurrentConfig(ctx, client, ns)
		if err != nil {
			fmt.Println("\033[31mERROR\033[0m Odigos profile unavailable - no configuration found:", err)
			os.Exit(1)
		}
		config.ConfigVersion += 1

		// Check if the profile is already applied
		if slices.Contains(config.Profiles, selectedProfile.ProfileName) {
			fmt.Println("\033[34mINFO\033[0m Profile", profileName, "is already applied.")
			os.Exit(0)
		}

		// Add the profile to the current configuration
		config.Profiles = append(config.Profiles, selectedProfile.ProfileName)

		// Apply the updated configuration
		resourceManagers := resources.CreateResourceManagers(client, ns, currentTier, nil, config, currentOdigosVersion, string(installationmethod.K8sInstallationMethodOdigosCli))
		err = resources.ApplyResourceManagers(ctx, client, resourceManagers, "Updating")
		if err != nil {
			fmt.Println("Odigos profile add failed - unable to apply Odigos resources.")
			os.Exit(1)
		}
		err = resources.DeleteOldOdigosSystemObjects(ctx, client, ns, config)
		if err != nil {
			fmt.Println("Odigos profile add failed - unable to cleanup old Odigos resources.")
			os.Exit(1)
		}

		fmt.Printf("Profile '%s' successfully applied to the Odigos installation.\n", profileName)
	},
}

var removeProfileCmd = &cobra.Command{
	Use:   "remove <profile_name>",
	Short: "Remove a profile from the current Odigos installation",
	Long:  `Remove a profile by its name from the current Odigos installation.`,
	Args:  cobra.ExactArgs(1), // Ensure exactly one argument is passed (the profile name)
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		client := cmdcontext.KubeClientFromContextOrExit(ctx)

		ns, err := resources.GetOdigosNamespace(client, ctx)
		if resources.IsErrNoOdigosNamespaceFound(err) {
			fmt.Println("\033[31mERROR\033[0m No odigos installation found in the current cluster. Use \"odigos install\" to install odigos in the cluster or check that kubeconfig is pointing to the correct cluster.")
			os.Exit(1)
		} else if err != nil {
			fmt.Printf("\033[31mERROR\033[0m Failed to check if Odigos is already installed: %s\n", err)
			os.Exit(1)
		}

		currentOdigosVersion, err := getters.GetOdigosVersionInClusterFromConfigMap(ctx, client.Clientset, ns)
		if err != nil {
			fmt.Println("Odigos cloud login failed - unable to read the current Odigos version.")
			os.Exit(1)
		}

		profileName := args[0]
		currentTier, err := odigospro.GetCurrentOdigosTier(ctx, client, ns)
		if err != nil {
			fmt.Println("\033[31mERROR\033[0m Failed to retrieve current Odigos tier:", err)
			os.Exit(1)
		}

		config, err := resources.GetCurrentConfig(ctx, client, ns)
		if err != nil {
			fmt.Println("\033[31mERROR\033[0m Odigos profile unavailable - no configuration found:", err)
			os.Exit(1)
		}
		config.ConfigVersion += 1

		profileExists := false
		newProfiles := []common.ProfileName{}

		// Check if the profile is already applied
		for _, appliedProfile := range config.Profiles {
			if appliedProfile == common.ProfileName(profileName) {
				profileExists = true
			} else {
				newProfiles = append(newProfiles, appliedProfile)
			}
		}

		if !profileExists {
			fmt.Printf("\033[34mINFO\033[0m Profile '%s' is not applied.\n", profileName)
			os.Exit(0)
		}

		config.Profiles = newProfiles

		// Apply the updated configuration
		resourceManagers := resources.CreateResourceManagers(client, ns, currentTier, nil, config, currentOdigosVersion, string(installationmethod.K8sInstallationMethodOdigosCli))
		err = resources.ApplyResourceManagers(ctx, client, resourceManagers, "Updating")
		if err != nil {
			fmt.Println("Odigos profile remove failed - unable to apply Odigos resources.")
			os.Exit(1)
		}
		err = resources.DeleteOldOdigosSystemObjects(ctx, client, ns, config)
		if err != nil {
			fmt.Println("Odigos profile remove failed - unable to cleanup old Odigos resources.")
			os.Exit(1)
		}

		fmt.Printf("Profile '%s' successfully removed from Odigos installation.\n", profileName)
	},
}

func init() {
	rootCmd.AddCommand(profileCmd)

	profileCmd.Flags().BoolP("available", "a", false, "list all available profiles to use")

	profileCmd.AddCommand(addProfileCmd)
	profileCmd.AddCommand(removeProfileCmd)
}
