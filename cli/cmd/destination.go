/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/odigos-io/odigos/cli/cmd/resources"
	"github.com/odigos-io/odigos/cli/pkg/kube"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/spf13/cobra"
)

// destinationCmd represents the destination command
var destinationCmd = &cobra.Command{
	Use:   "destination",
	Short: "Perform destination operations",
	Long: `Perform add, remove, list, and other operations on destinations.

A destination is an observability backend that is able to store OTLP data.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("destination called")
	},
}

type fieldConfig struct {
	name          string
	yamlFieldName string
}

type destinationType struct {
	displayName    string
	requiredFlags  []fieldConfig
	optionalFlags  []fieldConfig
	validateConfig func(map[string]string) error
}

var destinationTypes = map[string]destinationType{
	"clickhouse": {
		displayName: "ClickHouse",
		requiredFlags: []fieldConfig{
			{name: "endpoint", yamlFieldName: "CLICKHOUSE_ENDPOINT"},
		},
	},
	// Add more destination types here
}

func validateSignals(signals string) error {
	validSignals := map[string]bool{
		"traces":  true,
		"metrics": true,
		"logs":    true,
	}

	signalsList := strings.Split(strings.ToLower(signals), ",")
	for _, signal := range signalsList {
		if !validSignals[signal] {
			return fmt.Errorf("invalid signal: %s. Valid signals are: traces, metrics, logs", signal)
		}
	}
	return nil
}

func validateDestinationType(destType string) (destinationType, error) {
	destType = strings.ToLower(destType)
	if dt, exists := destinationTypes[destType]; exists {
		return dt, nil
	}
	return destinationType{}, fmt.Errorf("invalid destination type: %s", destType)
}

func executeAuthCommand(authCommand string, clusterName string) {
	if authCommand == "" {
		return
	}

	authCommand = strings.Replace(authCommand, "@clusterName", clusterName, -1)
	fmt.Printf("> Executing auth command: %s\n", authCommand)
	cmd := exec.Command("sh", "-c", authCommand)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error executing auth command: %v\n", err)
		fmt.Printf("Command output:\n%s\n", string(output))
		return
	}
	fmt.Printf("Auth command output:\n%s\n", string(output))
	return
}

func addDestinationToCluster(cmd *cobra.Command, name string, cluster string, authCommand string) error {
	if cluster == "" {
		fmt.Printf("Adding destination \033[1m%s\033[0m to current cluster\n", name)
	} else {
		fmt.Printf("Adding destination \033[1m%s\033[0m to cluster \033[1m%s\033[0m\n", name, cluster)
	}

	executeAuthCommand(authCommand, cluster)

	client := kube.GetCLIClientOrExit(cmd)
	_, err := resources.GetOdigosNamespace(client, cmd.Context())
	if err != nil {
		return nil
	}

	// TODO: Actual destination addition logic here
	return nil
}

var addDestinationCmd = &cobra.Command{
	Use:   "add [name]",
	Short: "Add a destination",
	Long:  `Add a destination to the list of destinations.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		destType, _ := cmd.Flags().GetString("type")
		signals, _ := cmd.Flags().GetString("signals")
		authCommand, _ := cmd.Flags().GetString("auth-command")
		clusterList, _ := cmd.Flags().GetString("cluster-list")

		if err := validateSignals(signals); err != nil {
			fmt.Println(err)
			return
		}

		dt, err := validateDestinationType(destType)
		if err != nil {
			fmt.Println(err)
			return
		}

		// Collect all flag values
		for _, field := range dt.requiredFlags {
			value, _ := cmd.Flags().GetString(field.name)
			if value == "" {
				fmt.Printf("Flag --%s is required for destination type %s\n", field.name, destType)
				return
			}
		}

		var clusters []string
		if clusterList != "" {
			file, err := os.Open(clusterList)
			if err != nil {
				fmt.Printf("Error opening cluster list file: %v\n", err)
				return
			}
			defer file.Close()

			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				clusters = append(clusters, scanner.Text())
			}
		}

		if len(clusters) == 0 {
			if err := addDestinationToCluster(cmd, args[0], "", authCommand); err != nil {
				return
			}
		} else {
			for _, cluster := range clusters {
				if err := addDestinationToCluster(cmd, args[0], cluster, authCommand); err != nil {
					return
				}
			}
		}
	},
}

func generateDestination(cmd *cobra.Command, destType *destinationType) *odigosv1.Destination {
	return &odigosv1.Destination{
		ObjectMeta: metav1.ObjectMeta{
			Name:                       "",
			GenerateName:               "",
			Namespace:                  "",
			DeletionGracePeriodSeconds: nil,
			Labels:                     nil,
			Annotations:                nil,
			OwnerReferences:            nil,
			Finalizers:                 nil,
			ManagedFields:              nil,
		},
		Spec:   odigosv1.DestinationSpec{},
		Status: odigosv1.DestinationStatus{},
	}
}

func init() {
	rootCmd.AddCommand(destinationCmd)
	destinationCmd.AddCommand(addDestinationCmd)

	// Add required flags to addDestinationCmd
	addDestinationCmd.Flags().String("type", "", "Type of the destination")
	addDestinationCmd.Flags().String("signals", "", "Signals to collect (comma-separated: traces,metrics,logs)")
	addDestinationCmd.Flags().String("endpoint", "", "Endpoint URL (required for some destination types)")
	addDestinationCmd.Flags().String("auth-command", "", "Command to execute to get authentication token. Use @clusterName to reference the current cluster name")
	addDestinationCmd.Flags().String("cluster-list", "", "Path to file containing cluster names (one per line) for batch destination addition")

	// Add global optional flags
	addDestinationCmd.MarkFlagRequired("type")
	addDestinationCmd.MarkFlagRequired("signals")
}
