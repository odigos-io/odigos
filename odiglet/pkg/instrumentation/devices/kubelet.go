package devices

import (
	"context"
	"errors"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	podresourcesapi "k8s.io/kubelet/pkg/apis/podresources/v1alpha1"
	"strings"
	"time"
)

var (
	socketDir  = "/var/lib/kubelet/pod-resources"
	socketPath = "unix://" + socketDir + "/kubelet.sock"

	connectionTimeout = 10 * time.Second
	ErrNoDeviceFound  = errors.New("no device found")
)

type kubeletClient struct {
	conn *grpc.ClientConn
}

func NewKubeletClient() (*kubeletClient, error) {
	conn, err := connectToKubelet(socketPath)
	if err != nil {
		return nil, err
	}

	return &kubeletClient{
		conn: conn,
	}, nil
}

func (c *kubeletClient) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

func (c *kubeletClient) GetAllocations() (map[PodDetails]string, error) {
	pods, err := c.listPods()
	if err != nil {
		return nil, err
	}

	allocations := make(map[PodDetails]string)
	for _, pod := range pods.GetPodResources() {
		podDetails := NewPodDetails(pod.Name, pod.Namespace)
		for _, container := range pod.Containers {
			for _, device := range container.Devices {
				for _, id := range device.DeviceIds {
					if strings.Contains(device.GetResourceName(), "odigos.io") {
						allocations[podDetails] = id
					}
				}
			}
		}
	}

	return allocations, nil
}

func connectToKubelet(socket string) (*grpc.ClientConn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), connectionTimeout)
	defer cancel()

	conn, err := grpc.DialContext(ctx, socket, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())

	if err != nil {
		return nil, fmt.Errorf("failure connecting to %s: %v", socket, err)
	}

	return conn, nil
}

func (c *kubeletClient) listPods() (*podresourcesapi.ListPodResourcesResponse, error) {
	client := podresourcesapi.NewPodResourcesListerClient(c.conn)

	ctx, cancel := context.WithTimeout(context.Background(), connectionTimeout)
	defer cancel()

	resp, err := client.List(ctx, &podresourcesapi.ListPodResourcesRequest{})
	if err != nil {
		return nil, fmt.Errorf("failure getting pod resources %v", err)
	}

	return resp, nil
}
