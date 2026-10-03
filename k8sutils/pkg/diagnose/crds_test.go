package diagnose

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	k8stesting "k8s.io/client-go/testing"
	"sigs.k8s.io/yaml"
)

// dgDiscovery serves a fixed discovery result. ServerGroupsAndResources is documented to
// return partial results together with an error, which diagnose relies on.
type dgDiscovery struct {
	discovery.DiscoveryInterface
	lists []*metav1.APIResourceList
	err   error
}

func (d *dgDiscovery) ServerGroupsAndResources() ([]*metav1.APIGroup, []*metav1.APIResourceList, error) {
	return nil, d.lists, d.err
}

func dgResourceList(groupVersion string, resources ...string) *metav1.APIResourceList {
	list := &metav1.APIResourceList{GroupVersion: groupVersion}
	for _, r := range resources {
		list.APIResources = append(list.APIResources, metav1.APIResource{Name: r})
	}
	return list
}

// Only odigos' own CRDs belong in a bundle: a customer's CRDs may hold secrets, and the
// group test is a suffix match that must not be fooled by a lookalike group name.
func TestDiscoverOdigosCRDsSelectsOnlyTheOdigosApiGroups(t *testing.T) {
	discovered := DiscoverOdigosCRDs(&dgDiscovery{lists: []*metav1.APIResourceList{
		dgResourceList("odigos.io/v1alpha1", "destinations", "sources", "sources/status"),
		dgResourceList("actions.odigos.io/v1alpha1", "piimaskings"),
		nil,
		dgResourceList("notodigos.io/v1", "impostors"),
		dgResourceList("odigos.io.example.com/v1", "impostors"),
		dgResourceList("apps/v1", "deployments"),
		dgResourceList("v1", "pods", "secrets"),
		dgResourceList("bad/group/version", "unparsable"),
	}})

	assert.Equal(t, []schema.GroupVersionResource{
		{Group: "odigos.io", Version: "v1alpha1", Resource: "destinations"},
		{Group: "odigos.io", Version: "v1alpha1", Resource: "sources"},
		{Group: "actions.odigos.io", Version: "v1alpha1", Resource: "piimaskings"},
	}, discovered, "subresources, other groups and unparsable group versions are all skipped")
}

func TestDiscoverOdigosCRDsUsesThePartialResultsOfAFailedDiscovery(t *testing.T) {
	// A cluster with a broken aggregated apiserver reports an error for that group while
	// still returning every other group.
	discovered := DiscoverOdigosCRDs(&dgDiscovery{
		lists: []*metav1.APIResourceList{dgResourceList("odigos.io/v1alpha1", "sources")},
		err:   fmt.Errorf("unable to retrieve the complete list of server APIs: metrics.k8s.io/v1beta1"),
	})

	assert.Equal(t, []schema.GroupVersionResource{
		{Group: "odigos.io", Version: "v1alpha1", Resource: "sources"},
	}, discovered)
}

func TestDiscoverOdigosCRDsWithNothingInstalled(t *testing.T) {
	assert.Empty(t, DiscoverOdigosCRDs(&dgDiscovery{}))
}

func TestFetchOdigosCRDsFilesEachObjectUnderItsOwnNamespace(t *testing.T) {
	dynamicClient := dgDynamicClient(
		dgCRDObject("odigos.io/v1alpha1", "Destination", "", "jaeger"),
		dgCRDObject("odigos.io/v1alpha1", "InstrumentationConfig", dgAppNs, "deployment-checkout"),
	)
	discoveryClient := &dgDiscovery{lists: []*metav1.APIResourceList{
		dgResourceList("odigos.io/v1alpha1", "destinations", "instrumentationconfigs"),
	}}
	builder := newDgBuilder()

	require.NoError(t, FetchOdigosCRDs(context.Background(), dynamicClient, discoveryClient, builder, dgRootDir, dgNamespace))

	assert.Equal(t, []string{
		dgRootDir + "/odigos-system/Destinations/jaeger.yaml",
		dgRootDir + "/shop/Instrumentationconfigs/deployment-checkout.yaml",
	}, builder.paths(), "cluster-scoped CRDs go under the odigos namespace, namespaced ones under their own")
}

func TestFetchOdigosCRDsStripsManagedFieldsFromEveryCollectedObject(t *testing.T) {
	object := dgCRDObject("odigos.io/v1alpha1", "Destination", "", "jaeger")
	require.NoError(t, unstructured.SetNestedSlice(object.Object, []interface{}{
		map[string]interface{}{"manager": "odigos-autoscaler"},
	}, "metadata", "managedFields"))
	builder := newDgBuilder()

	require.NoError(t, FetchOdigosCRDs(context.Background(), dgDynamicClient(object),
		&dgDiscovery{lists: []*metav1.APIResourceList{dgResourceList("odigos.io/v1alpha1", "destinations")}},
		builder, dgRootDir, dgNamespace))

	var collected map[string]interface{}
	require.NoError(t, yaml.Unmarshal(builder.body(t, dgRootDir+"/odigos-system/Destinations/jaeger.yaml"), &collected))
	assert.NotContains(t, collected["metadata"], "managedFields")
	assert.Equal(t, "jaeger", collected["metadata"].(map[string]interface{})["name"])
}

