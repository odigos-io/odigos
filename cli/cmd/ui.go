package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/odigos-io/odigos/cli/cmd/resources"
	cmdcontext "github.com/odigos-io/odigos/cli/pkg/cmd_context"
	"github.com/odigos-io/odigos/cli/pkg/kube"

	"github.com/spf13/cobra"
)

const (
	defaultPort = 3000
)

// uiCmd represents the ui command
var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "Start the Odigos UI",
	Long:  `Start the Odigos UI. This will start a web server that you can access in your browser and enables you to manage and configure Odigos.`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithCancel(cmd.Context())
		defer cancel()

		var wg sync.WaitGroup
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt)

		client := cmdcontext.KubeClientFromContextOrExit(ctx)

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

		localPort := cmd.Flag("port").Value.String()
		remotePort := cmd.Flag("remote-port").Value.String()
		localAddress := cmd.Flag("address").Value.String()

		fmt.Printf("Starting resilient port-forward to Odigos UI...\n")
		fmt.Printf("Press Ctrl+C to stop\n")

		// Start resilient port forwarding in a goroutine
		wg.Add(1)
		go func() {
			defer wg.Done()
			startResilientUIPortForward(ctx, client, localPort, remotePort, localAddress, ns)
		}()

		// Wait for interrupt signal
		<-sigCh
		fmt.Println("\nReceived interrupt. Stopping UI port forwarding...")
		cancel()
		wg.Wait()
	},
	Example: `
# Start the Odigos UI on http://localhost:3000
odigos ui

# Start the Odigos UI on specific port if 3000 is already in use
odigos ui --port 3456

# Start the Odigos UI and have it manage and configure a specific cluster
odigos ui --kubeconfig <path-to-kubeconfig>
`,
}

func init() {
	rootCmd.AddCommand(uiCmd)
	uiCmd.Flags().Int("port", defaultPort, "Port to listen on")
	uiCmd.Flags().Int("remote-port", defaultPort, "Port to forward to the remote UI")
	uiCmd.Flags().String("address", "localhost", "Address to serve the UI on")
}

func startResilientUIPortForward(ctx context.Context, client *kube.Client, localPort, remotePort, localAddress, namespace string) {
	retryDelay := time.Second * 3
	maxRetryDelay := time.Second * 30
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Find the current UI pod
		uiPod, err := kube.FindOdigosUIPod(client, ctx, namespace)
		if err != nil {
			fmt.Printf("\033[33mWARN\033[0m UI: Cannot find pod (will retry in %v): %v\n", retryDelay, err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(retryDelay):
				// Exponential backoff with max limit
				retryDelay = time.Duration(float64(retryDelay) * 1.5)
				if retryDelay > maxRetryDelay {
					retryDelay = maxRetryDelay
				}
				continue
			}
		}

		// Reset retry delay on successful pod discovery
		retryDelay = time.Second * 3
		fmt.Printf("\033[32mINFO\033[0m UI: Starting port-forward to pod %s\n", uiPod.Name)
		fmt.Printf("Odigos UI is available at: http://%s:%s\n", localAddress, localPort)
		fmt.Printf("Port-forwarding from %s/%s\n", uiPod.Namespace, uiPod.Name)

		// Create port forward
		fw, err := kube.PortForwardWithContext(ctx, uiPod, client, localPort, remotePort, localAddress)
		if err != nil {
			fmt.Printf("\033[33mWARN\033[0m UI: Failed to create port-forward (will retry): %v\n", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(retryDelay):
				continue
			}
		}

		// Start port forwarding (this blocks until connection is lost)
		err = fw.ForwardPorts()
		if err != nil && ctx.Err() == nil {
			// Only log as warning if context wasn't cancelled (i.e., not a clean shutdown)
			fmt.Printf("\033[33mWARN\033[0m UI: Port-forward connection lost (will retry): %v\n", err)
			// Brief pause before retrying to avoid tight retry loops
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second * 2):
				continue
			}
		}

		// If we reach here, either context was cancelled or we had a clean exit
		return
	}
}
