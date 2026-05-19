package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func mustGetBoolFlag(cmd *cobra.Command, name string) bool {
	value, err := cmd.Flags().GetBool(name)
	if err != nil {
		fmt.Printf("\033[31mERROR\033[0m Failed to read %s flag: %s\n", name, err)
		os.Exit(1)
	}
	return value
}
