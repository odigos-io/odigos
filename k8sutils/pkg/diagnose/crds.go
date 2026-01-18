package diagnose

import (
	"context"
	"fmt"
	"path"
	"strings"
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"
)

// DiscoverOdigosCRDs uses the discovery client to find all CRDs in groups ending with ".odigos.io"
func DiscoverOdigosCRDs(discoveryClient discovery.DiscoveryInterface) []schema.GroupVersionResource {
	var gvrs []schema.GroupVersionResource

	// Get all API groups and resources
	// Note: ServerGroupsAndResources can return partial results with errors for some groups,
	// which is normal. We use the results we get and ignore errors.
	_, apiResourceLists, _ := discoveryClient.ServerGroupsAndResources()

	for _, resourceList := range apiResourceLists {
		if resourceList == nil {
			continue
		}

		// Parse the GroupVersion
		gv, err := schema.ParseGroupVersion(resourceList.GroupVersion)
		if err != nil {
			continue
		}

		// Check if this group is "odigos.io" or ends with ".odigos.io" (e.g., "actions.odigos.io")
		if gv.Group != odigosGroupName && !strings.HasSuffix(gv.Group, "."+odigosGroupName) {
			continue
		}

		// Add all resources from this group
		for i := 0; i < len(resourceList.APIResources); i++ {
			resource := &resourceList.APIResources[i]
			// Skip subresources (they contain "/")
			if strings.Contains(resource.Name, "/") {
				continue
			}

			gvrs = append(gvrs, schema.GroupVersionResource{
				Group:    gv.Group,
				Version:  gv.Version,
				Resource: resource.Name,
			})
		}
	}

	return gvrs
}

// FetchOdigosCRDs collects all Odigos CRDs dynamically
// CRDs are organized by namespace:
// - Cluster-scoped CRDs go under the odigos namespace folder
// - Namespace-scoped CRDs go under their respective namespace folders
func FetchOdigosCRDs(
	ctx context.Context,
	dynamicClient dynamic.Interface,
	discoveryClient discovery.DiscoveryInterface,
	collector Collector,
	rootDir, odigosNamespace string,
) error {
	fmt.Printf("Fetching Odigos CRDs...\n")
	klog.V(2).InfoS("Fetching Odigos CRDs")

	gvrs := DiscoverOdigosCRDs(discoveryClient)

	var wg sync.WaitGroup
	errChan := make(chan error, len(gvrs))

	for _, gvr := range gvrs {
		gvr := gvr // capture range variable
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := collectCRD(ctx, dynamicClient, collector, rootDir, odigosNamespace, gvr); err != nil {
				klog.V(1).ErrorS(err, "Failed to collect CRD", "resource", gvr.Resource)
				errChan <- err
			}
		}()
	}

	wg.Wait()
	close(errChan)

	// Collect any errors
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to collect some CRDs: %v", errs)
	}

	return nil
}

func collectCRD(
	ctx context.Context,
	dynamicClient dynamic.Interface,
	collector Collector,
	rootDir, odigosNamespace string,
	gvr schema.GroupVersionResource,
) error {
	// Try to list from all namespaces first (works for both namespaced and cluster-scoped resources)
	list, err := dynamicClient.Resource(gvr).List(ctx, metav1.ListOptions{})
	if err != nil {
		// If all-namespace list fails, try namespace-scoped
		list, err = dynamicClient.Resource(gvr).Namespace(odigosNamespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return fmt.Errorf("failed to list %s: %w", gvr.Resource, err)
		}
	}

	if len(list.Items) == 0 {
		return nil
	}

	// Use capitalized resource name as directory (e.g., "destinations" -> "Destinations")
	crdTypeName := capitalizeFirst(gvr.Resource)

	for i := range list.Items {
		item := &list.Items[i]

		// Clean managedFields from the object
		unstructured.RemoveNestedField(item.Object, "metadata", "managedFields")

		// Determine the namespace folder:
		// - Cluster-scoped resources (no namespace) go under odigos namespace
		// - Namespace-scoped resources go under their own namespace
		itemNs := item.GetNamespace()
		if itemNs == "" {
			itemNs = odigosNamespace // cluster-scoped resources go under odigos namespace
		}

		// Create directory path: rootDir/<namespace>/<CRDType>/
		crdTypeDir := path.Join(rootDir, itemNs, crdTypeName)

		// Marshal to YAML
		yamlData, err := yaml.Marshal(item.Object)
		if err != nil {
			klog.V(1).ErrorS(err, "Failed to marshal CRD to YAML", "name", item.GetName())
			continue
		}

		filename := fmt.Sprintf("%s.yaml", item.GetName())
		if err := collector.AddFile(crdTypeDir, filename, yamlData); err != nil {
			return fmt.Errorf("failed to add CRD %s to collection: %w", item.GetName(), err)
		}
	}

	return nil
}

// capitalizeFirst capitalizes the first letter of a string
func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// FetchDestinations collects destination CRDs specifically
// This is kept for backward compatibility with the CLI approach
func FetchDestinations(ctx context.Context, dynamicClient dynamic.Interface, collector Collector, rootDir, odigosNamespace string) error {
	gvr := schema.GroupVersionResource{
		Group:    odigosGroupName,
		Version:  "v1alpha1",
		Resource: "destinations",
	}

	return collectCRD(ctx, dynamicClient, collector, rootDir, odigosNamespace, gvr)
}
