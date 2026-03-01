package criwrapper

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	criapi "k8s.io/cri-api/pkg/apis/runtime/v1"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	commonlogger "github.com/odigos-io/odigos/common/logger"
)

var (
	ErrDetectingCRIEndpoint          = errors.New("Unable to detect CRI runtime endpoint")
	ErrFailedToValidateCRIConnection = errors.New("Failed to validate CRI runtime connection")
)

type CriClient struct {
	conn          *grpc.ClientConn
	imageClient   criapi.ImageServiceClient
	runtimeClient criapi.RuntimeServiceClient
}

// Default runtime endpoints
var defaultRuntimeEndpoints = []string{
	"unix:///run/containerd/containerd.sock",
	"unix:///run/crio/crio.sock",
	"unix:///var/run/cri-dockerd.sock",
	"unix:///run/k3s/containerd/containerd.sock",
}

func detectRuntimeSocket() string {
	// CONTAINER_RUNTIME_SOCK environment variable is set when the user specifies
	// a custom container runtime socket path in the Odigos configuration
	if envSocket := os.Getenv(k8sconsts.CustomContainerRuntimeSocketEnvVar); envSocket != "" {
		return fmt.Sprintf("unix://%s", envSocket)
	}

	// Fallback to checking the default runtime endpoints
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
func (rc *CriClient) Connect(ctx context.Context) error {
	logger := commonlogger.Logger().With("subsystem", "cri")
	var err error

	endpoint := detectRuntimeSocket()
	if endpoint == "" {
		return ErrDetectingCRIEndpoint
	}

	logger.Info("Starting connection attempt to CRI runtime", "endpoint", endpoint)

	rc.conn, err = grpc.NewClient(
		endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	if err != nil {
		return fmt.Errorf("failed to create gRPC connection: %v", err)
	}

	// Create a new RuntimeService client
	rc.runtimeClient = criapi.NewRuntimeServiceClient(rc.conn)

	// Create a new ImageServiceClient
	rc.imageClient = criapi.NewImageServiceClient(rc.conn)

	// Validate the connection by invoking a lightweight method
	ctx, cancel := context.WithTimeout(ctx, time.Second*2)
	defer cancel()

	_, err = rc.runtimeClient.Version(ctx, &criapi.VersionRequest{})
	if err != nil {
		return ErrFailedToValidateCRIConnection
	}

	logger.Info("Successfully connected to CRI runtime", "endpoint", endpoint)
	return nil
}

func (rc *CriClient) GetContainerEnvVarsList(ctx context.Context, envVarKeys []string, containerID string) ([]odigosv1.EnvVar, error) {
	envVars, err := rc.GetContainerImageEnvVars(ctx, containerID)
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
		logger := commonlogger.Logger().With("subsystem", "cri")
		logger.Info("Closing gRPC connection")
		if err := rc.conn.Close(); err != nil {
			logger.Error("Failed to close gRPC connection", "err", err)
		} else {
			logger.Info("gRPC connection closed successfully")
		}
	}
}

// ExtractContainerID extracts the actual container ID from a containerID string.
// The input format is '<type>://<container_id>'.
// If the input is invalid, it returns an empty string.
func extractContainerID(containerUri string) string {
	if containerUri == "" || !strings.Contains(containerUri, "://") {
		return ""
	}
	parts := strings.SplitN(containerUri, "://", 2)
	return parts[1]
}

func (rc *CriClient) GetContainerImageEnvVars(ctx context.Context, containerID string) (map[string]string, error) {
	containerID = extractContainerID(containerID)
	if containerID == "" {
		return nil, errors.New("invalid container ID")
	}

	if rc.runtimeClient == nil || rc.imageClient == nil {
		return nil, errors.New("runtime or image client is not connected")
	}
	// Set a timeout for the request
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	// Step 1: Get the container status to fetch the image ID
	statusResp, err := rc.runtimeClient.ContainerStatus(timeoutCtx, &criapi.ContainerStatusRequest{
		ContainerId: containerID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get container status: %v", err)
	}
	imageRef := statusResp.GetStatus().GetImage().GetImage()
	if imageRef == "" {
		return nil, fmt.Errorf("image ref is empty for container %s", containerID)
	}

	// Step 2: Use CRI image service to inspect image env vars
	return rc.getImageEnvVarsFromCRI(ctx, imageRef)
}

func (rc *CriClient) getImageEnvVarsFromCRI(ctx context.Context, imageRef string) (map[string]string, error) {
	if imageRef == "" {
		return nil, errors.New("invalid image ref")
	}

	if rc.imageClient == nil {
		return nil, errors.New("image client not initialized")
	}

	resp, err := rc.imageClient.ImageStatus(ctx, &criapi.ImageStatusRequest{
		Image:   &criapi.ImageSpec{Image: imageRef},
		Verbose: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get image status from CRI: %v", err)
	}

	infoMap := resp.GetInfo()
	infoRaw, ok := infoMap["info"]
	if !ok {
		return nil, errors.New("image info not found in response")
	}

	var infoJSON struct {
		ImageSpec struct {
			Config struct {
				Env []string `json:"Env"`
			} `json:"config"`
		} `json:"imageSpec"`
	}

	if err := json.Unmarshal([]byte(infoRaw), &infoJSON); err != nil {
		return nil, fmt.Errorf("failed to parse image info JSON: %v", err)
	}
	envVars := make(map[string]string, len(infoJSON.ImageSpec.Config.Env))
	for _, env := range infoJSON.ImageSpec.Config.Env {
		if parts := strings.SplitN(env, "=", 2); len(parts) == 2 {
			envVars[parts[0]] = parts[1]
		}
	}
	return envVars, nil
}
