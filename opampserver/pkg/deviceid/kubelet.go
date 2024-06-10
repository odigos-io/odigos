package deviceid

import (
	"context"
	"fmt"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	podresourcesapi "k8s.io/kubelet/pkg/apis/podresources/v1alpha1"
)

var (
	socketDir  = "/var/lib/kubelet/pod-resources"
	socketPath = "unix://" + socketDir + "/kubelet.sock"

	connectionTimeout = 10 * time.Second
)

type ContainerDetails struct {
	PodName         string
	PodNamespace    string
	ContainersInPod int
	ContainerName   string
}

type KubeletClient struct {
	conn *grpc.ClientConn
}

func NewKubeletClient() (*KubeletClient, error) {
	conn, err := connectToKubelet(socketPath)
	if err != nil {
		return nil, err
	}

	return &KubeletClient{
		conn: conn,
	}, nil
}

func (c *KubeletClient) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

// this function will query the current state of the kubelet device plugins
// and map each relevant device id to the container details which is using it.
// givin a device id, we can know which pod and container it is allocated to and work our way
// to complete the info for workload name and kind.
//
// unfortunately, I did not find a way to get just one device id from the kubelet, we need
// to list and iterate over all the devices to get a current snapshot of allocations on this node
// and then look for the ones we are interested in.
func (c *KubeletClient) DeviceIdsToContainerDetails() (map[string]*ContainerDetails, error) {
	pods, err := c.listPods()
	if err != nil {
		return nil, err
	}

	allocations := make(map[string]*ContainerDetails)
	for _, pod := range pods.GetPodResources() {
		for _, container := range pod.Containers {
			for _, device := range container.Devices {
				for _, id := range device.DeviceIds {
					if strings.Contains(device.GetResourceName(), "odigos.io") {
						allocations[id] = &ContainerDetails{
							PodName:         pod.Name,
							PodNamespace:    pod.Namespace,
							ContainerName:   container.Name,
							ContainersInPod: len(pod.Containers),
						}
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

func (c *KubeletClient) listPods() (*podresourcesapi.ListPodResourcesResponse, error) {
	client := podresourcesapi.NewPodResourcesListerClient(c.conn)

	ctx, cancel := context.WithTimeout(context.Background(), connectionTimeout)
	defer cancel()

	resp, err := client.List(ctx, &podresourcesapi.ListPodResourcesRequest{})
	if err != nil {
		return nil, fmt.Errorf("failure getting pod resources %v", err)
	}

	return resp, nil
}
