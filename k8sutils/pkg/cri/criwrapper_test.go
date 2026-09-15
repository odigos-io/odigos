package criwrapper

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	criapi "k8s.io/cri-api/pkg/apis/runtime/v1"

	"github.com/odigos-io/odigos/api/k8sconsts"
)

// criFakeRuntime serves canned RuntimeService answers over a real gRPC connection, so the
// tests drive the same generated client stubs odiglet uses against a live container runtime.
type criFakeRuntime struct {
	criapi.UnimplementedRuntimeServiceServer

	versionErr error
	statusResp *criapi.ContainerStatusResponse
	statusErr  error

	mu sync.Mutex
	// containerIDs records the ContainerId of every ContainerStatus request, so a test can
	// assert which identifier actually reached the runtime.
	containerIDs []string
}

func (f *criFakeRuntime) Version(_ context.Context, _ *criapi.VersionRequest) (*criapi.VersionResponse, error) {
	if f.versionErr != nil {
		return nil, f.versionErr
	}
	return &criapi.VersionResponse{Version: "0.1.0", RuntimeName: "fake"}, nil
}

func (f *criFakeRuntime) ContainerStatus(_ context.Context, req *criapi.ContainerStatusRequest) (*criapi.ContainerStatusResponse, error) {
	f.mu.Lock()
	f.containerIDs = append(f.containerIDs, req.GetContainerId())
	f.mu.Unlock()

	if f.statusErr != nil {
		return nil, f.statusErr
	}
	if f.statusResp == nil {
		return &criapi.ContainerStatusResponse{}, nil
	}
	return f.statusResp, nil
}

func (f *criFakeRuntime) requestedContainerIDs() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.containerIDs...)
}

// criFakeImage serves canned ImageService answers and records the requests it received.
type criFakeImage struct {
	criapi.UnimplementedImageServiceServer

	statusResp *criapi.ImageStatusResponse
	statusErr  error

	mu       sync.Mutex
	requests []*criapi.ImageStatusRequest
}

func (f *criFakeImage) ImageStatus(_ context.Context, req *criapi.ImageStatusRequest) (*criapi.ImageStatusResponse, error) {
	f.mu.Lock()
	f.requests = append(f.requests, req)
	f.mu.Unlock()

	if f.statusErr != nil {
		return nil, f.statusErr
	}
	if f.statusResp == nil {
		return &criapi.ImageStatusResponse{}, nil
	}
	return f.statusResp, nil
}

func (f *criFakeImage) receivedRequests() []*criapi.ImageStatusRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]*criapi.ImageStatusRequest(nil), f.requests...)
}

// criImageStatusWithEnv builds the verbose ImageStatus response a container runtime returns,
// where the image config lives in the "info" entry as a JSON document.
func criImageStatusWithEnv(t *testing.T, env []string) *criapi.ImageStatusResponse {
	t.Helper()

	info := map[string]any{
		"imageSpec": map[string]any{
			"config": map[string]any{
				"Env": env,
			},
		},
	}
	raw, err := json.Marshal(info)
	require.NoError(t, err)

	return &criapi.ImageStatusResponse{Info: map[string]string{"info": string(raw)}}
}

// criContainerStatusWithImage builds a ContainerStatus response pointing at an image ref.
func criContainerStatusWithImage(imageRef string) *criapi.ContainerStatusResponse {
	return &criapi.ContainerStatusResponse{
		Status: &criapi.ContainerStatus{
			Image: &criapi.ImageSpec{Image: imageRef},
		},
	}
}

// startCriTestServer serves the two CRI services on a unix socket and returns its path.
// The socket does not live under t.TempDir() because unix socket paths are capped at ~100
// bytes and the subtest names in this package would overflow that.
func startCriTestServer(t *testing.T, runtimeSvc criapi.RuntimeServiceServer, imageSvc criapi.ImageServiceServer) string {
	t.Helper()

	dir, err := os.MkdirTemp("", "cri")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	socketPath := filepath.Join(dir, "cri.sock")
	listener, err := net.Listen("unix", socketPath)
	require.NoError(t, err)

	server := grpc.NewServer()
	if runtimeSvc != nil {
		criapi.RegisterRuntimeServiceServer(server, runtimeSvc)
	}
	if imageSvc != nil {
		criapi.RegisterImageServiceServer(server, imageSvc)
	}

	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		_ = server.Serve(listener)
	}()
	t.Cleanup(func() {
		server.Stop()
		<-stopped
	})

	return socketPath
}

