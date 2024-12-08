package deviceid

import (
	"context"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

type K8sResourceAttributes struct {
	Namespace     string
	WorkloadKind  string
	WorkloadName  string
	PodName       string
	ContainerName string
}

type DeviceIdCache struct {
	logger          logr.Logger
	podInfoResolver *K8sPodInfoResolver
	kubeletClient   *KubeletClient

	// this map holds the last snapshot of the device ids and their container details
	// it is always updated as a whole with a fresh list of all the pods containers and devices on the node
	resolvedDeviceIds map[string]*ContainerDetails
}

func NewDeviceIdCache(logger logr.Logger, kubeclient *kubernetes.Clientset) (*DeviceIdCache, error) {
	kubeletClient, err := NewKubeletClient()
	if err != nil {
		return nil, err
	}
	return &DeviceIdCache{
		logger:          logger,
		podInfoResolver: NewK8sPodInfoResolver(logger, kubeclient),
		kubeletClient:   kubeletClient,
	}, nil
}

func (d *DeviceIdCache) GetAttributesFromDevice(ctx context.Context, deviceId string) (*K8sResourceAttributes, *corev1.Pod, error) {

	// try to get container details from cache
	containerDetails := d.resolvedDeviceIds[deviceId]
	if containerDetails == nil {
		// not found in cache, update the container details cache from kubelet
		newResolvedDeviceIds, err := d.kubeletClient.DeviceIdsToContainerDetails()
		if err != nil {
			return nil, nil, err
		}
		d.resolvedDeviceIds = newResolvedDeviceIds
		containerDetails = d.resolvedDeviceIds[deviceId]
		// device id not found in kubelet as well
		if containerDetails == nil {
			return nil, nil, nil
		}
	}

	// now that we have a resolved container details (from cache or kubelet), lets resolve it to k8s attributes
	// TODO: use cache for this as well
	workloadName, workloadKind, pod, err := d.podInfoResolver.GetWorkloadNameByOwner(ctx, containerDetails.PodNamespace, containerDetails.PodName)
	if err != nil {
		return nil, pod, err
	}

	k8sAttributes := &K8sResourceAttributes{
		Namespace:     containerDetails.PodNamespace,
		PodName:       containerDetails.PodName,
		ContainerName: containerDetails.ContainerName,
		WorkloadKind:  workloadKind,
		WorkloadName:  workloadName,
	}

	d.logger.V(1).Info("resolved device id to container details", "k8sAttributes", k8sAttributes)
	return k8sAttributes, pod, nil
}
