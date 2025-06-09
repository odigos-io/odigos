package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"golang.org/x/sync/errgroup"
)

func ExtractDataStreamsFromEntities(sources []v1alpha1.Source, destinations []v1alpha1.Destination) []*model.DataStream {
	var dataStreams []*model.DataStream
	dataStreams = append(dataStreams, &model.DataStream{Name: "default"})

	// Collect stream names without duplicates
	seen := make(map[string]bool)
	seen["default"] = true

	for _, src := range sources {
		for labelKey, labelValue := range src.Labels {
			if strings.Contains(labelKey, k8sconsts.SourceGroupLabelPrefix) && labelValue == "true" {
				name := strings.TrimPrefix(labelKey, k8sconsts.SourceGroupLabelPrefix)
				if !seen[name] {
					seen[name] = true
					dataStreams = append(dataStreams, &model.DataStream{
						Name: name,
					})
				}
			}
		}

	}

	for _, dest := range destinations {
		if destinationGroupsNotNull(&dest) {
			for _, name := range dest.Spec.SourceSelector.Groups {
				if !seen[name] {
					seen[name] = true
					dataStreams = append(dataStreams, &model.DataStream{
						Name: name,
					})
				}
			}
		}
	}

	return dataStreams
}

// ExtractDataStreamsFromSource extracts data stream names from the given primary and secondary sources.
// It ensures that the data stream names are unique and that the 'false' labels are respected.
// - The 'primarySource' is expected to be a Workload source (or a Namespace source when alone).
// - The 'secondarySource' is expected to be a Namespace source.
func ExtractDataStreamsFromSource(primarySource *v1alpha1.Source, secondarySource *v1alpha1.Source) []*string {
	seen := make(map[string]bool)
	forbidden := make(map[string]bool)
	result := make([]*string, 0)

	// Get all data stream names from the workload source
	if primarySource != nil {
		for labelKey, labelValue := range primarySource.Labels {
			if strings.Contains(labelKey, k8sconsts.SourceGroupLabelPrefix) {
				dsName := strings.TrimPrefix(labelKey, k8sconsts.SourceGroupLabelPrefix)

				if labelValue == "false" {
					forbidden[dsName] = true
				}

				if !seen[dsName] && !forbidden[dsName] {
					seen[dsName] = true
					result = append(result, &dsName)
				}
			}
		}
	}

	// Get all data stream names from the namespace source (if it was not defined as 'false' in the workload source)
	if secondarySource != nil {
		for labelKey, labelValue := range secondarySource.Labels {
			if strings.Contains(labelKey, k8sconsts.SourceGroupLabelPrefix) {
				dsName := strings.TrimPrefix(labelKey, k8sconsts.SourceGroupLabelPrefix)

				if labelValue == "false" {
					continue
				}

				if !seen[dsName] && !forbidden[dsName] {
					seen[dsName] = true
					result = append(result, &dsName)
				}
			}
		}
	}

	return result
}

func destinationGroupsNotNull(destination *v1alpha1.Destination) bool {
	if destination.Spec.SourceSelector != nil && destination.Spec.SourceSelector.Groups != nil {
		return true
	}
	return false
}

func removeStreamNameFromDestination(destination *v1alpha1.Destination, dataStreamName string) {
	if destinationGroupsNotNull(destination) {
		// Remove the current stream name from the source selector
		destination.Spec.SourceSelector.Groups = RemoveStringFromSlice(destination.Spec.SourceSelector.Groups, dataStreamName)
	}
}

func shouldDeleteDestination(destination *v1alpha1.Destination) bool {
	if destinationGroupsNotNull(destination) {
		// If the source selector is not empty after removing the current stream name, we should not delete the destination
		return len(destination.Spec.SourceSelector.Groups) == 0
	}
	return true
}

