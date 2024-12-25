package services

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const cdnUrl = "https://d15jtxgb40qetw.cloudfront.net"

func GetImageURL(image string) string {
	return path.Join(cdnUrl, image)
}

func ConvertFieldsToString(fields map[string]string) string {
	if len(fields) == 0 {
		return ""
	}

	var parts []string
	for key, value := range fields {
		parts = append(parts, fmt.Sprintf("%s: %s", key, value))
	}

	return strings.Join(parts, ", ")
}

func ConvertSignals(signals []model.SignalType) ([]common.ObservabilitySignal, error) {
	var result []common.ObservabilitySignal
	for _, s := range signals {
		switch s {
		case model.SignalTypeTraces:
			result = append(result, common.TracesObservabilitySignal)
		case model.SignalTypeMetrics:
			result = append(result, common.MetricsObservabilitySignal)
		case model.SignalTypeLogs:
			result = append(result, common.LogsObservabilitySignal)
		default:
			return nil, fmt.Errorf("unknown signal type: %v", s)
		}
	}
	return result, nil
}

func DerefString(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}

func StringPtr(s string) *string {
	return &s
}

func Metav1TimeToString(latestStatusTime metav1.Time) string {
	if latestStatusTime.IsZero() {
		return ""
	}
	return latestStatusTime.Time.Format(time.RFC3339)
}

func CreateWorkloadSource(ctx context.Context, nsName string, workloadName string, workloadKind WorkloadKind) error {
	newSource := &v1alpha1.Source{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "source-",
		},
		Spec: v1alpha1.SourceSpec{
			Workload: workload.PodWorkload{
				Namespace: nsName,
				Name:      workloadName,
				Kind:      workload.WorkloadKind(workloadKind),
			},
		},
		Status: v1alpha1.SourceStatus{
			Conditions: []metav1.Condition{},
		},
	}

	switch workloadKind {
	case WorkloadKindDeployment:
	case WorkloadKindStatefulSet:
	case WorkloadKindDaemonSet:
		_, err := kube.DefaultClient.OdigosClient.Sources("").Create(ctx, newSource, metav1.CreateOptions{})
		return err
	default:
		return errors.New("unsupported workload kind " + string(workloadKind))
	}

	return nil
}

func DeleteWorkloadSource(ctx context.Context, nsName string, workloadName string, workloadKind WorkloadKind) error {
	switch workloadKind {
	case WorkloadKindDeployment:
	case WorkloadKindStatefulSet:
	case WorkloadKindDaemonSet:
		err := kube.DefaultClient.OdigosClient.Sources("").Delete(ctx, workloadName, metav1.DeleteOptions{})
		return err
	default:
		return errors.New("unsupported workload kind " + string(workloadKind))
	}

	return nil
}

func ToggleWorkloadSource(ctx context.Context, nsName string, workloadName string, workloadKind WorkloadKind, enabled *bool) error {
	if enabled == nil {
		return errors.New("enabled must be provided")
	}

	if *enabled {
		CreateWorkloadSource(ctx, nsName, workloadName, workloadKind)
	} else {
		DeleteWorkloadSource(ctx, nsName, workloadName, workloadKind)
	}

	return nil
}
