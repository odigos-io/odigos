/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	_ "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	"github.com/keyval-dev/odigos/cli/cmd/observability/backend"
	"github.com/spf13/cobra"
)

// observabilityCmd represents the observability command
var observabilityCmd = &cobra.Command{
	Use:   "observability",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := isValidBackend(backendFlag); err != nil {
			return err
		}

		be := backend.Get(backendFlag)
		if err := be.ValidateFlags(cmd); err != nil {
			return err
		}

		fmt.Println("observability called")
		return nil
	},
}

var (
	backendFlag string
	apiKeyFlag  string
	urlFlag     string
)

func isValidBackend(name string) error {
	avail := backend.GetAvailableBackends()
	for _, s := range avail {
		if name == s {
			return nil
		}
	}

	return fmt.Errorf("invalid backend %s, choose from %+v", name, avail)
}

func init() {
	skillCmd.AddCommand(observabilityCmd)
	observabilityCmd.Flags().StringVar(&backendFlag, "backend", "", "Backend for observability data")
	//observabilityCmd.MarkFlagRequired("backend")
	observabilityCmd.Flags().StringVarP(&urlFlag, "url", "u", "", "URL of the backend for observability data")
	//observabilityCmd.MarkFlagRequired("url")
	observabilityCmd.Flags().StringVar(&apiKeyFlag, "api-key", "", "API key for the selected backend")
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// observabilityCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// observabilityCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
