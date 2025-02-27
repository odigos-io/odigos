package services

import (
	"errors"
	"fmt"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/frontend/graph/model"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	cdnUrl                     = "https://d15jtxgb40qetw.cloudfront.net"
	ODIGOS_UI_PAGINATION_LIMIT = "ODIGOS_UI_PAGINATION_LIMIT"
)

func GetImageURL(image string) string {
	return path.Join(cdnUrl, image)
}

func GetPageLimit() int {
	defaultValue := 10
	envValue, exists := os.LookupEnv(ODIGOS_UI_PAGINATION_LIMIT)

	if exists && envValue != "" {
		envValue, err := strconv.Atoi(envValue)

		if err != nil {
			return defaultValue
		}
		return envValue
	}
	return defaultValue
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
