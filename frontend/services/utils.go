package services

import (
	"context"
	"errors"
	"fmt"
	"path"
	"slices"
	"sort"
	"time"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"github.com/odigos-io/odigos/k8sutils/pkg/feature"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"

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

func TransformConditionStatus(condStatus metav1.ConditionStatus, condType string, condReason string) model.ConditionStatus {
	var status model.ConditionStatus

	switch condStatus {
	case metav1.ConditionUnknown:
		status = model.ConditionStatusLoading
	case metav1.ConditionTrue:
		status = model.ConditionStatusSuccess
	case metav1.ConditionFalse:
		status = model.ConditionStatusError
	}

	// force "disabled" status ovverrides for certain "reasons"
	if v1alpha1.IsReasonStatusDisabled(condReason) {
		status = model.ConditionStatusDisabled
	}

	return status
}

func SortConditions(conditions []*model.Condition) {
	sort.Slice(conditions, func(i, j int) bool {
		if conditions[i].LastTransitionTime == nil {
			return false
		}
		if conditions[j].LastTransitionTime == nil {
			return true
		}

		timeI, errI := time.Parse(time.RFC3339, *conditions[i].LastTransitionTime)
		timeJ, errJ := time.Parse(time.RFC3339, *conditions[j].LastTransitionTime)

		if errI != nil {
			return false
		}
		if errJ != nil {
			return true
		}

		return timeI.Before(timeJ)
	})
}

// generated names can cause conflicts with k8s < 1.32.
// the best practice is to retry the create operation if we got a conflict error (409).
// this function takes care of checking the k8s version and retrying the create operation if needed.
func CreateResourceWithGenerateName[T any](ctx context.Context, createFunc func() (T, error)) (T, error) {
	if feature.RetryGenerateName(feature.GA) {
		// in k8s 1.32+, the generate name is enabled by default in the api server, so we don't need to retry.
		return createFunc()
	} else {
		var result T
		err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
			var err error
			result, err = createFunc()
			return err
		})
		return result, err
	}
}
