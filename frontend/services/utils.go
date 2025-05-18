package services

import (
	"context"
	"errors"
	"fmt"
	"path"
	"time"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"slices"

	"sigs.k8s.io/yaml"
)

const (
	cdnUrl = "https://d15jtxgb40qetw.cloudfront.net"
)

func GetImageURL(image string) string {
	return path.Join(cdnUrl, image)
}

func GetPageLimit(ctx context.Context) (int, error) {
	defaultValue := 100
	odigosNs := env.GetCurrentNamespace()

	configMap, err := kube.DefaultClient.CoreV1().ConfigMaps(odigosNs).Get(ctx, consts.OdigosConfigurationName, metav1.GetOptions{})
	if err != nil {
		return defaultValue, err
	}

	var odigosConfig common.OdigosConfiguration
	err = yaml.Unmarshal([]byte(configMap.Data[consts.OdigosConfigurationFileName]), &odigosConfig)
	if err != nil {
		return defaultValue, err
	}

	configValue := odigosConfig.UiPaginationLimit
	if configValue > 0 {
		return configValue, nil
	}

	return defaultValue, nil
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

func CheckWorkloadKind(kind WorkloadKind) error {
	switch kind {
	case WorkloadKindDeployment, WorkloadKindStatefulSet, WorkloadKindDaemonSet:
		return nil
	default:
		return errors.New("unsupported workload kind: " + string(kind))
	}
}

func ArrayContains(arr []string, str string) bool {
	return slices.Contains(arr, str)
}

func RemoveStringFromSlice(slice []string, target string) []string {
	result := make([]string, 0, len(slice))
	for _, s := range slice {
		if s != target {
			result = append(result, s)
		}
	}
	return result
}