// A CLI user whose role is namespaced cannot list across all namespaces; the odigos
// namespace is still readable and is where the components' own CRs live.
func TestCollectCRDFallsBackToTheOdigosNamespaceWhenListingEveryNamespaceIsDenied(t *testing.T) {
	dynamicClient := dgDynamicClient(dgCRDObject("odigos.io/v1alpha1", "Destination", dgNamespace, "jaeger"))
	dynamicClient.PrependReactor("list", "destinations", func(action k8stesting.Action) (bool, runtime.Object, error) {
		if action.GetNamespace() == "" {
			return true, nil, apierrors.NewForbidden(schema.GroupResource{Resource: "destinations"}, "", fmt.Errorf("cluster-wide list denied"))
		}
		return false, nil, nil
	})
	builder := newDgBuilder()

	require.NoError(t, collectCRD(context.Background(), dynamicClient, builder, dgRootDir, dgNamespace,
		odigosGVR("destinations")))

	assert.Equal(t, []string{dgRootDir + "/odigos-system/Destinations/jaeger.yaml"}, builder.paths())
}

func TestCollectCRDReportsWhichResourceItCouldNotList(t *testing.T) {
	dynamicClient := dgDynamicClient()
	dynamicClient.PrependReactor("list", "destinations", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewForbidden(schema.GroupResource{Resource: "destinations"}, "", fmt.Errorf("denied"))
	})

	err := collectCRD(context.Background(), dynamicClient, newDgBuilder(), dgRootDir, dgNamespace,
		odigosGVR("destinations"))

	assert.ErrorContains(t, err, "failed to list destinations")
	assert.True(t, apierrors.IsForbidden(err), "the cause must survive wrapping so the reason is visible in the stage status")
}

func TestFetchOdigosCRDsReportsEveryResourceItFailedOn(t *testing.T) {
	dynamicClient := dgDynamicClient()
	for _, resource := range []string{"destinations", "sources"} {
		dynamicClient.PrependReactor("list", resource, func(k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, apierrors.NewForbidden(schema.GroupResource{Resource: resource}, "", fmt.Errorf("denied"))
		})
	}

	err := FetchOdigosCRDs(context.Background(), dynamicClient,
		&dgDiscovery{lists: []*metav1.APIResourceList{dgResourceList("odigos.io/v1alpha1", "destinations", "sources")}},
		newDgBuilder(), dgRootDir, dgNamespace)

	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to collect some CRDs")
	assert.ErrorContains(t, err, "destinations")
	assert.ErrorContains(t, err, "sources")
}

func TestCollectCRDReportsABuilderFailureAgainstTheObjectItWasWriting(t *testing.T) {
	builder := newDgBuilder()
	builder.failOn = func(string, string) error { return fmt.Errorf("no space left on device") }

	err := collectCRD(context.Background(), dgDynamicClient(dgCRDObject("odigos.io/v1alpha1", "Destination", "", "jaeger")),
		builder, dgRootDir, dgNamespace, odigosGVR("destinations"))

	assert.ErrorContains(t, err, "failed to add CRD jaeger to collection")
}

func TestCollectCRDWritesNothingForAResourceWithNoObjects(t *testing.T) {
	builder := newDgBuilder()

	require.NoError(t, collectCRD(context.Background(), dgDynamicClient(), builder, dgRootDir, dgNamespace,
		odigosGVR("destinations")))

	assert.Empty(t, builder.paths())
}

func TestCapitalizeFirst(t *testing.T) {
	for input, expected := range map[string]string{
		"":                       "",
		"destinations":           "Destinations",
		"instrumentationconfigs": "Instrumentationconfigs",
		"Sources":                "Sources",
		"piimaskings":            "Piimaskings",
	} {
		assert.Equal(t, expected, capitalizeFirst(input))
	}
}

func TestFetchDestinationsReadsTheOdigosDestinationResource(t *testing.T) {
	dynamicClient := dgDynamicClient(dgCRDObject("odigos.io/v1alpha1", "Destination", "", "jaeger"))
	var listed []schema.GroupVersionResource
	dynamicClient.PrependReactor("list", "*", func(action k8stesting.Action) (bool, runtime.Object, error) {
		listed = append(listed, action.GetResource())
		return false, nil, nil
	})
	builder := newDgBuilder()

	require.NoError(t, FetchDestinations(context.Background(), dynamicClient, builder, dgRootDir, dgNamespace))

	require.NotEmpty(t, listed)
	assert.Equal(t, "odigos.io", listed[0].Group)
	assert.Equal(t, "v1alpha1", listed[0].Version)
	assert.Equal(t, "destinations", listed[0].Resource)
	assert.Equal(t, []string{dgRootDir + "/odigos-system/Destinations/jaeger.yaml"}, builder.paths())
}

func dgCRDObject(apiVersion, kind, namespace, name string) *unstructured.Unstructured {
	metadata := map[string]interface{}{"name": name}
	if namespace != "" {
		metadata["namespace"] = namespace
	}
	return &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": apiVersion,
		"kind":       kind,
		"metadata":   metadata,
	}}
}
