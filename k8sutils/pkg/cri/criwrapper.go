package criwrapper

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-logr/logr"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	criapi "k8s.io/cri-api/pkg/apis/runtime/v1"
)

type CriClient struct {
	conn   *grpc.ClientConn
	client criapi.RuntimeServiceClient
	Logger logr.Logger
}

// Default runtime endpoints
var defaultRuntimeEndpoints = []string{
	"unix:///run/containerd/containerd.sock",
	"unix:///run/crio/crio.sock",
	"unix:///var/run/cri-dockerd.sock",
}

func detectRuntimeSocket() string {
	for _, endpoint := range defaultRuntimeEndpoints {
		// Extract the file path from the endpoint
		socketPath := strings.TrimPrefix(endpoint, "unix://")
		if _, err := os.Stat(socketPath); err == nil {
			return endpoint
		}
	}
	return ""
}

// Connect attempts to establish a connection to a CRI runtime.
func (rc *CriClient) Connect() error {
	var err error

	endpoint := detectRuntimeSocket()
	if endpoint == "" {
		return fmt.Errorf("unable to detect CRI runtime endpoint")
	}

	rc.Logger.Info("Starting connection attempt to CRI runtime", "endpoint", endpoint)

	rc.conn, err = grpc.NewClient(
		endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	if err != nil {
		return fmt.Errorf("failed to create gRPC connection: %v", err)
	}

	// Create a new RuntimeService client
	rc.client = criapi.NewRuntimeServiceClient(rc.conn)

	// Validate the connection by invoking a lightweight method
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()

	_, err = rc.client.Version(ctx, &criapi.VersionRequest{})
	if err != nil {
		return fmt.Errorf("Failed to validate CRI runtime connection")
	}

	rc.Logger.Info("Successfully connected to CRI runtime", "endpoint", endpoint)
	return nil

}

// GetContainerInfo retrieves the "info" field of the specified container.
func (rc *CriClient) GetContainerInfo(ctx context.Context, containerID string) (map[string]string, error) {
	containerID = extractContainerID(containerID)
	if rc.client == nil {
		return nil, fmt.Errorf("runtime client is not connected")
	}

	// Call the ContainerStatus API
	response, err := rc.client.ContainerStatus(ctx, &criapi.ContainerStatusRequest{
		ContainerId: containerID,
		Verbose:     true,
	})
	if err != nil {
		return nil, err
	}

	return response.GetInfo(), nil
}

func (rc *CriClient) GetContainerEnvVars(ctx context.Context, containerID string) (map[string]string, error) {
	info, err := rc.GetContainerInfo(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get container status: %v", err)
	}

	infoData, found := info["info"]
	if !found {
		return nil, fmt.Errorf("info not found in container status")
	}

	// Parse "info" into JSON structure
	var infoJSON struct {
		RuntimeSpec struct {
			Process struct {
				Env []string `json:"env"`
			} `json:"process"`
		} `json:"runtimeSpec"`
	}

	if err := json.Unmarshal([]byte(infoData), &infoJSON); err != nil {
		return nil, fmt.Errorf("failed to parse runtimeSpec info: %v", err)
	}

	// Convert "KEY=VALUE" to a map
	envVars := make(map[string]string, len(infoJSON.RuntimeSpec.Process.Env))
	for _, env := range infoJSON.RuntimeSpec.Process.Env {
		if parts := strings.SplitN(env, "=", 2); len(parts) == 2 {
			envVars[parts[0]] = parts[1]
		}
	}

	return envVars, nil
}

func (rc *CriClient) GetContainerEnvVarsList(ctx context.Context, envVarKeys []string, containerID string) ([]odigosv1.EnvVar, error) {
	envVars, err := rc.GetContainerEnvVars(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get container environment variables: %v", err)
	}

	// Extract only requested environment variables
	result := make([]odigosv1.EnvVar, 0, len(envVarKeys))
	for _, key := range envVarKeys {
		if value, exists := envVars[key]; exists {
			result = append(result, odigosv1.EnvVar{Name: key, Value: value})
		}
	}

	return result, nil
}

// Close closes the gRPC connection.
func (rc *CriClient) Close() {
	if rc.conn != nil {
		rc.Logger.V(0).Info("gRPC connection is closed")
		rc.conn.Close()
	}
}

// ExtractContainerID extracts the actual container ID from a containerID string.
// The input format is '<type>://<container_id>'.
// If the input is invalid, it returns an empty string.
func extractContainerID(containerID string) string {
	if containerID == "" || !strings.Contains(containerID, "://") {
		return ""
	}
	parts := strings.SplitN(containerID, "://", 2)
	return parts[1]
}
