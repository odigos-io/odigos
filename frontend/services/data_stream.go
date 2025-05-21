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

func DestinationGroupsNotNull(destination *v1alpha1.Destination) bool {
	if destination.Spec.SourceSelector != nil && destination.Spec.SourceSelector.Groups != nil {
		return true
	}
	return false
}

func RemoveStreamNameFromDestination(destination *v1alpha1.Destination, dataStreamName string) {
	if DestinationGroupsNotNull(destination) {
		// Remove the current stream name from the source selector
		destination.Spec.SourceSelector.Groups = RemoveStringFromSlice(destination.Spec.SourceSelector.Groups, dataStreamName)
	}
}

func ShouldDeleteDestination(destination *v1alpha1.Destination) bool {
	if DestinationGroupsNotNull(destination) {
		// If the source selector is not empty after removing the current stream name, we should not delete the destination
		return len(destination.Spec.SourceSelector.Groups) == 0
	}
	return true
}

func DeleteDestinationOrRemoveStreamName(ctx context.Context, dest *v1alpha1.Destination, currentStreamName string) error {
	RemoveStreamNameFromDestination(dest, currentStreamName)

	if ShouldDeleteDestination(dest) {
		if err := DeleteDestinationAndSecret(ctx, dest); err != nil {
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
	var g errgroup.Group

	for _, dest := range destinations.Items {
		dest := dest // capture range variable

		g.Go(func() error {
			if DestinationGroupsNotNull(&dest) && ArrayContains(dest.Spec.SourceSelector.Groups, currentStreamName) {
				err := DeleteDestinationOrRemoveStreamName(ctx, &dest, currentStreamName)
				if err != nil {
					return fmt.Errorf("failed to delete destination or remove stream name: %v", err)
				}
			}
			return nil
		})
	}

	// wait for goroutines to complete
	if err := g.Wait(); err != nil {
		return err
	}

	return nil
}

func UpdateDestinationsCurrentStreamName(ctx context.Context, destinations *v1alpha1.DestinationList, currentStreamName string, newStreamName string) error {
	var g errgroup.Group

	for _, dest := range destinations.Items {
		dest := dest // capture range variable

		g.Go(func() error {
			if DestinationGroupsNotNull(&dest) && ArrayContains(dest.Spec.SourceSelector.Groups, dest.Name) {
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

	// wait for goroutines to complete
	if err := g.Wait(); err != nil {
		return err
	}
	return nil
}

func DeleteSourcesOrRemoveStreamName(ctx context.Context, sources *v1alpha1.SourceList, currentStreamName string) error {
	var g errgroup.Group

	for _, source := range sources.Items {
		source := source // capture range variable

		g.Go(func() error {
			for key := range source.Labels {
				if strings.TrimPrefix(key, k8sconsts.SourceGroupLabelPrefix) == currentStreamName {
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

	// wait for goroutines to complete
	if err := g.Wait(); err != nil {
		return err
	}
	return nil
}

func UpdateSourcesCurrentStreamName(ctx context.Context, sources *v1alpha1.SourceList, currentStreamName string, newStreamName string) error {
	var g errgroup.Group

	for _, source := range sources.Items {
		source := source // capture range variable

		g.Go(func() error {
			for key := range source.Labels {
				if strings.TrimPrefix(key, k8sconsts.SourceGroupLabelPrefix) == currentStreamName {
					// remove the old label
					_, err := UpdateSourceCRDLabel(ctx, source.Namespace, source.Name, k8sconsts.SourceGroupLabelPrefix+currentStreamName, "")
					if err != nil {
						return fmt.Errorf("failed to update source %s: %v", source.Name, err)
					}

					// add the new label
					_, err = UpdateSourceCRDLabel(ctx, source.Namespace, source.Name, k8sconsts.SourceGroupLabelPrefix+newStreamName, "true")
					if err != nil {
						return fmt.Errorf("failed to update source %s: %v", source.Name, err)
					}
				}
			}
			return nil
		})
	}

	// wait for goroutines to complete
	if err := g.Wait(); err != nil {
		return err
	}
	return nil
}
