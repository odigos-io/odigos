package kube

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/httpstream"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

func PortForwardWithContext(ctx context.Context, pod *corev1.Pod, client *Client, localPort, uiPort, localAddress string) (*portforward.PortForwarder, error) {
	stopChannel := make(chan struct{}, 1)
	readyChannel := make(chan struct{})

	go func() {
		<-ctx.Done()
		close(stopChannel)
	}()

	req := client.CoreV1().RESTClient().
		Post().
		Resource("pods").
		Namespace(pod.Namespace).
		Name(pod.Name).
		SubResource("portforward")

	dialer, err := createDialer("POST", req.URL(), client.Config)
	if err != nil {
		return nil, err
	}

	ports := fmt.Sprintf("%s:%s", localPort, uiPort)
	fw, err := portforward.NewOnAddresses(dialer, []string{localAddress}, []string{ports}, stopChannel, readyChannel, os.Stdout, os.Stderr)
	if err != nil {
		return nil, err
	}

	return fw, err
}

// ResilientPortForwardConfig contains configuration for resilient port forwarding
type ResilientPortForwardConfig struct {
	WaitGroup    *sync.WaitGroup // Optional: if provided, will be used for coordination
	Client       *Client
	LocalPort    string
	RemotePort   string
	LocalAddress string
	Namespace    string
	Name         string // Name for logging (e.g., "UI", "Backend")
	AppLabel     string // App label to find the pod (e.g., "odigos-ui", "odigos-central-backend")
}

// StartResilientPortForward starts a resilient port forwarding session that automatically
// recovers from pod restarts by re-discovering pods and re-establishing connections
func StartResilientPortForward(ctx context.Context, config ResilientPortForwardConfig) {
	if config.WaitGroup != nil {
		config.WaitGroup.Add(1)
	}

	go func() {
		if config.WaitGroup != nil {
			defer config.WaitGroup.Done()
		}

		retryDelay := time.Second * 3
		maxRetryDelay := time.Second * 30
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			// Find the current pod
			pod, err := FindPodWithAppLabel(config.Client, ctx, config.Namespace, config.AppLabel)
			if err != nil {
				fmt.Printf("\033[33mWARN\033[0m %s: Cannot find pod (will retry in %v): %v\n", config.Name, retryDelay, err)
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
			fmt.Printf("\033[32mINFO\033[0m %s: Starting port-forward to pod %s\n", config.Name, pod.Name)

			// Create port forward
			fw, err := PortForwardWithContext(ctx, pod, config.Client, config.LocalPort, config.RemotePort, config.LocalAddress)
			if err != nil {
				fmt.Printf("\033[33mWARN\033[0m %s: Failed to create port-forward (will retry): %v\n", config.Name, err)
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
				fmt.Printf("\033[33mWARN\033[0m %s: Port-forward connection lost (will retry): %v\n", config.Name, err)
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
	}()
}

// FindPodWithAppLabel finds a running pod by app label - generic version for any app
func FindPodWithAppLabel(client *Client, ctx context.Context, namespace, appLabel string) (*corev1.Pod, error) {
	pods, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s", appLabel),
	})
	if err != nil {
		return nil, err
	}

	runningPods := []corev1.Pod{}
	for _, pod := range pods.Items {
		if pod.Status.Phase == corev1.PodRunning {
			runningPods = append(runningPods, pod)
		}
	}
	if len(runningPods) == 0 {
		return nil, fmt.Errorf("%s pod is not running", appLabel)
	}
	if len(runningPods) > 1 {
		return nil, fmt.Errorf("expected to get 1 running (%s) pod, got %d", appLabel, len(runningPods))
	}

	pod := &runningPods[0]
	return pod, nil
}

func createDialer(method string, url *url.URL, cfg *rest.Config) (httpstream.Dialer, error) {
	transport, upgrader, err := spdy.RoundTripperFor(cfg)
	if err != nil {
		return nil, err
	}
	spdyDialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, method, url)

	tunnelDialer, err := portforward.NewSPDYOverWebsocketDialer(url, cfg)
	if err != nil {
		return nil, err
	}

	return portforward.NewFallbackDialer(tunnelDialer, spdyDialer, httpstream.IsUpgradeFailure), nil
}
