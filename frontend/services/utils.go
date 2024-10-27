package services

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const cdnUrl = "https://d15jtxgb40qetw.cloudfront.net"

func GetImageURL(image string) string {
	return path.Join(cdnUrl, image)
}

func setWorkloadInstrumentationLabel(ctx context.Context, nsName string, workloadName string, workloadKind WorkloadKind, enabled *bool) error {
	jsonMergePatchData := GetJsonMergePatchForInstrumentationLabel(enabled)

	switch workloadKind {
	case WorkloadKindDeployment:
		_, err := kube.DefaultClient.AppsV1().Deployments(nsName).Patch(ctx, workloadName, types.MergePatchType, jsonMergePatchData, metav1.PatchOptions{})
		return err
	case WorkloadKindStatefulSet:
		_, err := kube.DefaultClient.AppsV1().StatefulSets(nsName).Patch(ctx, workloadName, types.MergePatchType, jsonMergePatchData, metav1.PatchOptions{})
		return err
	case WorkloadKindDaemonSet:
		_, err := kube.DefaultClient.AppsV1().DaemonSets(nsName).Patch(ctx, workloadName, types.MergePatchType, jsonMergePatchData, metav1.PatchOptions{})
		return err
	default:
		return errors.New("unsupported workload kind " + string(workloadKind))
	}
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