func DeleteDestinationOrRemoveStreamName(ctx context.Context, dest *v1alpha1.Destination, currentStreamName string) error {
	removeStreamNameFromDestination(dest, currentStreamName)

	if shouldDeleteDestination(dest) {
		if err := deleteDestinationAndSecret(ctx, dest); err != nil {
			return err
		}
	} else {
		if err := UpdateDestination(ctx, dest); err != nil {
			return err
		}
	}

	return nil
}

func DeleteDestinationsOrRemoveStreamName(ctx context.Context, destinations *v1alpha1.DestinationList, currentStreamName string) error {
	g, ctx := errgroup.WithContext(ctx)

	for _, dest := range destinations.Items {
		dest := dest // capture range variable

		g.Go(func() error {
			if destinationGroupsNotNull(&dest) && ArrayContains(dest.Spec.SourceSelector.Groups, currentStreamName) {
				err := DeleteDestinationOrRemoveStreamName(ctx, &dest, currentStreamName)
				if err != nil {
					return fmt.Errorf("failed to delete destination or remove stream name: %v", err)
				}
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	return nil
}

func UpdateDestinationsCurrentStreamName(ctx context.Context, destinations *v1alpha1.DestinationList, currentStreamName string, newStreamName string) error {
	g, ctx := errgroup.WithContext(ctx)

	for _, dest := range destinations.Items {
		dest := dest // capture range variable

		g.Go(func() error {
			if destinationGroupsNotNull(&dest) && ArrayContains(dest.Spec.SourceSelector.Groups, currentStreamName) {
				// Remove the current stream name from the source selector
				dest.Spec.SourceSelector.Groups = RemoveStringFromSlice(dest.Spec.SourceSelector.Groups, currentStreamName)

				// Add the new stream name to the source selector
				if !ArrayContains(dest.Spec.SourceSelector.Groups, newStreamName) {
					dest.Spec.SourceSelector.Groups = append(dest.Spec.SourceSelector.Groups, newStreamName)
				}

				err := UpdateDestination(ctx, &dest)
				if err != nil {
					return err
				}
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	return nil
}

func DeleteSourcesOrRemoveStreamName(ctx context.Context, sources *v1alpha1.SourceList, currentStreamName string) error {
	toPersist := make([]*model.PersistNamespaceSourceInput, 0)

	for _, source := range sources.Items {
		source := source // capture range variable

		for labelKey := range source.Labels {
			if strings.TrimPrefix(labelKey, k8sconsts.SourceGroupLabelPrefix) == currentStreamName {
				toPersist = append(toPersist, &model.PersistNamespaceSourceInput{
					Namespace:         source.Spec.Workload.Namespace,
					Name:              source.Spec.Workload.Name,
					Kind:              model.K8sResourceKind(source.Spec.Workload.Kind),
					Selected:          false, // to remove label, or delete entirely
					CurrentStreamName: currentStreamName,
				})
			}
		}
	}

	err := SyncWorkloadsInNamespace(ctx, toPersist)
	if err != nil {
		return fmt.Errorf("failed to sync workloads: %v", err)
	}

	return nil
}

func UpdateSourcesCurrentStreamName(ctx context.Context, sources *v1alpha1.SourceList, currentStreamName string, newStreamName string) error {
	g, ctx := errgroup.WithContext(ctx)

	for _, source := range sources.Items {
		source := source // capture range variable

		g.Go(func() error {
			for labelKey, labelValue := range source.Labels {
				if strings.TrimPrefix(labelKey, k8sconsts.SourceGroupLabelPrefix) == currentStreamName {
					// remove the old label
					_, err := UpdateSourceCRDLabel(ctx, source.Namespace, source.Name, k8sconsts.SourceGroupLabelPrefix+currentStreamName, "")
					if err != nil {
						return fmt.Errorf("failed to update source %s: %v", source.Name, err)
					}

					// add the new label
					_, err = UpdateSourceCRDLabel(ctx, source.Namespace, source.Name, k8sconsts.SourceGroupLabelPrefix+newStreamName, labelValue)
					if err != nil {
						return fmt.Errorf("failed to update source %s: %v", source.Name, err)
					}

					return nil
				}
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	return nil
}
