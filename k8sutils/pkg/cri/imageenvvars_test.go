package criwrapper

import (
	"context"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	criapi "k8s.io/cri-api/pkg/apis/runtime/v1"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/common/envOverwrite"
)

func TestExtractContainerID(t *testing.T) {
	tests := []struct {
		name        string
		containerID string
		want        string
	}{
		{name: "containerd", containerID: "containerd://abc123", want: "abc123"},
		{name: "docker", containerID: "docker://abc123", want: "abc123"},
		{name: "cri-o", containerID: "cri-o://abc123", want: "abc123"},
		{name: "empty", containerID: "", want: ""},
		{name: "no separator", containerID: "abc123", want: ""},
		{name: "separator but no id", containerID: "containerd://", want: ""},
		{name: "no runtime type", containerID: "://abc123", want: "abc123"},
		{name: "separator inside the id is kept", containerID: "containerd://abc://123", want: "abc://123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, extractContainerID(tt.containerID))
		})
	}
}

func TestGetContainerImageEnvVarsStripsTheRuntimePrefix(t *testing.T) {
	runtimeSvc := &criFakeRuntime{statusResp: criContainerStatusWithImage("registry.io/app:v1")}
	imageSvc := &criFakeImage{statusResp: criImageStatusWithEnv(t, []string{"PYTHONPATH=/app"})}
	client := connectedCriClient(t, runtimeSvc, imageSvc)

	_, err := client.GetContainerImageEnvVars(context.Background(), "containerd://abc123")
	require.NoError(t, err)

	// The kubelet reports '<type>://<id>' but the CRI API only accepts the bare id.
	assert.Equal(t, []string{"abc123"}, runtimeSvc.requestedContainerIDs())
}

func TestGetContainerImageEnvVarsAsksForVerboseImageStatus(t *testing.T) {
	runtimeSvc := &criFakeRuntime{statusResp: criContainerStatusWithImage("registry.io/app@sha256:deadbeef")}
	imageSvc := &criFakeImage{statusResp: criImageStatusWithEnv(t, []string{"PYTHONPATH=/app"})}
	client := connectedCriClient(t, runtimeSvc, imageSvc)

	_, err := client.GetContainerImageEnvVars(context.Background(), "containerd://abc123")
	require.NoError(t, err)

	requests := imageSvc.receivedRequests()
	require.Len(t, requests, 1)
	// The image ref must come from the container status, and the CRI contract only populates
	// the "info" document (which carries the env vars) when Verbose is set.
	assert.Equal(t, "registry.io/app@sha256:deadbeef", requests[0].GetImage().GetImage())
	assert.True(t, requests[0].GetVerbose())
}

func TestGetContainerImageEnvVarsRejectsAnUnusableContainerID(t *testing.T) {
	tests := []struct {
		name        string
		containerID string
	}{
		{name: "empty", containerID: ""},
		{name: "no runtime prefix", containerID: "abc123"},
		{name: "prefix without an id", containerID: "containerd://"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runtimeSvc := &criFakeRuntime{statusResp: criContainerStatusWithImage("registry.io/app:v1")}
			client := connectedCriClient(t, runtimeSvc, &criFakeImage{})

			_, err := client.GetContainerImageEnvVars(context.Background(), tt.containerID)

			assert.EqualError(t, err, "invalid container ID")
			assert.Empty(t, runtimeSvc.requestedContainerIDs())
		})
	}
}

func TestGetContainerImageEnvVarsRequiresBothServiceClients(t *testing.T) {
	// Assembled by hand rather than through Connect, because odiglet keeps a CriClient whose
	// Connect may have failed and still calls this method on every pod reconcile.
	connected := connectedCriClient(t, &criFakeRuntime{}, &criFakeImage{})

	tests := []struct {
		name   string
		client *CriClient
	}{
		{name: "neither client", client: &CriClient{}},
		{name: "runtime client only", client: &CriClient{runtimeClient: connected.runtimeClient}},
		{name: "image client only", client: &CriClient{imageClient: connected.imageClient}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.client.GetContainerImageEnvVars(context.Background(), "containerd://abc123")
			assert.EqualError(t, err, "runtime or image client is not connected")
		})
	}
}

func TestGetContainerImageEnvVarsWhenTheContainerStatusFails(t *testing.T) {
	runtimeSvc := &criFakeRuntime{statusErr: status.Error(codes.NotFound, "no such container")}
	client := connectedCriClient(t, runtimeSvc, &criFakeImage{})

	_, err := client.GetContainerImageEnvVars(context.Background(), "containerd://abc123")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get container status")
	assert.Contains(t, err.Error(), "no such container")
}

