//go:build linux

package runtime_details

import (
	"context"
	"testing"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func newInspectionResults() InspectionResults {
	return InspectionResults{
		containerDetectedLanguages: make(map[string][]common.ProgramLanguageDetails),
	}
}

func TestCollectDetectedLanguages_SeparateContainers_NoConflict(t *testing.T) {
	t.Parallel()

	// Arrange: two containers, each with a single (different) language — no conflict
	results := newInspectionResults()

	containerALangs := map[int]common.ProgramLanguageDetails{
		100: {Language: common.JavaProgrammingLanguage, RuntimeVersion: "17.0.1"},
	}
	containerBLangs := map[int]common.ProgramLanguageDetails{
		200: {Language: common.PythonProgrammingLanguage, RuntimeVersion: "3.11.0"},
	}

	// Act
	collectDetectedLanguages("container-a", containerALangs, &results)
	collectDetectedLanguages("container-b", containerBLangs, &results)

	// Assert: no multi-language conflict — each container has exactly one language
	assert.Len(t, results.containerDetectedLanguages, 2)
	assert.Len(t, results.containerDetectedLanguages["container-a"], 1)
	assert.Equal(t, common.JavaProgrammingLanguage, results.containerDetectedLanguages["container-a"][0].Language)
	assert.Len(t, results.containerDetectedLanguages["container-b"], 1)
	assert.Equal(t, common.PythonProgrammingLanguage, results.containerDetectedLanguages["container-b"][0].Language)
}

func TestCollectDetectedLanguages_TwoContainersWithConflicts(t *testing.T) {
	t.Parallel()

	// Arrange: two containers, each with a multi-language conflict
	results := newInspectionResults()

	containerALangs := map[int]common.ProgramLanguageDetails{
		100: {Language: common.JavaProgrammingLanguage, RuntimeVersion: "17.0.1"},
		101: {Language: common.PythonProgrammingLanguage, RuntimeVersion: "3.11.0"},
	}
	containerBLangs := map[int]common.ProgramLanguageDetails{
		200: {Language: common.JavascriptProgrammingLanguage, RuntimeVersion: "18.0.0"},
		201: {Language: common.PythonProgrammingLanguage, RuntimeVersion: "3.11.0"},
	}

	// Act
	collectDetectedLanguages("container-a", containerALangs, &results)
	collectDetectedLanguages("container-b", containerBLangs, &results)

	// Assert: both containers have multi-language conflicts
	assert.Len(t, results.containerDetectedLanguages, 2)
	assert.Len(t, results.containerDetectedLanguages["container-a"], 2)
	assert.Len(t, results.containerDetectedLanguages["container-b"], 2)
}

func newInstrumentationConfigClient(t *testing.T, ic *odigosv1.InstrumentationConfig) client.Client {
	t.Helper()

	scheme := runtime.NewScheme()
	require.NoError(t, odigosv1.AddToScheme(scheme))

	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(ic).WithStatusSubresource(ic).Build()
}

func runtimeDetailsFor(t *testing.T, ctx context.Context, kubeclient client.Client, ic *odigosv1.InstrumentationConfig) []odigosv1.RuntimeDetailsByContainer {
	t.Helper()

	persisted := &odigosv1.InstrumentationConfig{}
	require.NoError(t, kubeclient.Get(ctx, client.ObjectKeyFromObject(ic), persisted))
	return persisted.Status.RuntimeDetailsByContainer
}

// A container added to (or renamed in) the workload pod template after the initial detection has
// no entry to merge into, and dropping its results leaves it without runtime details forever.
func TestPersistRuntimeDetails_RecordsContainerWithNoExistingDetails(t *testing.T) {
	ctx := context.Background()

	ic := &odigosv1.InstrumentationConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "deployment-test", Namespace: "default"},
		Status: odigosv1.InstrumentationConfigStatus{
			RuntimeDetailsByContainer: []odigosv1.RuntimeDetailsByContainer{
				{ContainerName: "app", Language: common.GoProgrammingLanguage, RuntimeVersion: "1.24.0"},
			},
		},
	}
	kubeclient := newInstrumentationConfigClient(t, ic)

	results := newInspectionResults()
	results.containerNameToNewRuntimeDetails = map[string]odigosv1.RuntimeDetailsByContainer{
		// the existing container also reports a new runtime version, to verify that recording the
		// new containers does not discard merges done into the existing entries.
		"app":     {ContainerName: "app", Language: common.GoProgrammingLanguage, RuntimeVersion: "1.25.0"},
		"added":   {ContainerName: "added", Language: common.PythonProgrammingLanguage, RuntimeVersion: "3.13.0"},
		"another": {ContainerName: "another", Language: common.JavaProgrammingLanguage, RuntimeVersion: "21.0.1"},
	}

	require.NoError(t, persistRuntimeDetailsToInstrumentationConfig(ctx, kubeclient, ic, results))

	assert.Equal(t, []odigosv1.RuntimeDetailsByContainer{
		{ContainerName: "app", Language: common.GoProgrammingLanguage, RuntimeVersion: "1.25.0"},
		{ContainerName: "added", Language: common.PythonProgrammingLanguage, RuntimeVersion: "3.13.0"},
		{ContainerName: "another", Language: common.JavaProgrammingLanguage, RuntimeVersion: "21.0.1"},
	}, runtimeDetailsFor(t, ctx, kubeclient, ic))
}

func TestPersistRuntimeDetails_KeepsExistingDetailsWhenNothingChanged(t *testing.T) {
	ctx := context.Background()

	ic := &odigosv1.InstrumentationConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "deployment-test", Namespace: "default"},
		Status: odigosv1.InstrumentationConfigStatus{
			RuntimeDetailsByContainer: []odigosv1.RuntimeDetailsByContainer{
				{ContainerName: "app", Language: common.GoProgrammingLanguage, RuntimeVersion: "1.24.0"},
			},
		},
	}
	kubeclient := newInstrumentationConfigClient(t, ic)

	results := newInspectionResults()
	results.containerNameToNewRuntimeDetails = map[string]odigosv1.RuntimeDetailsByContainer{
		"app": {ContainerName: "app", Language: common.GoProgrammingLanguage, RuntimeVersion: "1.24.0"},
	}

	require.NoError(t, persistRuntimeDetailsToInstrumentationConfig(ctx, kubeclient, ic, results))

	assert.Equal(t, []odigosv1.RuntimeDetailsByContainer{
		{ContainerName: "app", Language: common.GoProgrammingLanguage, RuntimeVersion: "1.24.0"},
	}, runtimeDetailsFor(t, ctx, kubeclient, ic))
}
