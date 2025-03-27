package cmd

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"

	"k8s.io/client-go/rest"

	"k8s.io/apimachinery/pkg/util/httpstream"
	"k8s.io/client-go/transport/spdy"

	"k8s.io/client-go/tools/portforward"

	"github.com/odigos-io/odigos/api/k8sconsts"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/odigos-io/odigos/cli/cmd/resources"
	cmdcontext "github.com/odigos-io/odigos/cli/pkg/cmd_context"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	corev1 "k8s.io/api/core/v1"

	"github.com/spf13/cobra"
)

const (
	defaultPort         = 3000
	centralBackendPort  = "8081"
	defaultLocalAddress = "localhost"
)

// uiCmd represents the ui command
var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "Start the Odigos UI",
	Long:  `Start the Odigos UI. This will start a web server that you can access in your browser and enables you to manage and configure Odigos.`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
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
		localAddress := cmd.Flag("address").Value.String()

		centralized, _ := cmd.Flags().GetBool("centralized")
		var uiPod *corev1.Pod
		if centralized {
			uiPod, err = findUIPodByLabel(client, ctx, ns, "app", "central-ui")
			go func() {
				backendPod, err := findUIPodByLabel(client, ctx, ns, "app", "central-backend")
				if err != nil {
					fmt.Printf("\033[31mERROR\033[0m Cannot find central-backend pod: %s\n", err)
					return
				}
				err = portForwardPod(ctx, client, backendPod, centralBackendPort, centralBackendPort, defaultLocalAddress, false)
				if err != nil {
					fmt.Printf("\033[31mERROR\033[0m Port-forwarding central backend failed: %s\n", err)
				}
			}()
		} else {
			uiPod, err = findUIPodByLabel(client, ctx, ns, "app", k8sconsts.UIAppLabelValue)
		}

		if err != nil {
			fmt.Printf("\033[31mERROR\033[0m Cannot find odigos-ui pod: %s\n", err)
			os.Exit(1)
		}

		if err := portForwardPod(ctx, client, uiPod, localPort, "", localAddress, true); err != nil {
			fmt.Printf("\033[31mERROR\033[0m Cannot start port-forward: %s\n", err)
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

func portForwardPod(
	ctx context.Context,
	client *kube.Client,
	pod *corev1.Pod,
	localPort string,
	remotePort string,
	localAddress string,
	printInfo bool,
) error {
	stopChannel := make(chan struct{}, 1)
	readyChannel := make(chan struct{})
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)
	defer signal.Stop(signals)

	go func() {
		select {
		case <-signals:
		case <-ctx.Done():
		}
		close(stopChannel)
	}()

	if printInfo {
		fmt.Printf("Odigos UI is available at: http://%s:%s\n\n", localAddress, localPort)
		fmt.Printf("Port-forwarding from %s/%s\n", pod.Namespace, pod.Name)
		fmt.Printf("Press Ctrl+C to stop\n")
	}

	req := client.CoreV1().RESTClient().
		Post().
		Resource("pods").
		Namespace(pod.Namespace).
		Name(pod.Name).
		SubResource("portforward")

	return forwardPorts("POST", req.URL(), client.Config, stopChannel, readyChannel, localPort, remotePort, localAddress)
}

func createDialer(method string, url *url.URL, cfg *rest.Config) (httpstream.Dialer, error) {
	transport, upgrader, err := spdy.RoundTripperFor(cfg)
	if err != nil {
		return nil, err
	}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, method, url)

	tunnelingDialer, err := portforward.NewSPDYOverWebsocketDialer(url, cfg)
	if err != nil {
		return nil, err
	}

	// First attempt tunneling (websocket) dialer, then fallback to spdy dialer.
	dialer = portforward.NewFallbackDialer(tunnelingDialer, dialer, httpstream.IsUpgradeFailure)
	return dialer, nil
}

func forwardPorts(
	method string,
	url *url.URL,
	cfg *rest.Config,
	stopCh chan struct{},
	readyCh chan struct{},
	localPort string,
	remotePort string,
	localAddress string,
) error {
	dialer, err := createDialer(method, url, cfg)
	if err != nil {
		return err
	}

	if remotePort == "" {
		remotePort = fmt.Sprintf("%d", defaultPort)
	}

	port := fmt.Sprintf("%s:%s", localPort, remotePort)
	fw, err := portforward.NewOnAddresses(
		dialer,
		[]string{localAddress},
		[]string{port},
		stopCh,
		readyCh,
		nil,
		os.Stderr,
	)
	if err != nil {
		return err
	}
	return fw.ForwardPorts()
}

func findUIPodByLabel(client *kube.Client, ctx context.Context, ns string, key string, value string) (*corev1.Pod, error) {
	pods, err := client.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", key, value),
	})

	if err != nil {
		return nil, err
	}

	if len(pods.Items) != 1 {
		return nil, fmt.Errorf("expected to get 1 pod got %d", len(pods.Items))
	}

	pod := &pods.Items[0]
	if pod.Status.Phase != corev1.PodRunning {
		return nil, fmt.Errorf("%s pod is not running", value)
	}

	return pod, nil
}
func init() {
	rootCmd.AddCommand(uiCmd)
	uiCmd.Flags().Int("port", defaultPort, "Port to listen on")
	uiCmd.Flags().String("address", "localhost", "Address to serve the UI on")
	uiCmd.Flags().Bool("centralized", false, "Use centralized Odigos UI (for centralized mode)")
}