func TestGetContainerImageEnvVarsWhenTheImageRefIsMissing(t *testing.T) {
	tests := []struct {
		name   string
		status *criapi.ContainerStatusResponse
	}{
		{name: "no status", status: &criapi.ContainerStatusResponse{}},
		{name: "no image", status: &criapi.ContainerStatusResponse{Status: &criapi.ContainerStatus{}}},
		{name: "empty image ref", status: criContainerStatusWithImage("")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imageSvc := &criFakeImage{}
			client := connectedCriClient(t, &criFakeRuntime{statusResp: tt.status}, imageSvc)

			_, err := client.GetContainerImageEnvVars(context.Background(), "containerd://abc123")

			// The message names the stripped id, which is what the runtime was asked about.
			assert.EqualError(t, err, "image ref is empty for container abc123")
			assert.Empty(t, imageSvc.receivedRequests())
		})
	}
}

func TestGetImageEnvVarsFromCRIRejectsAnEmptyImageRef(t *testing.T) {
	client := connectedCriClient(t, &criFakeRuntime{}, &criFakeImage{})

	_, err := client.getImageEnvVarsFromCRI(context.Background(), "")

	assert.EqualError(t, err, "invalid image ref")
}

func TestGetImageEnvVarsFromCRIRequiresAnImageClient(t *testing.T) {
	_, err := (&CriClient{}).getImageEnvVarsFromCRI(context.Background(), "registry.io/app:v1")

	assert.EqualError(t, err, "image client not initialized")
}

func TestGetImageEnvVarsFromCRIWhenImageStatusFails(t *testing.T) {
	imageSvc := &criFakeImage{statusErr: status.Error(codes.Unavailable, "image service down")}
	client := connectedCriClient(t, &criFakeRuntime{}, imageSvc)

	_, err := client.getImageEnvVarsFromCRI(context.Background(), "registry.io/app:v1")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get image status from CRI")
	assert.Contains(t, err.Error(), "image service down")
}

func TestGetImageEnvVarsFromCRIWhenTheInfoDocumentIsAbsent(t *testing.T) {
	tests := []struct {
		name string
		info map[string]string
	}{
		{name: "no info map", info: nil},
		{name: "empty info map", info: map[string]string{}},
		{name: "other keys only", info: map[string]string{"config": `{"imageSpec":{"config":{"Env":["A=1"]}}}`}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imageSvc := &criFakeImage{statusResp: &criapi.ImageStatusResponse{Info: tt.info}}
			client := connectedCriClient(t, &criFakeRuntime{}, imageSvc)

			_, err := client.getImageEnvVarsFromCRI(context.Background(), "registry.io/app:v1")

			assert.EqualError(t, err, "image info not found in response")
		})
	}
}

func TestGetImageEnvVarsFromCRIWhenTheInfoDocumentIsNotJSON(t *testing.T) {
	imageSvc := &criFakeImage{statusResp: &criapi.ImageStatusResponse{
		Info: map[string]string{"info": "not json at all"},
	}}
	client := connectedCriClient(t, &criFakeRuntime{}, imageSvc)

	_, err := client.getImageEnvVarsFromCRI(context.Background(), "registry.io/app:v1")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse image info JSON")
}