// connectedCriClient points the socket detection at a live test server and connects to it.
func connectedCriClient(t *testing.T, runtimeSvc criapi.RuntimeServiceServer, imageSvc criapi.ImageServiceServer) *CriClient {
	t.Helper()

	socketPath := startCriTestServer(t, runtimeSvc, imageSvc)
	t.Setenv(k8sconsts.CustomContainerRuntimeSocketEnvVar, socketPath)

	client := &CriClient{}
	require.NoError(t, client.Connect(context.Background()))
	t.Cleanup(client.Close)

	return client
}

// withDefaultRuntimeEndpoints replaces the package level fallback list for one test. The
// tests in this package therefore must not run in parallel.
func withDefaultRuntimeEndpoints(t *testing.T, endpoints []string) {
	t.Helper()

	original := defaultRuntimeEndpoints
	defaultRuntimeEndpoints = endpoints
	t.Cleanup(func() { defaultRuntimeEndpoints = original })
}

// criExistingSocketPath creates a file that os.Stat can find, standing in for a runtime socket.
func criExistingSocketPath(t *testing.T, name string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, nil, 0o600))

	return path
}

// The four fallback endpoints are the socket paths of the container runtimes Odigos claims to
// support. Dropping one silently disables runtime inspection on that runtime rather than
// failing, so the list is pinned literally.
func TestDefaultRuntimeEndpointsCoverTheSupportedRuntimes(t *testing.T) {
	assert.Equal(t, []string{
		"unix:///run/containerd/containerd.sock",
		"unix:///run/crio/crio.sock",
		"unix:///var/run/cri-dockerd.sock",
		"unix:///run/k3s/containerd/containerd.sock",
	}, defaultRuntimeEndpoints)
}

func TestDetectRuntimeSocketPrefersTheConfiguredSocket(t *testing.T) {
	existing := criExistingSocketPath(t, "containerd.sock")
	withDefaultRuntimeEndpoints(t, []string{"unix://" + existing})
	t.Setenv(k8sconsts.CustomContainerRuntimeSocketEnvVar, "/custom/runtime.sock")

	// The configured value is a bare path; gRPC needs the unix:// scheme added to it.
	assert.Equal(t, "unix:///custom/runtime.sock", detectRuntimeSocket())
}

func TestDetectRuntimeSocketDoesNotRequireTheConfiguredSocketToExist(t *testing.T) {
	withDefaultRuntimeEndpoints(t, []string{"unix://" + criExistingSocketPath(t, "containerd.sock")})
	missing := filepath.Join(t.TempDir(), "absent.sock")
	t.Setenv(k8sconsts.CustomContainerRuntimeSocketEnvVar, missing)

	assert.Equal(t, "unix://"+missing, detectRuntimeSocket())
}

func TestDetectRuntimeSocketFallsBackToTheFirstExistingDefault(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "absent.sock")
	second := criExistingSocketPath(t, "crio.sock")
	third := criExistingSocketPath(t, "cri-dockerd.sock")
	withDefaultRuntimeEndpoints(t, []string{
		"unix://" + missing,
		"unix://" + second,
		"unix://" + third,
	})
	t.Setenv(k8sconsts.CustomContainerRuntimeSocketEnvVar, "")

	// The scheme is stripped only to stat the path; the endpoint is returned with it.
	assert.Equal(t, "unix://"+second, detectRuntimeSocket())
}

func TestDetectRuntimeSocketReturnsEmptyWhenNoSocketExists(t *testing.T) {
	dir := t.TempDir()
	withDefaultRuntimeEndpoints(t, []string{
		"unix://" + filepath.Join(dir, "containerd.sock"),
		"unix://" + filepath.Join(dir, "crio.sock"),
	})
	t.Setenv(k8sconsts.CustomContainerRuntimeSocketEnvVar, "")

	assert.Empty(t, detectRuntimeSocket())
}

