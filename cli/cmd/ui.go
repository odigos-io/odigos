package cmd

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/pflag"

	"github.com/keyval-dev/odigos/cli/cmd/resources"
	"github.com/keyval-dev/odigos/cli/pkg/kube"
	"github.com/spf13/cobra"
)

const (
	defaultPort   = 3000
	uiDownloadUrl = "https://github.com/keyval-dev/odigos/releases/download/v%s/ui_%s_%s_%s.tar.gz"
)

// uiCmd represents the ui command
var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "Start the Odigos UI",
	Long:  `Start the Odigos UI. This will start a web server that will serve the UI`,
	Run: func(cmd *cobra.Command, args []string) {

		if checkOdigosInstallation(cmd) {
			// Look for binary named odigos-ui in the same directory as the current binary
			// and execute it.
			currentBinaryPath, err := os.Executable()
			if err != nil {
				fmt.Printf("Error getting current binary path: %v\n", err)
				os.Exit(1)
			}

			currentDir := filepath.Dir(currentBinaryPath)
			binaryPath := filepath.Join(currentDir, "odigos-ui")
			if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
				fmt.Printf("Could not find UI binary, downloading latest release\n")
				err = downloadLatestUIVersion(runtime.GOARCH, runtime.GOOS, currentDir)
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
		} else {
			fmt.Printf("\033[31mERROR\033[0m Unable to find Odigos in kubernetes cluster.\n")
			os.Exit(1)
		}

	},
}

func checkOdigosInstallation(cmd *cobra.Command) bool {
	client, err := kube.CreateClient(cmd)
	if err != nil {
		kube.PrintClientErrorAndExit(err)
		return false
	}
	ctx := cmd.Context()

	// check if odigos is installed
	_, err = resources.GetOdigosNamespace(client, ctx)
	if err == nil {
		return true
	} else if !resources.IsErrNoOdigosNamespaceFound(err) {
		fmt.Printf("\033[31mERROR\033[0m Cannot install/start UI. Failed to check if Odigos is already installed: %s\n", err)
		return false
	}
	return true
}

func downloadLatestUIVersion(arch string, os string, currentDir string) error {
	latestRelease, err := GetLatestReleaseVersion()
	if err != nil {
		return err
	}

	fmt.Printf("Downloading version %s of Odigos UI ...\n", latestRelease)
	url := getDownloadUrl(os, arch, latestRelease)
	return downloadAndExtractTarGz(url, currentDir)
}

func downloadAndExtractTarGz(url string, dir string) error {
	// Step 1: Download the tar.gz file
	response, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download file: %v", err)
	}
	defer response.Body.Close()

	// Step 2: Create a new gzip reader
	gzipReader, err := gzip.NewReader(response.Body)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %v", err)
	}
	defer gzipReader.Close()

	// Step 3: Create a new tar reader
	tarReader := tar.NewReader(gzipReader)

	// Step 4: Extract files one by one
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break // Reached the end of the tar archive
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %v", err)
		}

		// Step 5: Create the file or directory
		targetPath := filepath.Join(dir, header.Name)
		if header.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return fmt.Errorf("failed to create directory: %v", err)
			}
			continue
		}

		file, err := os.Create(targetPath)
		if err != nil {
			return fmt.Errorf("failed to create file: %v", err)
		}

		// Step 6: Copy file contents from tar to the target file
		if _, err := io.Copy(file, tarReader); err != nil {
			file.Close()
			return fmt.Errorf("failed to extract file: %v", err)
		}

		file.Close()
	}

	return os.Chmod(filepath.Join(dir, "odigos-ui"), 0755)
}

func getDownloadUrl(os string, arch string, version string) string {
	return fmt.Sprintf(uiDownloadUrl, version, version, os, arch)
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

	ver, _ := strings.CutPrefix(release.TagName, "v")
	return ver, nil
}

func init() {
	rootCmd.AddCommand(uiCmd)
	uiCmd.Flags().String("address", "localhost", "address to listen on")
	uiCmd.Flags().Int("port", defaultPort, "port to listen on")
	uiCmd.Flags().Bool("debug", false, "enable debug mode")
}
