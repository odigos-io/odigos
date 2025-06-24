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
			if strings.HasPrefix(labelKey, k8sconsts.SourceDataStreamLabelPrefix) {
				nameFromLabel := strings.TrimPrefix(labelKey, k8sconsts.SourceDataStreamLabelPrefix)
				ds := nameFromLabel

				if labelValue == "false" {
					forbidden[ds] = true
				}

				if !seen[ds] && !forbidden[ds] {
					seen[ds] = true
					result = append(result, &ds)
				}
			}
		}
	}

	// Get all data stream names from the namespace source (if it was not defined as 'false' in the workload source)
	if secondarySource != nil {
		for labelKey, labelValue := range secondarySource.Labels {
			if strings.HasPrefix(labelKey, k8sconsts.SourceDataStreamLabelPrefix) {
				nameFromLabel := strings.TrimPrefix(labelKey, k8sconsts.SourceDataStreamLabelPrefix)
				ds := nameFromLabel

				if labelValue == "false" {
					continue
				}

				if !seen[ds] && !forbidden[ds] {
					seen[ds] = true
					result = append(result, &ds)
				}
			}
		}
	}

	return result
}

func ExtractDataStreamsFromInstrumentationConfig(ic *v1alpha1.InstrumentationConfig) []*string {
	seen := make(map[string]bool)
	result := make([]*string, 0)

	if ic != nil {
		for labelKey, labelValue := range ic.Labels {
			if strings.HasPrefix(labelKey, k8sconsts.SourceDataStreamLabelPrefix) {
				nameFromLabel := strings.TrimPrefix(labelKey, k8sconsts.SourceDataStreamLabelPrefix)
				ds := nameFromLabel

				if !seen[ds] && labelValue != "false" {
					seen[ds] = true
					result = append(result, &ds)
				}
			}
		}
	}

	return result
}

func ExtractDataStreamsFromDestination(destination v1alpha1.Destination) []*string {
	seen := make(map[string]bool)
	result := make([]*string, 0)

	if destinationDataStreamsNotNull(&destination) {
		for _, name := range destination.Spec.SourceSelector.DataStreams {
			ds := name

			if !seen[ds] {
				seen[ds] = true
				result = append(result, &ds)
			}
		}

	}

	return result
}

func destinationDataStreamsNotNull(destination *v1alpha1.Destination) bool {
	if destination.Spec.SourceSelector != nil && destination.Spec.SourceSelector.DataStreams != nil {
		return true
	}
	return false
}

func removeStreamNameFromDestination(destination *v1alpha1.Destination, dataStreamName string) {
	if destinationDataStreamsNotNull(destination) {
		// Remove the current stream name from the source selector
		destination.Spec.SourceSelector.DataStreams = RemoveStringFromSlice(destination.Spec.SourceSelector.DataStreams, dataStreamName)
	}
}

func shouldDeleteDestination(destination *v1alpha1.Destination) bool {
	if destinationDataStreamsNotNull(destination) {
		// If the source selector is not empty after removing the current stream name, we should not delete the destination
		return len(destination.Spec.SourceSelector.DataStreams) == 0
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
			if destinationDataStreamsNotNull(&dest) && ArrayContains(dest.Spec.SourceSelector.DataStreams, currentStreamName) {
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
			if destinationDataStreamsNotNull(&dest) && ArrayContains(dest.Spec.SourceSelector.DataStreams, currentStreamName) {
				// Remove the current stream name from the source selector
				dest.Spec.SourceSelector.DataStreams = RemoveStringFromSlice(dest.Spec.SourceSelector.DataStreams, currentStreamName)

				// Add the new stream name to the source selector
				if !ArrayContains(dest.Spec.SourceSelector.DataStreams, newStreamName) {
					dest.Spec.SourceSelector.DataStreams = append(dest.Spec.SourceSelector.DataStreams, newStreamName)
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
			if strings.HasPrefix(labelKey, k8sconsts.SourceDataStreamLabelPrefix) {
				if strings.TrimPrefix(labelKey, k8sconsts.SourceDataStreamLabelPrefix) == currentStreamName {
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
				if strings.HasPrefix(labelKey, k8sconsts.SourceDataStreamLabelPrefix) {
					if strings.TrimPrefix(labelKey, k8sconsts.SourceDataStreamLabelPrefix) == currentStreamName {
						// Note: we have to add a label 1st, then remove the old one, to avoid issues with source webhook.
						// The source webhook will re-add a default label, if the source has no labels at all.
						// So when trying to remove the default label, it would add itself back before we even get to apply the new label.

						// add the new label
						newDataStreamLabelKey := k8sconsts.SourceDataStreamLabelPrefix + newStreamName
						_, err := UpdateSourceCRDLabel(ctx, source.Namespace, source.Name, newDataStreamLabelKey, labelValue)
						if err != nil {
							return fmt.Errorf("failed to update source %s: %v", source.Name, err)
						}

						// remove the old label
						oldDataStreamLabelKey := k8sconsts.SourceDataStreamLabelPrefix + currentStreamName
						_, err = UpdateSourceCRDLabel(ctx, source.Namespace, source.Name, oldDataStreamLabelKey, "")
						if err != nil {
							return fmt.Errorf("failed to update source %s: %v", source.Name, err)
						}

						return nil
					}
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
