package criwrapper

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"google.golang.org/grpc"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
)

type RuntimeClient struct {
	conn   *grpc.ClientConn
	client runtimeapi.RuntimeServiceClient
}

// Default runtime endpoints
var defaultRuntimeEndpoints = []string{
	"unix:///run/containerd/containerd.sock",
	"unix:///run/crio/crio.sock",
	"unix:///var/run/cri-dockerd.sock",
}

// Connect attempts to establish a connection to a CRI runtime.
func (rc *RuntimeClient) Connect(ctx context.Context) error {
	var err error

	for _, endpoint := range defaultRuntimeEndpoints {
		log.Printf("Attempting to connect to CRI runtime at %s", endpoint)

		// Attempt to dial the gRPC server
		rc.conn, err = grpc.DialContext(
			ctx,
			endpoint,
			grpc.WithInsecure(),
			grpc.WithBlock(),
		)
		if err != nil {
			log.Printf("Failed to connect to %s: %v", endpoint, err)
			continue
		}

		// Create a new RuntimeService client
		rc.client = runtimeapi.NewRuntimeServiceClient(rc.conn)
		log.Printf("Successfully connected to CRI runtime at %s", endpoint)
		return nil
	}

	return fmt.Errorf("unable to connect to any CRI runtime endpoints: %v", defaultRuntimeEndpoints)
}

// GetContainerStatus retrieves the status and info of the specified container.
func (rc *RuntimeClient) GetContainerStatus(ctx context.Context, containerID string) (*runtimeapi.ContainerStatus, map[string]string, error) {
	if rc.client == nil {
		return nil, nil, fmt.Errorf("runtime client is not connected")
	}

	// Create a request to get the container's status
	request := &runtimeapi.ContainerStatusRequest{
		ContainerId: containerID,
		Verbose:     true,
	}

	// Call the ContainerStatus API
	response, err := rc.client.ContainerStatus(ctx, request)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get container status: %v", err)
	}

	// Return the container status and the "info" field
	return response.GetStatus(), response.GetInfo(), nil
}

func (rc *RuntimeClient) GetContainerEnvVars(ctx context.Context, containerID string) (map[string]string, error) {
	_, info, err := rc.GetContainerStatus(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get container status: %v", err)
	}

	// Parse the nested "runtimeSpec" from the "info" JSON string
	infoData, ok := info["info"]
	if !ok {
		return nil, fmt.Errorf("info not found in container status")
	}

	var infoJSON map[string]interface{}
	if err := json.Unmarshal([]byte(infoData), &infoJSON); err != nil {
		return nil, fmt.Errorf("failed to parse info: %v", err)
	}

	runtimeSpecData, ok := infoJSON["runtimeSpec"]
	if !ok {
		return nil, fmt.Errorf("runtimeSpec not found in info")
	}

	// Parse the "runtimeSpec" JSON string
	var runtimeSpec struct {
		Process struct {
			Env []string `json:"env"`
		} `json:"process"`
	}
	runtimeSpecBytes, err := json.Marshal(runtimeSpecData)
	if err != nil {
		return nil, fmt.Errorf("failed to re-marshal runtimeSpec: %v", err)
	}

	if err := json.Unmarshal(runtimeSpecBytes, &runtimeSpec); err != nil {
		return nil, fmt.Errorf("failed to parse runtimeSpec: %v", err)
	}

	// Convert environment variables from "KEY=VALUE" to a map
	envVars := make(map[string]string)
	for _, env := range runtimeSpec.Process.Env {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			envVars[parts[0]] = parts[1]
		}
	}

	return envVars, nil
}

// Close closes the gRPC connection.
func (rc *RuntimeClient) Close() {
	if rc.conn != nil {
		rc.conn.Close()
	}
}
