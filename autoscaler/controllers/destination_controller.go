/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"

	controllerconfig "github.com/odigos-io/odigos/autoscaler/controllers/controller_config"
	"github.com/odigos-io/odigos/autoscaler/controllers/gateway"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	v1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
)

type DestinationReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	ImagePullSecrets []string
	OdigosVersion    string
	Config           *controllerconfig.ControllerConfig
}

func (r *DestinationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.V(0).Info("Reconciling Destination")

	var destination v1.Destination
	if err := r.Client.Get(ctx, req.NamespacedName, &destination); err != nil {
		logger.Error(err, "Failed to get Destination")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var sources v1.SourceList
	if err := r.Client.List(ctx, &sources); err != nil {
		logger.Error(err, "Failed to list Sources")
		return ctrl.Result{}, err
	}
	logger.V(1).Info("Sources", "sources", sources.Items)
	logger.V(1).Info("Destination", "destination", destination)
	logger.V(1).Info("SourceSelector", "sourceSelector", destination.Spec.SourceSelector)
	filteredSources := filterSources(sources.Items, destination.Spec.SourceSelector)
	logger.V(1).Info("Filtered Sources", "filteredSources", filteredSources)
	// Generate route configuration
	err := generateRouteConfig(ctx, r.Client, destination, filteredSources)
	if err != nil {
		logger.Error(err, "Failed to generate route configuration")
		return ctrl.Result{}, err
	}

	err = gateway.Sync(ctx, r.Client, r.Scheme, r.ImagePullSecrets, r.OdigosVersion, r.Config)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DestinationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Destination{}).
		// auto scaler only cares about the spec of each destination.
		// filter out events on resource status and metadata changes.
		WithEventFilter(&predicate.GenerationChangedPredicate{}).
		Complete(r)
}

func generateRouteConfig(ctx context.Context, client client.Client, destination v1.Destination, sources []v1.Source) error {
	// Build route configuration based on filtered sources and destination signals
	routeConfig := buildRouteConfig(destination, sources)

	// Apply the route configuration to the OpenTelemetry collector
	// This could involve updating a ConfigMap or another custom resource
	// that the collector watches
	configMapName := "otelcol-route-config"
	configMapNamespace := "odigos-system" // Replace with your namespace
	err := updateConfigMap(ctx, client, configMapName, configMapNamespace, routeConfig)
	if err != nil {
		return err
	}

	return nil
}

func buildRouteConfig(destination v1.Destination, sources []v1.Source) map[string]interface{} {
	// Example structure of the route configuration
	routeConfig := map[string]interface{}{
		"destination": destination.Spec.DestinationName,
		"signals":     destination.Spec.Signals,
		"sources":     []string{},
	}

	for _, source := range sources {
		routeConfig["sources"] = append(routeConfig["sources"].([]string), source.Name)
	}

	return routeConfig
}

func updateConfigMap(ctx context.Context, client client.Client, name, namespace string, data map[string]interface{}) error {

	// Update the ConfigMap with the new data
	// This could involve creating a new ConfigMap or updating an existing one
	// based on the name and namespace provided
	return nil
}

func filterSources(sources []v1.Source, selector *v1.SourceSelector) []v1.Source {
	if selector == nil || selector.Mode == "all" {
		// Return all sources if selector is nil or mode is "all"
		return sources
	}

	var filtered []v1.Source
	for _, source := range sources {
		switch selector.Mode {
		case "namespaces":
			for _, ns := range selector.Namespaces {
				if source.Spec.Workload.Namespace == ns {
					filtered = append(filtered, source)
					break
				}
			}
		case "groups":
			for _, group := range selector.Groups {
				for _, srcGroup := range source.Spec.Groups {
					if group == srcGroup {
						filtered = append(filtered, source)
						break
					}
				}
			}
		}
	}

	return filtered
}