func TestGetImageEnvVarsFromCRIParsesTheImageEnvironment(t *testing.T) {
	tests := []struct {
		name string
		env  []string
		want map[string]string
	}{
		{
			name: "no env at all",
			env:  nil,
			want: map[string]string{},
		},
		{
			name: "agent flags are kept verbatim",
			env:  []string{"JAVA_TOOL_OPTIONS=-javaagent:/app/agent.jar -Dfoo=bar"},
			want: map[string]string{"JAVA_TOOL_OPTIONS": "-javaagent:/app/agent.jar -Dfoo=bar"},
		},
		{
			name: "only the first separator splits the entry",
			env:  []string{"NODE_OPTIONS=--require=/a.js --require=/b.js"},
			want: map[string]string{"NODE_OPTIONS": "--require=/a.js --require=/b.js"},
		},
		{
			name: "path lists survive",
			env:  []string{"PYTHONPATH=/app:/app/vendor", "LD_PRELOAD=/lib/a.so:/lib/b.so"},
			want: map[string]string{"PYTHONPATH": "/app:/app/vendor", "LD_PRELOAD": "/lib/a.so:/lib/b.so"},
		},
		{
			// The caller has to tell "the image does not set this" apart from "the image sets
			// this to empty", so an explicitly empty value must survive as a present key.
			name: "an explicitly empty value is a present key",
			env:  []string{"JAVA_TOOL_OPTIONS="},
			want: map[string]string{"JAVA_TOOL_OPTIONS": ""},
		},
		{
			name: "entries without a separator are dropped",
			env:  []string{"NO_SEPARATOR", "", "KEPT=1"},
			want: map[string]string{"KEPT": "1"},
		},
		{
			name: "a repeated key keeps the last value, as the OCI runtime does",
			env:  []string{"NODE_OPTIONS=--require=/first.js", "NODE_OPTIONS=--require=/second.js"},
			want: map[string]string{"NODE_OPTIONS": "--require=/second.js"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imageSvc := &criFakeImage{statusResp: criImageStatusWithEnv(t, tt.env)}
			client := connectedCriClient(t, &criFakeRuntime{}, imageSvc)

			envVars, err := client.getImageEnvVarsFromCRI(context.Background(), "registry.io/app:v1")

			require.NoError(t, err)
			assert.Equal(t, tt.want, envVars)
		})
	}
}

