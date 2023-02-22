package devices

import (
	"context"
	"github.com/google/uuid"
	"github.com/keyval-dev/odigos/odiglet/pkg/env"
	"github.com/keyval-dev/odigos/odiglet/pkg/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
	"strings"
	"time"
)

const (
	defaultMaxDevices = 100
)

type PodDetails string

func NewPodDetails(name string, namespace string) PodDetails {
	return PodDetails(name + "_" + namespace)
}

var (
	EmptyPodDetails = NewPodDetails("", "")
	defaultPeriod   = 1 * time.Second
)

func (p PodDetails) Details() (string, string) {
	parts := strings.Split(string(p), "_")
	return parts[0], parts[1]
}

type DeviceManager interface {
	Init(initialDevices int64)
	GetDevices() []*v1beta1.Device
}

type IDManager struct {
	devices []string
}

func NewIDManager(clientset *kubernetes.Clientset) (*IDManager, error) {
	initalDevices, err := getInitialDeviceAmount(clientset)
	if err != nil {
		return nil, err
	}

	m := &IDManager{
		devices: make([]string, initalDevices),
	}

	m.Init(initalDevices)
	return m, nil
}

func (m *IDManager) Init(initialDevices int64) {
	for i := int64(0); i < initialDevices; i++ {
		m.devices[i] = uuid.New().String()
	}
}

func (m *IDManager) GetDevices() []*v1beta1.Device {
	var devicesList []*v1beta1.Device
	for _, deviceId := range m.devices {
		devicesList = append(devicesList, &v1beta1.Device{
			ID:     deviceId,
			Health: v1beta1.Healthy,
		})
	}

	return devicesList
}

func getInitialDeviceAmount(clientset *kubernetes.Clientset) (int64, error) {
	// get max pods per current node
	node, err := clientset.CoreV1().Nodes().Get(context.Background(), env.Current.NodeName, metav1.GetOptions{})
	if err != nil {
		return 0, err
	}

	maxPods, ok := node.Status.Allocatable.Pods().AsInt64()
	if !ok {
		log.Logger.V(0).Info("Failed to get max pods from node status, using default value", "default", defaultMaxDevices)
		maxPods = defaultMaxDevices
	}

	return maxPods, nil
}
