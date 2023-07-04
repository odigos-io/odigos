/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/spf13/pflag"

	"github.com/spf13/cobra"
)

const (
	defaultPort = 3000
)

// uiCmd represents the ui command
var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "Start the Odigos UI",
	Long:  `Start the Odigos UI. This will start a web server that will serve the UI`,
	Run: func(cmd *cobra.Command, args []string) {
		// Look for binary named odigos-ui in the same directory as the current binary
		// and execute it.
		currentBinaryPath, err := os.Executable()
		if err != nil {
			fmt.Printf("Error getting current binary path: %v\n", err)
			os.Exit(1)
		}

		binaryPath := filepath.Join(filepath.Dir(currentBinaryPath), "odigos-ui")
		if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
			fmt.Printf("Could not find UI binary, downloading latest release\n")
			err = downloadLatestUIVersion(runtime.GOARCH, runtime.GOOS)
			if err != nil {
				fmt.Printf("Error downloading latest UI version: %v\n", err)
				os.Exit(1)
			}
		}

		// get all flags as slice of strings
		var flags []string
		cmd.Flags().Visit(func(f *pflag.Flag) {
			flags = append(flags, fmt.Sprintf("--%s=%s", f.Name, f.Value))
		})

		// execute UI binary with all flags and stream output
		process := exec.Command(binaryPath, flags...)
		process.Stdout = os.Stdout
		process.Stderr = os.Stderr
		err = process.Run()
		if err != nil {
			fmt.Printf("Error starting UI: %v\n", err)
			os.Exit(1)
		}
	},
}

func downloadLatestUIVersion(arch string, os string) error {
	latestRelease, err := GetLatestReleaseVersion()
	if err != nil {
		return err
	}

	fmt.Printf("Downloading version %s of Odigos UI ...\n", latestRelease)
	return nil
}

type Release struct {
	TagName string `json:"tag_name"`
}

func GetLatestReleaseVersion() (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/keyval-dev/odigos/releases/latest")

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch latest release: %s", resp.Status)
	}

	var release Release
	err = json.NewDecoder(resp.Body).Decode(&release)
	if err != nil {
		return "", err
	}

	return release.TagName, nil
}

func init() {
	rootCmd.AddCommand(uiCmd)
	uiCmd.Flags().String("address", "localhost", "Address to listen on")
	uiCmd.Flags().Int("port", defaultPort, "Port to listen on")
	uiCmd.Flags().Bool("debug", false, "Enable debug mode")
}