func TestConnectWiresBothServiceClients(t *testing.T) {
	runtimeSvc := &criFakeRuntime{statusResp: criContainerStatusWithImage("registry.io/app@sha256:abc")}
	imageSvc := &criFakeImage{statusResp: criImageStatusWithEnv(t, []string{"JAVA_TOOL_OPTIONS=-javaagent:/app/agent.jar"})}

	client := connectedCriClient(t, runtimeSvc, imageSvc)

	require.NotNil(t, client.conn)
	require.NotNil(t, client.runtimeClient)
	require.NotNil(t, client.imageClient)

	// Reading env vars needs both services, so a successful round trip proves neither client
	// was left unset or pointed at the wrong connection.
	envVars, err := client.GetContainerImageEnvVars(context.Background(), "containerd://c1")
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"JAVA_TOOL_OPTIONS": "-javaagent:/app/agent.jar"}, envVars)
}

func TestConnectWithoutADetectableSocket(t *testing.T) {
	withDefaultRuntimeEndpoints(t, []string{"unix://" + filepath.Join(t.TempDir(), "containerd.sock")})
	t.Setenv(k8sconsts.CustomContainerRuntimeSocketEnvVar, "")

	client := &CriClient{}
	err := client.Connect(context.Background())

	assert.ErrorIs(t, err, ErrDetectingCRIEndpoint)
	assert.Nil(t, client.conn)
	assert.Nil(t, client.runtimeClient)
	assert.Nil(t, client.imageClient)
}

func TestConnectWhenTheRuntimeRejectsTheVersionProbe(t *testing.T) {
	runtimeSvc := &criFakeRuntime{versionErr: status.Error(codes.Unimplemented, "no version for you")}
	socketPath := startCriTestServer(t, runtimeSvc, &criFakeImage{})
	t.Setenv(k8sconsts.CustomContainerRuntimeSocketEnvVar, socketPath)

	client := &CriClient{}
	t.Cleanup(client.Close)

	assert.ErrorIs(t, client.Connect(context.Background()), ErrFailedToValidateCRIConnection)
	// odiglet only logs this error and carries on behind a deferred Close, so the dialled
	// connection has to stay reachable for Close to release it.
	assert.NotNil(t, client.conn)
}

func TestConnectWhenNothingIsListeningOnTheSocket(t *testing.T) {
	t.Setenv(k8sconsts.CustomContainerRuntimeSocketEnvVar, filepath.Join(t.TempDir(), "absent.sock"))

	client := &CriClient{}
	t.Cleanup(client.Close)

	// grpc.NewClient dials lazily, so an unreachable socket only surfaces on the version probe.
	assert.ErrorIs(t, client.Connect(context.Background()), ErrFailedToValidateCRIConnection)
}

func TestConnectHonoursTheCallerContext(t *testing.T) {
	socketPath := startCriTestServer(t, &criFakeRuntime{}, &criFakeImage{})
	t.Setenv(k8sconsts.CustomContainerRuntimeSocketEnvVar, socketPath)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	client := &CriClient{}
	t.Cleanup(client.Close)

	// The validation timeout must be derived from the caller's context, not a fresh one, so a
	// cancelled odiglet startup does not sit in the version probe.
	assert.ErrorIs(t, client.Connect(ctx), ErrFailedToValidateCRIConnection)
}

func TestCloseWithoutAConnection(t *testing.T) {
	assert.NotPanics(t, (&CriClient{}).Close)
}

func TestCloseTerminatesTheConnection(t *testing.T) {
	runtimeSvc := &criFakeRuntime{statusResp: criContainerStatusWithImage("registry.io/app:v1")}
	imageSvc := &criFakeImage{statusResp: criImageStatusWithEnv(t, []string{"NODE_OPTIONS=--require /a.js"})}

	socketPath := startCriTestServer(t, runtimeSvc, imageSvc)
	t.Setenv(k8sconsts.CustomContainerRuntimeSocketEnvVar, socketPath)

	client := &CriClient{}
	require.NoError(t, client.Connect(context.Background()))

	_, err := client.GetContainerImageEnvVars(context.Background(), "containerd://c1")
	require.NoError(t, err)

	client.Close()

	_, err = client.GetContainerImageEnvVars(context.Background(), "containerd://c1")
	assert.Error(t, err)

	// Closing an already closed connection is reported by gRPC; it must not panic.
	assert.NotPanics(t, client.Close)
}