func TestGetImageEnvVarsFromCRIIgnoresEnvOutsideTheImageSpecConfig(t *testing.T) {
	tests := []struct {
		name string
		info string
	}{
		{name: "no imageSpec wrapper", info: `{"config":{"Env":["JAVA_TOOL_OPTIONS=-javaagent:/x.jar"]}}`},
		{name: "no config wrapper", info: `{"imageSpec":{"Env":["JAVA_TOOL_OPTIONS=-javaagent:/x.jar"]}}`},
		{name: "runtime env rather than image env", info: `{"env":["JAVA_TOOL_OPTIONS=-javaagent:/x.jar"]}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imageSvc := &criFakeImage{statusResp: &criapi.ImageStatusResponse{
				Info: map[string]string{"info": tt.info},
			}}
			client := connectedCriClient(t, &criFakeRuntime{}, imageSvc)

			envVars, err := client.getImageEnvVarsFromCRI(context.Background(), "registry.io/app:v1")

			// Only the image's own baked-in config counts. Reading a container's runtime env
			// instead would report Odigos' own injected values as the user's.
			require.NoError(t, err)
			assert.Empty(t, envVars)
		})
	}
}

func TestGetContainerEnvVarsListReturnsTheRequestedKeysInRequestOrder(t *testing.T) {
	imageEnv := []string{
		"JAVA_TOOL_OPTIONS=-javaagent:/app/agent.jar",
		"NODE_OPTIONS=--require=/app/otel.js",
		"PYTHONPATH=/app",
		"UNRELATED=x",
	}

	tests := []struct {
		name string
		keys []string
		want []odigosv1.EnvVar
	}{
		{
			name: "request order is preserved",
			keys: []string{"PYTHONPATH", "JAVA_TOOL_OPTIONS"},
			want: []odigosv1.EnvVar{
				{Name: "PYTHONPATH", Value: "/app"},
				{Name: "JAVA_TOOL_OPTIONS", Value: "-javaagent:/app/agent.jar"},
			},
		},
		{
			// The same keys in the opposite order must come back in that order too, which a
			// result assembled by walking the env map cannot guarantee.
			name: "reversed request order is preserved",
			keys: []string{"JAVA_TOOL_OPTIONS", "PYTHONPATH"},
			want: []odigosv1.EnvVar{
				{Name: "JAVA_TOOL_OPTIONS", Value: "-javaagent:/app/agent.jar"},
				{Name: "PYTHONPATH", Value: "/app"},
			},
		},
		{
			name: "keys the image does not set are omitted",
			keys: []string{"JAVA_TOOL_OPTIONS", "GODEBUG", "NODE_OPTIONS"},
			want: []odigosv1.EnvVar{
				{Name: "JAVA_TOOL_OPTIONS", Value: "-javaagent:/app/agent.jar"},
				{Name: "NODE_OPTIONS", Value: "--require=/app/otel.js"},
			},
		},
		{
			name: "env vars the caller did not ask for are never returned",
			keys: []string{"PYTHONPATH"},
			want: []odigosv1.EnvVar{{Name: "PYTHONPATH", Value: "/app"}},
		},
		{
			name: "no requested keys",
			keys: nil,
			want: []odigosv1.EnvVar{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runtimeSvc := &criFakeRuntime{statusResp: criContainerStatusWithImage("registry.io/app:v1")}
			imageSvc := &criFakeImage{statusResp: criImageStatusWithEnv(t, imageEnv)}
			client := connectedCriClient(t, runtimeSvc, imageSvc)

			envVars, err := client.GetContainerEnvVarsList(context.Background(), tt.keys, "containerd://abc123")

			require.NoError(t, err)
			assert.Equal(t, tt.want, envVars)
		})
	}
}

func TestGetContainerEnvVarsListKeepsAKeyTheImageSetsToEmpty(t *testing.T) {
	runtimeSvc := &criFakeRuntime{statusResp: criContainerStatusWithImage("registry.io/app:v1")}
	imageSvc := &criFakeImage{statusResp: criImageStatusWithEnv(t, []string{"JAVA_TOOL_OPTIONS="})}
	client := connectedCriClient(t, runtimeSvc, imageSvc)

	envVars, err := client.GetContainerEnvVarsList(context.Background(),
		[]string{"JAVA_TOOL_OPTIONS", "NODE_OPTIONS"}, "containerd://abc123")

	require.NoError(t, err)
	assert.Equal(t, []odigosv1.EnvVar{{Name: "JAVA_TOOL_OPTIONS", Value: ""}}, envVars)
}

func TestGetContainerEnvVarsListPropagatesLookupFailures(t *testing.T) {
	runtimeSvc := &criFakeRuntime{statusErr: status.Error(codes.NotFound, "no such container")}
	client := connectedCriClient(t, runtimeSvc, &criFakeImage{})

	envVars, err := client.GetContainerEnvVarsList(context.Background(),
		[]string{"JAVA_TOOL_OPTIONS"}, "containerd://abc123")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get container environment variables")
	// odiglet distinguishes "the image sets nothing" from "the CRI call failed" and falls back
	// to /proc only in the second case, so a failure must never look like an empty result.
	assert.Nil(t, envVars)
}

// Every env var Odigos overwrites has to be readable back out of the image spec: that value is
// what tells odiglet to prepend to the user's setting instead of replacing it. This drives the
// real, shipped key list rather than a copy of it, so a new language is covered automatically.
func TestGetContainerEnvVarsListReadsEveryOverwrittenEnvVar(t *testing.T) {
	require.NotEmpty(t, envOverwrite.EnvVarsForLanguage)

	values := map[string]string{
		"NODE_OPTIONS":             "--require=/user/tracing.js",
		"PYTHONPATH":               "/user/lib:/user/vendor",
		"JAVA_TOOL_OPTIONS":        "-javaagent:/user/agent.jar -Dkey=value",
		consts.LdPreloadEnvVarName: "/user/lib/preload.so",
	}

	languages := make([]common.ProgrammingLanguage, 0, len(envOverwrite.EnvVarsForLanguage))
	for language := range envOverwrite.EnvVarsForLanguage {
		languages = append(languages, language)
	}
	slices.Sort(languages)

	checked := 0
	for _, language := range languages {
		// The production caller appends LD_PRELOAD to the per-language list.
		keys := append(append([]string(nil), envOverwrite.EnvVarsForLanguage[language]...), consts.LdPreloadEnvVarName)

		imageEnv := make([]string, 0, len(keys))
		want := make([]odigosv1.EnvVar, 0, len(keys))
		for _, key := range keys {
			value, ok := values[key]
			require.Truef(t, ok, "no fixture value for env var %q overwritten for %s", key, language)
			imageEnv = append(imageEnv, key+"="+value)
			want = append(want, odigosv1.EnvVar{Name: key, Value: value})
		}

		t.Run(string(language), func(t *testing.T) {
			runtimeSvc := &criFakeRuntime{statusResp: criContainerStatusWithImage("registry.io/app:v1")}
			imageSvc := &criFakeImage{statusResp: criImageStatusWithEnv(t, imageEnv)}
			client := connectedCriClient(t, runtimeSvc, imageSvc)

			envVars, err := client.GetContainerEnvVarsList(context.Background(), keys, "containerd://abc123")

			require.NoError(t, err)
			assert.Equal(t, want, envVars)
		})
		checked += len(keys)
	}

	assert.Positive(t, checked)
}
