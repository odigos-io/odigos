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
	Close()
	UpdatesChannel() chan []*v1beta1.Device
}

type IDManager struct {
	devices       map[string]PodDetails
	updatesChan   chan []*v1beta1.Device
	periodicSync  <-chan time.Time
	stopChan      chan struct{}
	kubeletClient *kubeletClient
}

func NewIDManager(kc *kubeletClient, clientset *kubernetes.Clientset) (*IDManager, error) {
	initalDevices, err := getInitialDeviceAmount(clientset)
	if err != nil {
		return nil, err
	}

	m := &IDManager{
		devices:       make(map[string]PodDetails),
		kubeletClient: kc,
		updatesChan:   make(chan []*v1beta1.Device, initalDevices),
		periodicSync:  time.NewTicker(defaultPeriod).C,
		stopChan:      make(chan struct{}),
	}

	m.Init(initalDevices)
	go m.syncWithKubelet()
	return m, nil
}

func (m *IDManager) Init(initialDevices int64) {
	for i := int64(0); i < initialDevices; i++ {
		m.devices[uuid.New().String()] = EmptyPodDetails
	}

	m.reportUpdates()
}

func (m *IDManager) Close() {
	m.stopChan <- struct{}{}
	m.kubeletClient.Close()
}

func (m *IDManager) UpdatesChannel() chan []*v1beta1.Device {
	return m.updatesChan
}

func (m *IDManager) reportUpdates() {
	var devicesList []*v1beta1.Device
	for deviceId := range m.devices {
		devicesList = append(devicesList, &v1beta1.Device{
			ID:     deviceId,
			Health: v1beta1.Healthy,
		})
	}

	log.Logger.V(0).Info("Updated devices", "devices", devicesList)
	m.updatesChan <- devicesList
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

func (m *IDManager) syncWithKubelet() {
	for {
		select {
		case <-m.periodicSync:
			log.Logger.V(0).Info("Syncing with kubelet")
			currentAllocs, err := m.kubeletClient.GetAllocations()
			if err != nil {
				log.Logger.V(0).Error(err, "Failed to get allocations from kubelet")
				continue
			}
			// Case 1 - new allocation: device shown in currentAllocs but not in m.devices
			for pod, device := range currentAllocs {
				if val := m.devices[device]; val == EmptyPodDetails {
					m.devices[device] = pod
				}
			}

			// Case 2 - device released: device shown in m.devices but not in currentAllocs
			// TODO: WHY DO WE NEED THIS? 
			for device, pod := range m.devices {
				if pod == EmptyPodDetails {
					continue
				}

				if _, exists := currentAllocs[pod]; !exists {
					delete(m.devices, device)
					newId := uuid.New().String()
					m.devices[newId] = EmptyPodDetails
					log.Logger.V(0).Info("Device released", "device", device, "newId", newId)
					m.reportUpdates()
				}
			}
		case <-m.stopChan:
			log.Logger.V(0).Info("Stopping sync with kubelet")
			close(m.updatesChan)
			return
		}
	}
}
