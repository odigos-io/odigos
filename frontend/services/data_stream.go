package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/frontend/graph/model"
)

func ExtractDataStreamsFromEntities(sources []v1alpha1.Source, destinations []v1alpha1.Destination) []*model.DataStream {
	var dataStreams []*model.DataStream
	dataStreams = append(dataStreams, &model.DataStream{Name: "default"})

	// Collect stream names without duplicates
	seen := make(map[string]bool)
	seen["default"] = true

	for _, src := range sources {
		var sourceStreamNames []string
		for labelKey, labelValue := range src.Labels {
			if strings.Contains(labelKey, k8sconsts.SourceGroupLabelPrefix) && labelValue == "true" {
				sourceStreamNames = append(sourceStreamNames, strings.TrimPrefix(labelKey, k8sconsts.SourceGroupLabelPrefix))
			}
		}

		for _, streamName := range sourceStreamNames {
			if _, exists := seen[streamName]; !exists {
				seen[streamName] = true
				dataStreams = append(dataStreams, &model.DataStream{
					Name: streamName,
				})
			}
		}
	}

	for _, dest := range destinations {
		if dest.Spec.SourceSelector != nil && dest.Spec.SourceSelector.Groups != nil {
			for _, streamName := range dest.Spec.SourceSelector.Groups {
				if _, exists := seen[streamName]; !exists {
					seen[streamName] = true
					dataStreams = append(dataStreams, &model.DataStream{
						Name: streamName,
					})
				}
			}
		}
	}

	return dataStreams
}

func ExtractDataStreamsFromSource(workloadSource *v1alpha1.Source, namespaceSource *v1alpha1.Source) []*string {
	seen := make(map[string]bool)
	forbiddenNames := make(map[string]bool)
	dataStreamNames := make([]*string, 0)

	// Get all data stream names from the workload source
	if workloadSource != nil {
		for labelKey, labelValue := range workloadSource.Labels {
			if strings.Contains(labelKey, k8sconsts.SourceGroupLabelPrefix) {
				dsName := strings.TrimPrefix(labelKey, k8sconsts.SourceGroupLabelPrefix)

				if labelValue == "false" {
					forbiddenNames[dsName] = true
				}

				if _, exists := seen[dsName]; !exists && !forbiddenNames[dsName] {
					seen[dsName] = true
					dataStreamNames = append(dataStreamNames, &dsName)
				}
			}
		}
	}

	// Get all data stream names from the namespace source (if it was not defined as 'false' in the workload source)
	if namespaceSource != nil {
		for labelKey, labelValue := range namespaceSource.Labels {
			if strings.Contains(labelKey, k8sconsts.SourceGroupLabelPrefix) && labelValue == "true" {
				dsName := strings.TrimPrefix(labelKey, k8sconsts.SourceGroupLabelPrefix)

				if _, exists := seen[dsName]; !exists && !forbiddenNames[dsName] {
					seen[dsName] = true
					dataStreamNames = append(dataStreamNames, &dsName)
				}
			}
		}
	}

	return dataStreamNames
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
	err := WithGoRoutine(ctx, len(destinations.Items), func(goFunc func(func() error)) {
		for _, dest := range destinations.Items {
			dest := dest // capture range variable

			goFunc(func() error {
				if destinationGroupsNotNull(&dest) && ArrayContains(dest.Spec.SourceSelector.Groups, currentStreamName) {
					err := DeleteDestinationOrRemoveStreamName(ctx, &dest, currentStreamName)
					if err != nil {
						return fmt.Errorf("failed to delete destination or remove stream name: %v", err)
					}
				}
				return nil
			})
		}
	})

	if err != nil {
		return err
	}

	return nil
}

func UpdateDestinationsCurrentStreamName(ctx context.Context, destinations *v1alpha1.DestinationList, currentStreamName string, newStreamName string) error {
	err := WithGoRoutine(ctx, len(destinations.Items), func(goFunc func(func() error)) {
		for _, dest := range destinations.Items {
			dest := dest // capture range variable

			goFunc(func() error {
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
	})

	if err != nil {
		return err
	}

	return nil
}

func DeleteSourcesOrRemoveStreamName(ctx context.Context, sources *v1alpha1.SourceList, currentStreamName string) error {
	err := WithGoRoutine(ctx, len(sources.Items), func(goFunc func(func() error)) {
		for _, source := range sources.Items {
			source := source // capture range variable

			goFunc(func() error {
				for labelKey, labelValue := range source.Labels {
					if strings.TrimPrefix(labelKey, k8sconsts.SourceGroupLabelPrefix) == currentStreamName && labelValue == "true" {
						toPersist := []model.PersistNamespaceSourceInput{{
							Name:              source.Spec.Workload.Name,
							Kind:              model.K8sResourceKind(source.Spec.Workload.Kind),
							Selected:          false, // to remove label, or delete entirely
							CurrentStreamName: currentStreamName,
						}}

						err := SyncWorkloadsInNamespace(ctx, source.Namespace, toPersist)
						if err != nil {
							return fmt.Errorf("failed to sync workload %s: %v", source.Name, err)
						}
					}
				}
				return nil
			})
		}
	})

	if err != nil {
		return err
	}

	return nil
}

func UpdateSourcesCurrentStreamName(ctx context.Context, sources *v1alpha1.SourceList, currentStreamName string, newStreamName string) error {
	err := WithGoRoutine(ctx, len(sources.Items), func(goFunc func(func() error)) {
		for _, source := range sources.Items {
			source := source // capture range variable

			goFunc(func() error {
				for labelKey, labelValue := range source.Labels {
					if strings.TrimPrefix(labelKey, k8sconsts.SourceGroupLabelPrefix) == currentStreamName && labelValue == "true" {
						// remove the old label
						_, err := UpdateSourceCRDLabel(ctx, source.Namespace, source.Name, k8sconsts.SourceGroupLabelPrefix+currentStreamName, "false")
						if err != nil {
							return fmt.Errorf("failed to update source %s: %v", source.Name, err)
						}

						// add the new label
						_, err = UpdateSourceCRDLabel(ctx, source.Namespace, source.Name, k8sconsts.SourceGroupLabelPrefix+newStreamName, "true")
						if err != nil {
							return fmt.Errorf("failed to update source %s: %v", source.Name, err)
						}

						return nil
					}
				}
				return nil
			})
		}
	})

	if err != nil {
		return err
	}

	return nil
}
