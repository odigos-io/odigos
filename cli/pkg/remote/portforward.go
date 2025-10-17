package remote

import (
	"fmt"

	"k8s.io/client-go/tools/portforward"
)

const (
	DefaultAddress = "localhost"
	UiPort         = 3000
)

var (
	DefaultLocalPort = "0"
)

func NewUIClient(fw *portforward.PortForwarder) (*UIClientViaPortForward, error) {
	return &UIClientViaPortForward{
		pf:       fw,
		isClosed: false,
	}, nil
}

type UIClientViaPortForward struct {
	pf       *portforward.PortForwarder
	isClosed bool
}

func (u *UIClientViaPortForward) DiscoverLocalPort() (string, error) {
	ports, err := u.pf.GetPorts()
	if err != nil {
		return "", err
	}

	if len(ports) != 1 {
		return "", fmt.Errorf("expected to get 1 port got %d", len(ports))
	}

	portNum := ports[0].Local
	port := fmt.Sprintf("%d", portNum)
	DefaultLocalPort = port
	return port, nil
}

func (u *UIClientViaPortForward) Start() error {
	return u.pf.ForwardPorts()
}

func (u *UIClientViaPortForward) Ready() <-chan struct{} {
	return u.pf.Ready
}

func (u *UIClientViaPortForward) Close() error {
	// Check if channel is closed
	if !u.isClosed {
		fmt.Println("Closing port-forward to UI pod")
		u.pf.Close()
		u.isClosed = true
	}
	return nil
}
