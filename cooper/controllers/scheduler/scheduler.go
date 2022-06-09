package scheduler

import (
	"context"
	v1 "github.com/keyval-dev/odigos/cooper/api/v1"
	"github.com/keyval-dev/odigos/cooper/utils"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ScheduleApplicationsToCollectors assign InstrumentedApplications to ready collectors
func ScheduleApplicationsToCollectors(ctx context.Context, c client.Client) error {
	logger := log.FromContext(ctx)

	// Get all applications
	var appList v1.InstrumentedApplicationList
	err := c.List(ctx, &appList, client.InNamespace(utils.GetCurrentNamespace()))
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
	// TODO: move to collector CRD
	labels := map[string]string{
		utils.CollectorLabel: "true",
	}
	var collectorsList corev1.PodList
	err = c.List(ctx, &collectorsList, client.InNamespace(utils.GetCurrentNamespace()), client.MatchingLabels(labels))
	if err != nil {
		logger.Error(err, "scheduling failed, could not list collectors")
		return err
	}

	readyCollectorsByAddr := make(map[string]corev1.Pod)
	for _, col := range collectorsList.Items {
		if col.Status.Phase == corev1.PodRunning {
			readyCollectorsByAddr[col.Name] = col
		}
	}

	if len(readyCollectorsByAddr) == 0 {
		logger.V(0).Info("no available collectors, skipping scheduling")
		return nil
	}

	for _, app := range langDetectedApps {
		if app.Spec.CollectorAddr == "" {
			// Schedule to available collector
			err = scheduleApp(&app, readyCollectorsByAddr, ctx, c)
			if err != nil {
				logger.Error(err, "could not schedule app to collector", "app", app.Name)
			}
		} else {
			// Validate that collector still exists
			_, collectorExists := readyCollectorsByAddr[app.Spec.CollectorAddr]
			if !collectorExists {
				err = scheduleApp(&app, readyCollectorsByAddr, ctx, c)
				if err != nil {
					logger.Error(err, "could not schedule app to collector", "app", app.Name)
				}
			}
		}
	}

	logger.V(0).Info("scheduling finished")
	return nil
}

func scheduleApp(app *v1.InstrumentedApplication, collectors map[string]corev1.Pod, ctx context.Context, c client.Client) error {
	for _, collector := range collectors {
		app.Spec.CollectorAddr = collector.Name
		break
	}

	return c.Update(ctx, app)
}
