package tracecorrelations

import (
	"errors"
	"strings"
)

// ErrMetricsStoreUnavailable is returned when the correlations VictoriaMetrics store cannot be reached.
var ErrMetricsStoreUnavailable = errors.New("trace correlations metrics store is unavailable")

func mapMetricsStoreError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrMetricsStoreUnavailable) || isMetricsStoreConnectionError(err) {
		return ErrMetricsStoreUnavailable
	}
	return err
}

func isMetricsStoreConnectionError(err error) bool {
	if err == nil {
		return false
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "no such host") ||
		strings.Contains(msg, "dial tcp") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "i/o timeout") ||
		strings.Contains(msg, "connect: network is unreachable") ||
		strings.Contains(msg, "metrics store not available") ||
		strings.Contains(msg, "metrics store url is empty")
}
