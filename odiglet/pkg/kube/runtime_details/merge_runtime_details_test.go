package runtime_details

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
)

func libcPtr(t common.LibCType) *common.LibCType {
	return &t
}

// The libc type is only inspected once the container's language is known
// (procdiscovery/pkg/libc.ShouldInspectForLanguage), so the unknown -> known
// language upgrade that mergeRuntimeDetails explicitly supports is always the
// pass that first carries a libc type.
func TestMergeRuntimeDetails_LibCTypeArrivesWithUnknownToKnownLanguage(t *testing.T) {
	existing := odigosv1.RuntimeDetailsByContainer{
		ContainerName: "app",
		Language:      common.UnknownProgrammingLanguage,
	}
	newDetails := odigosv1.RuntimeDetailsByContainer{
		ContainerName:  "app",
		Language:       common.DotNetProgrammingLanguage,
		RuntimeVersion: "8.0.0",
		LibCType:       libcPtr(common.Glibc),
	}

	updated := mergeRuntimeDetails(&existing, newDetails, "default/deployment-app")

	assert.True(t, updated)
	assert.Equal(t, common.DotNetProgrammingLanguage, existing.Language)
	require.NotNil(t, existing.LibCType, "libc type detected alongside the language must reach the status")
	assert.Equal(t, common.Glibc, *existing.LibCType)
}

// A libc inspection that failed on the first pass (for example the process exited
// before /proc/<pid>/exe could be read) leaves the field nil on a container whose
// language was already resolved. A later successful inspection must fill it in.
func TestMergeRuntimeDetails_LibCTypeRecoversAfterFailedInspection(t *testing.T) {
	existing := odigosv1.RuntimeDetailsByContainer{
		ContainerName: "app",
		Language:      common.DotNetProgrammingLanguage,
	}
	newDetails := odigosv1.RuntimeDetailsByContainer{
		ContainerName: "app",
		Language:      common.DotNetProgrammingLanguage,
		LibCType:      libcPtr(common.Musl),
	}

	updated := mergeRuntimeDetails(&existing, newDetails, "default/deployment-app")

	require.NotNil(t, existing.LibCType)
	assert.Equal(t, common.Musl, *existing.LibCType)
	assert.True(t, updated, "the status has to be written for the recovered libc type to be persisted")
}

func TestMergeRuntimeDetails_LibCTypeChangeIsPickedUp(t *testing.T) {
	existing := odigosv1.RuntimeDetailsByContainer{
		ContainerName: "app",
		Language:      common.DotNetProgrammingLanguage,
		LibCType:      libcPtr(common.Glibc),
	}
	newDetails := odigosv1.RuntimeDetailsByContainer{
		ContainerName: "app",
		Language:      common.DotNetProgrammingLanguage,
		LibCType:      libcPtr(common.Musl),
	}

	updated := mergeRuntimeDetails(&existing, newDetails, "default/deployment-app")

	require.NotNil(t, existing.LibCType)
	assert.Equal(t, common.Musl, *existing.LibCType)
	assert.True(t, updated)
}

// Control: an inspection that did not resolve a libc type must not wipe one that is
// already recorded, and must not report an update on its own.
func TestMergeRuntimeDetails_MissingLibCTypeDoesNotWipeExisting(t *testing.T) {
	existing := odigosv1.RuntimeDetailsByContainer{
		ContainerName: "app",
		Language:      common.DotNetProgrammingLanguage,
		LibCType:      libcPtr(common.Glibc),
	}
	newDetails := odigosv1.RuntimeDetailsByContainer{
		ContainerName: "app",
		Language:      common.DotNetProgrammingLanguage,
	}

	updated := mergeRuntimeDetails(&existing, newDetails, "default/deployment-app")

	require.NotNil(t, existing.LibCType)
	assert.Equal(t, common.Glibc, *existing.LibCType)
	assert.False(t, updated)
}

// End to end over the status write: the first inspection pass stores an unknown
// language, the second resolves .NET with its libc type. The persisted status is what
// instrumentor's addLibcDistroParamFromRuntimeDetails reads to decide whether the
// dotnet-community distro (requireParameters: LIBC_TYPE) can be injected at all.
func TestPersistRuntimeDetails_LibCTypeReachesTheStatus(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, odigosv1.AddToScheme(scheme))

	ic := &odigosv1.InstrumentationConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "deployment-app", Namespace: "default"},
	}
	kubeclient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ic).
		WithStatusSubresource(ic).
		Build()

	ctx := context.Background()

	firstPass := InspectionResults{
		containerNameToNewRuntimeDetails: map[string]odigosv1.RuntimeDetailsByContainer{
			"app": {
				ContainerName: "app",
				Language:      common.UnknownProgrammingLanguage,
			},
		},
		containerDetectedLanguages: map[string][]common.ProgramLanguageDetails{},
	}
	require.NoError(t, persistRuntimeDetailsToInstrumentationConfig(ctx, kubeclient, ic, firstPass))

	secondPass := InspectionResults{
		containerNameToNewRuntimeDetails: map[string]odigosv1.RuntimeDetailsByContainer{
			"app": {
				ContainerName:  "app",
				Language:       common.DotNetProgrammingLanguage,
				RuntimeVersion: "8.0.0",
				LibCType:       libcPtr(common.Glibc),
			},
		},
		containerDetectedLanguages: map[string][]common.ProgramLanguageDetails{},
	}
	require.NoError(t, persistRuntimeDetailsToInstrumentationConfig(ctx, kubeclient, ic, secondPass))

	persisted := &odigosv1.InstrumentationConfig{}
	require.NoError(t, kubeclient.Get(ctx, client.ObjectKeyFromObject(ic), persisted))
	require.Len(t, persisted.Status.RuntimeDetailsByContainer, 1)

	container := persisted.Status.RuntimeDetailsByContainer[0]
	assert.Equal(t, common.DotNetProgrammingLanguage, container.Language)
	require.NotNil(t, container.LibCType, "instrumentor disables the dotnet agent when libCType is nil")
	assert.Equal(t, common.Glibc, *container.LibCType)
}
