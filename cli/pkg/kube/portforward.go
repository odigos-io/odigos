package kube

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/httpstream"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

func PortForwardWithContext(ctx context.Context, pod *corev1.Pod, client *Client, localPort, localAddress string) error {
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

	return forwardPorts("POST", req.URL(), client.Config, stopChannel, readyChannel, localPort, localAddress)
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

func forwardPorts(method string, url *url.URL, cfg *rest.Config, stopCh chan struct{}, readyCh chan struct{}, localPort string, localAddress string) error {
	dialer, err := createDialer(method, url, cfg)
	if err != nil {
		return err
	}

	ports := []string{fmt.Sprintf("%s:%s", localPort, localPort)}
	fw, err := portforward.NewOnAddresses(dialer, []string{localAddress}, ports, stopCh, readyCh, os.Stdout, os.Stderr)
	if err != nil {
		return err
	}

	return fw.ForwardPorts()
}
