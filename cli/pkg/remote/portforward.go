package remote

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"k8s.io/apimachinery/pkg/util/httpstream"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"

	"github.com/odigos-io/odigos/cli/cmd/resources"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	DefaultLocalPort = "3333"
	DefaultAddress   = "localhost"
	UiPort           = 3000
)

func NewUIClient(client *kube.Client, ctx context.Context) (*UIClientViaPortForward, error) {
	odigosNs, err := resources.GetOdigosNamespace(client, ctx)
	if err != nil {
		return nil, err
	}

	uiPod, err := findOdigosUIPod(client, ctx, odigosNs)
	if err != nil {
		return nil, err
	}

	req := client.CoreV1().RESTClient().
		Post().
		Resource("pods").
		Namespace(uiPod.Namespace).
		Name(uiPod.Name).
		SubResource("portforward")

	dialer, err := createDialer("POST", req.URL(), client.Config)
	if err != nil {
		return nil, err
	}

	port := fmt.Sprintf("%s:%d", DefaultLocalPort, UiPort)
	stopChannel := make(chan struct{}, 1)
	readyChannel := make(chan struct{})

	fw, err := portforward.NewOnAddresses(dialer,
		[]string{DefaultAddress},
		[]string{port}, stopChannel, readyChannel, nil, os.Stderr)

	if err != nil {
		return nil, err
	}

	return &UIClientViaPortForward{
		pf:       fw,
		stopCh:   stopChannel,
		readyCh:  readyChannel,
		isClosed: false,
	}, nil
}

type UIClientViaPortForward struct {
	pf       *portforward.PortForwarder
	readyCh  chan struct{}
	stopCh   chan struct{}
	isClosed bool
}

func (u *UIClientViaPortForward) Start() error {
	return u.pf.ForwardPorts()
}

func (u *UIClientViaPortForward) Ready() <-chan struct{} {
	return u.readyCh
}

func (u *UIClientViaPortForward) Close() error {
	// Check if channel is closed
	if !u.isClosed {
		fmt.Println("Closing port-forward to UI pod")
		close(u.stopCh)
		u.isClosed = true
	}
	return nil
}

func findOdigosUIPod(client *kube.Client, ctx context.Context, ns string) (*corev1.Pod, error) {
	pods, err := client.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s", resources.UIAppLabelValue),
	})

	if err != nil {
		return nil, err
	}

	if len(pods.Items) != 1 {
		return nil, fmt.Errorf("expected to get 1 pod got %d", len(pods.Items))
	}

	pod := &pods.Items[0]
	if pod.Status.Phase != corev1.PodRunning {
		return nil, fmt.Errorf("odigos-ui pod is not running")
	}

	return &pods.Items[0], nil
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
