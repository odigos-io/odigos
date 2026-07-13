package serviceioconnector

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/cespare/xxhash/v2"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

const (
	attributeHashSeparator = byte(0)
	defaultMetricSeriesTTL = 15 * time.Minute
	defaultMaxMetricSeries = 10000
)

type metricSeries struct {
	dimensions pcommon.Map
	resource   pcommon.Map
	count      int64
	updatedAt  time.Time
}

// hashAttributes returns a deterministic hash of sorted metric attribute names and values.
func hashAttributes(attrs pcommon.Map) uint64 {
	names := make([]string, 0, attrs.Len())
	attrs.Range(func(name string, _ pcommon.Value) bool {
		names = append(names, name)
		return true
	})
	sort.Strings(names)

	h := xxhash.New()
	sep := []byte{attributeHashSeparator}
	for _, name := range names {
		value, _ := attrs.Get(name)
		strValue, ok := attributeValueAsString(value)
		if !ok {
			continue
		}
		_, _ = h.WriteString(name)
		_, _ = h.Write(sep)
		_, _ = h.WriteString(strValue)
	}
	return h.Sum64()
}

func (c *serviceioConnector) nowWithOffset() time.Time {
	return time.Now().Add(-c.config.MetricsTimestampOffset)
}

func (c *serviceioConnector) buildMetrics() (pmetric.Metrics, error) {
	m := pmetric.NewMetrics()

	c.seriesMutex.Lock()
	defer c.seriesMutex.Unlock()

	c.pruneStaleSeriesLocked(time.Now())
	if len(c.keyToMetric) == 0 {
		return m, nil
	}

	type resourceGroup struct {
		resource   pcommon.Map
		seriesList []metricSeries
	}

	grouped := make(map[uint64]*resourceGroup)
	for _, series := range c.keyToMetric {
		resourceKey := hashAttributes(series.resource)
		group, ok := grouped[resourceKey]
		if !ok {
			group = &resourceGroup{resource: series.resource}
			grouped[resourceKey] = group
		}
		group.seriesList = append(group.seriesList, series)
	}

	for _, group := range grouped {
		rm := m.ResourceMetrics().AppendEmpty()
		group.resource.CopyTo(rm.Resource().Attributes())

		sm := rm.ScopeMetrics().AppendEmpty()
		sm.Scope().SetName(metricScopeName())

		mCount := sm.Metrics().AppendEmpty()
		mCount.SetName(metricNameConnectionTotal)
		mCount.SetEmptySum().SetIsMonotonic(true)
		mCount.Sum().SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)

		for _, series := range group.seriesList {
			dp := mCount.Sum().DataPoints().AppendEmpty()
			dp.SetStartTimestamp(pcommon.NewTimestampFromTime(c.startTime))
			dp.SetTimestamp(pcommon.NewTimestampFromTime(c.nowWithOffset()))
			dp.SetIntValue(series.count)
			series.dimensions.CopyTo(dp.Attributes())
			dp.Attributes().PutStr(collectorInstanceAttributeId, c.collectorInstanceID)
		}
	}

	return m, nil
}

func (c *serviceioConnector) pruneStaleSeriesLocked(now time.Time) {
	for key, series := range c.keyToMetric {
		if series.updatedAt.IsZero() {
			continue
		}
		if series.updatedAt.Add(defaultMetricSeriesTTL).Before(now) {
			delete(c.keyToMetric, key)
		}
	}
}

func (c *serviceioConnector) flushMetrics(ctx context.Context) error {
	md, err := c.buildMetrics()
	if err != nil {
		return fmt.Errorf("failed to build metrics: %w", err)
	}

	if md.MetricCount() == 0 {
		return nil
	}

	return c.metricsConsumer.ConsumeMetrics(ctx, md)
}
