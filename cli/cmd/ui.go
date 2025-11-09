package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"

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

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt)
		go func() {
			<-sigCh
			fmt.Println("\nReceived interrupt. Stopping UI port forwarding...")
			cancel()
		}()

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

		uiPod, err := kube.FindOdigosUIPod(client, ctx, ns)
		if err != nil {
			fmt.Printf("\033[31mERROR\033[0m Cannot find odigos-ui pod: %s\n", err)
			os.Exit(1)
		}
		fmt.Printf("Odigos UI is available at: http://%s:%s\n", localAddress, localPort)
		fmt.Printf("Port-forwarding from %s/%s\n", uiPod.Namespace, uiPod.Name)
		fmt.Printf("Press Ctrl+C to stop\n")

		fw, err := kube.PortForwardWithContext(ctx, uiPod, client, localPort, remotePort, localAddress)
		if err != nil {
			fmt.Printf("\033[31mERROR\033[0m Cannot start port-forward: %s\n", err)
			os.Exit(1)
		}
		err = fw.ForwardPorts()
		if err != nil {
			fmt.Printf("\033[31mERROR\033[0m Cannot forward ports: %s\n", err)
			os.Exit(1)
		}

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
