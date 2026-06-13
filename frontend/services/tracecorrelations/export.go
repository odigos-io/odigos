package tracecorrelations

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	prommodel "github.com/prometheus/common/model"
)

const (
	exportLookback        = 25 * time.Hour
	exportRequestTimeout  = 30 * time.Second
	exportScannerBufferSz = 1024 * 1024
)

type exportRow struct {
	Metric     map[string]string `json:"metric"`
	Timestamps []int64           `json:"timestamps"`
}

// queryFirstSeenFromExport reads historical samples from VictoriaMetrics and returns
// the earliest timestamp per time series. MetricsQL tfirst() is unavailable on the
// correlations store, so export is used instead.
func queryFirstSeenFromExport(ctx context.Context, baseURL string, start time.Time) (map[string]time.Time, error) {
	exportURL, err := exportURL(baseURL, start)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, exportURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build export request: %w", err)
	}

	client := &http.Client{Timeout: exportRequestTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("export request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("export request returned %s: %s", resp.Status, string(body))
	}

	return parseExportFirstSeen(resp.Body)
}

func exportURL(baseURL string, start time.Time) (string, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("parse metrics store URL: %w", err)
	}

	query := parsed.Query()
	query.Set("match[]", metricNameConnectionTotal)
	query.Set("start", start.UTC().Format(time.RFC3339))
	parsed.RawQuery = query.Encode()
	parsed.Path = "/api/v1/export"
	return parsed.String(), nil
}

func parseExportFirstSeen(r io.Reader) (map[string]time.Time, error) {
	scanner := bufio.NewScanner(r)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, exportScannerBufferSz)

	firstSeen := make(map[string]time.Time)
	for scanner.Scan() {
		var row exportRow
		if err := json.Unmarshal(scanner.Bytes(), &row); err != nil {
			return nil, fmt.Errorf("decode export row: %w", err)
		}
		if len(row.Timestamps) == 0 {
			continue
		}

		labels := metricLabelsToPromModel(row.Metric)
		detectedAt := time.UnixMilli(row.Timestamps[0]).UTC()
		key := labels.String()

		if existing, ok := firstSeen[key]; !ok || detectedAt.Before(existing) {
			firstSeen[key] = detectedAt
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read export stream: %w", err)
	}

	return firstSeen, nil
}

func metricLabelsToPromModel(labels map[string]string) prommodel.Metric {
	out := make(prommodel.Metric, len(labels))
	for key, value := range labels {
		out[prommodel.LabelName(key)] = prommodel.LabelValue(value)
	}
	return out
}

func formatFirstDetectedAt(ts time.Time) string {
	if ts.IsZero() {
		return ""
	}
	return ts.UTC().Format(time.RFC3339)
}
