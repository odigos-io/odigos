package sources_utils

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/common/consts"

	cmdcontext "github.com/odigos-io/odigos/cli/pkg/cmd_context"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

type SourceStatus struct {
	Namespace string
	Name      string
	Status    string
	Reason    string
	Message   string
	IsError   bool
}

func SourcesStatus(ctx context.Context) ([]SourceStatus, error) {
	var allStatuses []SourceStatus
	client := cmdcontext.KubeClientFromContextOrExit(ctx)
	instrumentationConfigs, err := client.OdigosClient.InstrumentationConfigs("").List(ctx, v1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list InstrumentationConfigs: %w", err)
	}

	for _, instruConfig := range instrumentationConfigs.Items {
		hasError := false
		errorMessages := []string{}

		for _, condition := range instruConfig.Status.Conditions {
			if condition.Status == "False" {
				hasError = true
				errorMessages = append(errorMessages, fmt.Sprintf("%s: %s", condition.Type, condition.Message))
			}
		}

		labelSelector := fmt.Sprintf("%s=%s", consts.InstrumentedAppNameLabel, instruConfig.Name)
		instancesList, err := client.OdigosClient.InstrumentationInstances(instruConfig.Namespace).List(ctx, v1.ListOptions{
			LabelSelector: labelSelector,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list InstrumentationInstances for %s: %w", instruConfig.Name, err)
		}

		healthyInstances := 0
		for _, instance := range instancesList.Items {
			if instance.Status.Healthy != nil && *instance.Status.Healthy {
				healthyInstances++
			} else {
				hasError = true
				errorMessages = append(errorMessages, fmt.Sprintf("%s is unhealthy", instance.Name))
			}
		}

		if hasError {
			allStatuses = append(allStatuses, SourceStatus{
				Namespace: instruConfig.Namespace,
				Name:      instruConfig.Name,
				Status:    "Error",
				Reason:    "",
				Message:   fmt.Sprintf("%s", errorMessages),
				IsError:   true,
			})
		} else {
			allStatuses = append(allStatuses, SourceStatus{
				Namespace: instruConfig.Namespace,
				Name:      instruConfig.Name,
				Status:    "Healthy",
				Reason:    "",
				Message:   "",
				IsError:   false,
			})
		}
	}

	return allStatuses, nil
}
