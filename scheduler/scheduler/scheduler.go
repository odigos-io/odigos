package scheduler

import (
	"context"
	v1 "github.com/keyval-dev/odigos/api/v1"
	"github.com/keyval-dev/odigos/common/utils"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ApplicationsToCollectors assign InstrumentedApplications to ready collectors
func ApplicationsToCollectors(ctx context.Context, c client.Client) error {
	logger := log.FromContext(ctx)

	// Get all applications
	var appList v1.InstrumentedApplicationList
	err := c.List(ctx, &appList)
	if err != nil {
		logger.Error(err, "scheduling failed, could not list applications")
		return err
	}
	var langDetectedApps []v1.InstrumentedApplication
	for _, app := range appList.Items {
		if len(app.Spec.Languages) > 0 {
			langDetectedApps = append(langDetectedApps, app)
		}
	}

	if len(langDetectedApps) == 0 {
		logger.V(0).Info("no available applications, skipping scheduling")
		return nil
	}

	// Get all ready collectors
	var collectorsList v1.CollectorList
	err = c.List(ctx, &collectorsList, client.InNamespace(utils.GetCurrentNamespace()))
	if err != nil {
		logger.Error(err, "scheduling failed, could not list collectors")
		return err
	}

	readyCollectors := make(map[string]v1.Collector)
	for _, col := range collectorsList.Items {
		if col.Status.Ready {
			readyCollectors[col.Name] = col
		}
	}

	if len(readyCollectors) == 0 {
		logger.V(0).Info("no available collectors, skipping scheduling")
		return nil
	}

	for _, app := range langDetectedApps {
		if app.Spec.CollectorAddr == "" {
			// Schedule to available collector
			err = scheduleApp(&app, readyCollectors, ctx, c)
			if err != nil {
				logger.Error(err, "could not schedule app to collector", "app", app.Name)
			}
		} else {
			// Validate that collector still exists
			_, collectorExists := readyCollectors[app.Spec.CollectorAddr]
			if !collectorExists {
				err = scheduleApp(&app, readyCollectors, ctx, c)
				if err != nil {
					logger.Error(err, "could not schedule app to collector", "app", app.Name)
				}
			}
		}
	}

	logger.V(0).Info("scheduling finished")
	return nil
}

func scheduleApp(app *v1.InstrumentedApplication, collectors map[string]v1.Collector, ctx context.Context, c client.Client) error {
	for _, collector := range collectors {
		app.Spec.CollectorAddr = collector.Name
		break
	}

	err := c.Update(ctx, app)
	if !apierrors.IsConflict(err) {
		return err
	}

	return nil
}
