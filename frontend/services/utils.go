package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"path"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/distribution/reference"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"github.com/odigos-io/odigos/k8sutils/pkg/feature"
	"golang.org/x/sync/errgroup"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/version"
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

	configMap, err := kube.DefaultClient.CoreV1().ConfigMaps(odigosNs).Get(ctx, consts.OdigosEffectiveConfigName, metav1.GetOptions{})
	if err != nil {
		return defaultValue, err
	}

	var odigosConfiguration common.OdigosConfiguration
	err = yaml.Unmarshal([]byte(configMap.Data[consts.OdigosConfigurationFileName]), &odigosConfiguration)
	if err != nil {
		return defaultValue, err
	}

	configValue := odigosConfiguration.UiPaginationLimit
	if configValue > 0 {
		return configValue, nil
	}

	return defaultValue, nil
}

func ConvertConditions(conditions []metav1.Condition) []*model.Condition {
	var result []*model.Condition
	for _, c := range conditions {
		if c.Type != "AppliedInstrumentationDevice" {
			reason := c.Reason
			message := c.Message
			if message == "" {
				message = string(c.Reason)
			}

			result = append(result, &model.Condition{
				Status:             TransformConditionStatus(c.Status, c.Type, reason),
				Type:               c.Type,
				Reason:             &reason,
				Message:            &message,
				LastTransitionTime: k8sLastTransitionTimeToGql(c.LastTransitionTime),
			})
		}
	}
	return result
}

func ConvertSignals(signals []model.SignalType) ([]common.ObservabilitySignal, error) {
	var result []common.ObservabilitySignal
	seen := make(map[common.ObservabilitySignal]bool)

	for _, s := range signals {
		var signal common.ObservabilitySignal
		switch s {
		case model.SignalTypeTraces:
			signal = common.TracesObservabilitySignal
		case model.SignalTypeMetrics:
			signal = common.MetricsObservabilitySignal
		case model.SignalTypeLogs:
			signal = common.LogsObservabilitySignal
		default:
			return nil, fmt.Errorf("unknown signal type: %v", s)
		}

		// Deduplicate: only add if not already seen
		if !seen[signal] {
			seen[signal] = true
			result = append(result, signal)
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

func randString(nByte int) (string, error) {
	b := make([]byte, nByte)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func setCallbackCookie(w http.ResponseWriter, r *http.Request, name string, value string, path string, maxAge int) {
	if maxAge <= 0 {
		maxAge = int(time.Hour.Seconds())
	}

	c := &http.Cookie{
		Name:     name,
		Value:    value,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode, // prevents CSRF
		MaxAge:   maxAge,
	}

	if path != "" {
		c.Path = path
	}

	http.SetCookie(w, c)
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

// Function to run multiple goroutines with a limit on concurrency.
func WithGoroutine(ctx context.Context, limit int, run func(goFunc func(func() error))) error {
	g, _ := errgroup.WithContext(ctx)
	if limit > 0 {
		g.SetLimit(limit)
	}

	run(g.Go)

	if err := g.Wait(); err != nil {
		return err
	}
	return nil
}

// getKubeVersion retrieves and parses the Kubernetes server version.
func getKubeVersion() (*version.Version, error) {
	verInfo, err := kube.DefaultClient.Discovery().ServerVersion()
	if err != nil {
		return nil, err
	}

	parsedVer, err := version.ParseGeneric(verInfo.GitVersion)
	if err != nil {
		return nil, err
	}

	return parsedVer, nil
}

func k8sLastTransitionTimeToGql(t metav1.Time) *string {
	if t.IsZero() {
		return nil
	}
	str := t.UTC().Format(time.RFC3339)
	return &str
}

// ExtractImageVersion extracts the tag (version) from a full container image reference.
func ExtractImageVersion(image string) string {
	if image == "" {
		return ""
	}

	if idx := strings.Index(image, "@"); idx >= 0 {
		image = image[:idx]
	}

	ref, err := reference.ParseNormalizedNamed(image)
	if err != nil {
		return ""
	}

	if tagged, ok := ref.(reference.Tagged); ok {
		return tagged.Tag()
	}

	return ""
}
