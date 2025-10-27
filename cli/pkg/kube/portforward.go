package kube

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/odigos-io/odigos/api/k8sconsts"

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

func FindOdigosUIPod(client *Client, ctx context.Context, ns string) (*corev1.Pod, error) {
	pods, err := client.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s", k8sconsts.UIAppLabelValue),
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
		return nil, fmt.Errorf("%s pod is not running", k8sconsts.UIAppLabelValue)
	}
	if len(runningPods) > 1 {
		return nil, fmt.Errorf("expected to get 1 running (%s) pod, got %d", k8sconsts.UIAppLabelValue, len(runningPods))
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
