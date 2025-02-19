package sources_utils

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/common/consts"

	cmdcontext "github.com/odigos-io/odigos/cli/pkg/cmd_context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

type SourceStatus struct {
	Namespace string
	Name      string
	Condition string
	Reason    string
	Message   string
	IsError   bool
}

func SourcesStatus(ctx context.Context) ([]SourceStatus, error) {
	var allStatuses []SourceStatus
	client := cmdcontext.KubeClientFromContextOrExit(ctx)
	instrumentationConfigs, err := client.OdigosClient.InstrumentationConfigs("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list InstrumentationConfigs: %w", err)
	}

	for _, instruConfig := range instrumentationConfigs.Items {
		for _, condition := range instruConfig.Status.Conditions {
			statusEntry := SourceStatus{
				Namespace: instruConfig.Namespace,
				Name:      instruConfig.Name,
				Condition: condition.Type,
				Reason:    condition.Reason,
				Message:   condition.Message,
				IsError:   condition.Status == "False",
			}
			allStatuses = append(allStatuses, statusEntry)
		}

		labelSelector := fmt.Sprintf("%s=%s", consts.InstrumentedAppNameLabel, instruConfig.Name)
		instancesList, err := client.OdigosClient.InstrumentationInstances(instruConfig.Namespace).List(ctx, metav1.ListOptions{
			LabelSelector: labelSelector,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list InstrumentationInstances for %s: %w", instruConfig.Name, err)
		}

		totalInstances := len(instancesList.Items)
		if totalInstances == 0 {
			continue
		}

		healthyInstances := 0
		for _, instance := range instancesList.Items {
			if instance.Status.Healthy != nil && *instance.Status.Healthy {
				healthyInstances++
			} else {
				statusEntry := SourceStatus{
					Namespace: instance.Namespace,
					Name:      instance.Name,
					Condition: "HealthyInstrumentationInstances",
					Reason:    "UnhealthyInstance",
					Message:   fmt.Sprintf("Only %d/%d instances are healthy", healthyInstances, totalInstances),
					IsError:   true,
				}
				allStatuses = append(allStatuses, statusEntry)
			}
		}
	}

	return allStatuses, nil
}
