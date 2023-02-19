package instrumentation

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	podresourcesapi "k8s.io/kubelet/pkg/apis/podresources/v1alpha1"
	"time"
)

var (
	socketDir  = "/var/lib/kubelet/pod-resources"
	socketPath = "unix://" + socketDir + "/kubelet.sock"

	connectionTimeout = 10 * time.Second
)

func getPodResources() error {
	c, cleanup, err := connectToServer(socketPath)
	if err != nil {
		return err
	}
	defer cleanup()

	pods, err := ListPods(c)
	if err != nil {
		return err
	}

	// Print pods
	for _, pod := range pods.GetPodResources() {
		fmt.Printf("Pod: %s\n", pod.GetName())
		for _, container := range pod.GetContainers() {
			fmt.Printf("  Container: %s\n", container.GetName())
			for _, dev := range container.GetDevices() {
				for _, id := range dev.GetDeviceIds() {
					fmt.Printf("    Device ID: %s\n", id)
				}
			}
		}
	}

	return nil
}

func connectToServer(socket string) (*grpc.ClientConn, func(), error) {
	ctx, cancel := context.WithTimeout(context.Background(), connectionTimeout)
	defer cancel()

	conn, err := grpc.DialContext(ctx, socket, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())

	if err != nil {
		return nil, func() {}, fmt.Errorf("failure connecting to %s: %v", socket, err)
	}

	return conn, func() { conn.Close() }, nil
}

func ListPods(conn *grpc.ClientConn) (*podresourcesapi.ListPodResourcesResponse, error) {
	client := podresourcesapi.NewPodResourcesListerClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), connectionTimeout)
	defer cancel()

	resp, err := client.List(ctx, &podresourcesapi.ListPodResourcesRequest{})
	if err != nil {
		return nil, fmt.Errorf("failure getting pod resources %v", err)
	}

	return resp, nil
}
