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
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/odigos-io/odigos/cli/cmd/resources"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/spf13/pflag"

	"github.com/spf13/cobra"
)

const (
	defaultPort   = 3000
	uiDownloadUrl = "https://github.com/odigos-io/odigos/releases/download/v%s/ui_%s_%s_%s.tar.gz"
)

// uiCmd represents the ui command
var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "Start the Odigos UI",
	Long:  `Start the Odigos UI. This will start a web server that will serve the UI`,
	Run: func(cmd *cobra.Command, args []string) {
		// get all flags as slice of strings
		var flags []string
		cmd.Flags().Visit(func(f *pflag.Flag) {
			flags = append(flags, fmt.Sprintf("--%s=%s", f.Name, f.Value))
		})

		ctx := cmd.Context()
		client, err := kube.CreateClient(cmd)
		if err != nil {
			kube.PrintClientErrorAndExit(err)
		}

		ns, err := resources.GetOdigosNamespace(client, ctx)
		if err != nil {
			if !resources.IsErrNoOdigosNamespaceFound(err) {
				fmt.Printf("\033[31mERROR\033[0m Cannot check if odigos is currently installed in your cluster: %s\n", err)
				ns = ""
			} else {
				fmt.Printf("\033[31mERROR\033[0m Odigos is not installed in your kubernetes cluster. Run 'odigos install' or switch your k8s context to use a different cluster \n")
				os.Exit(1)
			}
		}
		flags = append(flags, fmt.Sprintf("--namespace=%s", ns))

		var clusterVersion string
		if ns != "" {
			clusterVersion, err = GetOdigosVersionInCluster(ctx, client, ns)
			if err != nil {
				fmt.Println("not able to get odigos version from cluster")
				clusterVersion = ""
			}
		}
		fmt.Printf("Odigos version in cluster: %s\n", clusterVersion)
		binaryPath, binaryDir := GetOdigosUiBinaryPath()
		currentBinaryVersion, err := getCurrentBinaryVersion(binaryPath)
		if err != nil {
			fmt.Printf("Error getting current UI binary version: %v\n", err)
		}

		err = downloadOdigosUIVersionIfNeeded(runtime.GOARCH, runtime.GOOS, binaryDir, currentBinaryVersion, clusterVersion)
		if err != nil {
			fmt.Printf("Error downloading UI binary: %v\n", err)
			os.Exit(1)
		}

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

func getCurrentBinaryVersion(binaryPath string) (string, error) {
	_, err := os.Stat(binaryPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("Could not find current UI binary at %s\n", binaryPath)
			return "", nil
		} else {
			fmt.Printf("Error checking for UI binary: %v\n", err)
			return "", fmt.Errorf("error checking for UI binary: %v", err)
		}
	}

	cmd := exec.Command(binaryPath, "--version")
	outputBytes, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("unable to extract current odigos ui version: %v", err)
	}
	output := string(outputBytes)
	re := regexp.MustCompile(`v\d+\.\d+\.\d+`)
	return re.FindString(output), nil
}

func GetOdigosUiBinaryPath() (binaryPath, binaryDir string) {
	// Look for binary named odigos-ui in the same directory as the current binary
	// and execute it.
	// currentBinaryPath, err := os.Executable()
	currentBinaryPath, err := os.Executable()
	if err != nil {
		fmt.Printf("Error getting current binary path: %v\n", err)
		os.Exit(1)
	}

	if strings.HasPrefix(currentBinaryPath, "/ko-app") {
		// we are running in a docker container, no permission to write in the current directory
		// use /tmp as the binary directory
		binaryDir = "/tmp"
		binaryPath = filepath.Join(binaryDir, "odigos-ui")
		return
	}

	binaryDir = filepath.Dir(currentBinaryPath)
	binaryPath = filepath.Join(binaryDir, "odigos-ui")
	return
}

func downloadOdigosUIVersionIfNeeded(goarch string, goos string, currentDir string, currentBinaryVersion string, clusterVersion string) error {

	if clusterVersion != "" && clusterVersion == currentBinaryVersion {
		// common mainstream case
		// we already have the desired version of the ui. Nothing else to do
		return nil
	}

	// check if we can download latest version from github
	latestReleaseVersion, err := GetLatestReleaseVersion()
	if err != nil || latestReleaseVersion == "" {
		// no access to internet, if we have a binary, use it,
		// otherwise, we cannot proceed
		if currentBinaryVersion != "" {
			if clusterVersion != "" {
				fmt.Printf("\033[33mWARNING\033[0m No connection to github, will use current ui binary version %s to control your odigos installation of version %s.\n", currentBinaryVersion, clusterVersion)
			} else {
				fmt.Printf("\033[33mWARNING\033[0m No connection to github, will use current ui binary version %s\n", currentBinaryVersion)
			}
			return nil
		} else {
			fmt.Printf("Odigos ui binary not found and cannot download from github: %v\n", err)
			os.Exit(1)
		}
	}

	// the version does not match, attempt to download a new version
	// prefer the cluster version if known, or fallback to latest
	newUiVersion := clusterVersion
	if newUiVersion == "" {
		newUiVersion = latestReleaseVersion
	}

	return DoDownloadNewUiBinary(newUiVersion, currentDir, goarch, goos)
}

func DoDownloadNewUiBinary(version string, binaryDir string, goarch string, goos string) error {
	fmt.Printf("Downloading version %s of Odigos UI ...\n", version)
	// if the version starts with "v", remove it
	version = strings.TrimPrefix(version, "v")
	url := getDownloadUrl(goos, goarch, version)
	return downloadAndExtractTarGz(url, binaryDir)
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
	url := "https://api.github.com/repos/keyval-dev/odigos/releases/latest"

	timeout := time.Duration(3 * time.Second)
	client := http.Client{
		Timeout: timeout,
	}
	resp, err := client.Get(url)
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
