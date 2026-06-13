package tracecorrelations

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	prommodel "github.com/prometheus/common/model"

	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/graph/model"
)

const (
	metricNameConnectionTotal = "traces_service_io_connection_total"
	metricSelector            = `{__name__=~"traces_service_io_connection_total(_total)?"}`
)

type workloadKey struct {
	namespace string
	kind      string
	name      string
	container string
}

type attributeGroup struct {
	attrs map[string]string
	sig   string
}

type aggregatedSeries struct {
	workload       workloadKey
	input          attributeGroup
	output         attributeGroup
	connectionCount int64
	firstDetected  time.Time
}

// GetTraceCorrelations reads service I/O connection metrics from the trace correlations
// VictoriaMetrics store and groups them by workload, input attributes, and output attributes.
func GetTraceCorrelations(ctx context.Context, api v1.API, metricsStoreURL string, filter *model.WorkloadFilter) (*model.TraceCorrelations, error) {
	if api == nil {
		return nil, fmt.Errorf("trace correlations metrics store not available")
	}
	if metricsStoreURL == "" {
		return nil, fmt.Errorf("trace correlations metrics store URL is empty")
	}

	now := time.Now()
	counts, err := queryInstantVector(ctx, api, metricSelector, now)
	if err != nil {
		return nil, fmt.Errorf("query trace correlation connection counts: %w", err)
	}

	firstSeen, err := queryFirstSeenFromExport(ctx, metricsStoreURL, now.Add(-exportLookback))
	if err != nil {
		return nil, fmt.Errorf("query trace correlation first-seen timestamps: %w", err)
	}

	aggregated := aggregateSeries(counts, firstSeen, filter)
	return buildResponse(aggregated), nil
}

func queryInstantVector(ctx context.Context, api v1.API, query string, ts time.Time) (prommodel.Vector, error) {
	val, _, err := api.Query(ctx, query, ts)
	if err != nil {
		return nil, err
	}
	vec, ok := val.(prommodel.Vector)
	if !ok {
		return prommodel.Vector{}, nil
	}
	return vec, nil
}

func aggregateSeries(counts prommodel.Vector, firstSeenByLabels map[string]time.Time, filter *model.WorkloadFilter) map[workloadKey]map[string]*aggregatedSeries {
	result := make(map[workloadKey]map[string]*aggregatedSeries)

	for _, sample := range counts {
		labels := sample.Metric
		workload, ok := workloadFromMetric(labels)
		if !ok || !matchesFilter(workload, filter) {
			continue
		}

		input := attributeGroupFromMetric(labels, inputAttributePrefix)
		output := attributeGroupFromMetric(labels, outputAttributePrefix)
		seriesKey := input.sig + "\x00" + output.sig

		byOutput, ok := result[workload]
		if !ok {
			byOutput = make(map[string]*aggregatedSeries)
			result[workload] = byOutput
		}

		series, exists := byOutput[seriesKey]
		if !exists {
			series = &aggregatedSeries{
				workload: workload,
				input:    input,
				output:   output,
			}
			byOutput[seriesKey] = series
		}

		series.connectionCount += int64(sample.Value)

		if firstDetected, ok := firstSeenByLabels[labels.String()]; ok {
			if series.firstDetected.IsZero() || firstDetected.Before(series.firstDetected) {
				series.firstDetected = firstDetected
			}
		}
	}

	return result
}

func buildResponse(aggregated map[workloadKey]map[string]*aggregatedSeries) *model.TraceCorrelations {
	workloads := make([]*model.TraceCorrelationsWorkload, 0, len(aggregated))

	for workload, seriesByKey := range aggregated {
		inputGroups := groupByInput(seriesByKey)
		workloads = append(workloads, &model.TraceCorrelationsWorkload{
			Namespace:     workload.namespace,
			Kind:          kindToModel(workload.kind),
			Name:          workload.name,
			ContainerName: workload.container,
			Inputs:        inputGroups,
		})
	}

	sort.Slice(workloads, func(i, j int) bool {
		if workloads[i].Namespace != workloads[j].Namespace {
			return workloads[i].Namespace < workloads[j].Namespace
		}
		if workloads[i].Kind != workloads[j].Kind {
			return workloads[i].Kind < workloads[j].Kind
		}
		if workloads[i].Name != workloads[j].Name {
			return workloads[i].Name < workloads[j].Name
		}
		return workloads[i].ContainerName < workloads[j].ContainerName
	})

	return &model.TraceCorrelations{Workloads: workloads}
}

func groupByInput(seriesByKey map[string]*aggregatedSeries) []*model.TraceCorrelationsInputGroup {
	inputMap := make(map[string]*model.TraceCorrelationsInputGroup)

	for _, series := range seriesByKey {
		group, ok := inputMap[series.input.sig]
		if !ok {
			group = &model.TraceCorrelationsInputGroup{
				Attributes: toModelAttributes(series.input.attrs),
				Outputs:    make([]*model.TraceCorrelationsOutputSeries, 0),
			}
			inputMap[series.input.sig] = group
		}

		group.Outputs = append(group.Outputs, &model.TraceCorrelationsOutputSeries{
			Attributes:      toModelAttributes(series.output.attrs),
			ConnectionCount: int(series.connectionCount),
			FirstDetectedAt: formatFirstDetectedAt(series.firstDetected),
		})
	}

	groups := make([]*model.TraceCorrelationsInputGroup, 0, len(inputMap))
	for _, group := range inputMap {
		sort.Slice(group.Outputs, func(i, j int) bool {
			return attributeSignatureFromModel(group.Outputs[i].Attributes) < attributeSignatureFromModel(group.Outputs[j].Attributes)
		})
		groups = append(groups, group)
	}

	sort.Slice(groups, func(i, j int) bool {
		return attributeSignatureFromModel(groups[i].Attributes) < attributeSignatureFromModel(groups[j].Attributes)
	})

	return groups
}

func toModelAttributes(attrs map[string]string) []*model.NonIdentifyingAttribute {
	if len(attrs) == 0 {
		return []*model.NonIdentifyingAttribute{}
	}
	keys := make([]string, 0, len(attrs))
	for key := range attrs {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	out := make([]*model.NonIdentifyingAttribute, 0, len(keys))
	for _, key := range keys {
		out = append(out, &model.NonIdentifyingAttribute{
			Key:   key,
			Value: attrs[key],
		})
	}
	return out
}

func attributeSignatureFromModel(attrs []*model.NonIdentifyingAttribute) string {
	if len(attrs) == 0 {
		return ""
	}
	parts := make([]string, 0, len(attrs))
	for _, attr := range attrs {
		parts = append(parts, attr.Key+"="+attr.Value)
	}
	sort.Strings(parts)
	return strings.Join(parts, "\x00")
}

func kindToModel(kind string) model.K8sResourceKind {
	switch strings.ToLower(kind) {
	case "deployment":
		return model.K8sResourceKindDeployment
	case "statefulset":
		return model.K8sResourceKindStatefulSet
	case "daemonset":
		return model.K8sResourceKindDaemonSet
	case "cronjob":
		return model.K8sResourceKindCronJob
	case "job":
		return model.K8sResourceKindJob
	case "deploymentconfig":
		return model.K8sResourceKindDeploymentConfig
	case "rollout":
		return model.K8sResourceKindRollout
	case "staticpod":
		return model.K8sResourceKindStaticPod
	case "pod":
		return model.K8sResourceKindPod
	default:
		return model.K8sResourceKind("")
	}
}

func matchesFilter(workload workloadKey, filter *model.WorkloadFilter) bool {
	if filter == nil {
		return true
	}
	if filter.Namespace != nil && *filter.Namespace != workload.namespace {
		return false
	}
	if filter.Kind != nil && !strings.EqualFold(string(*filter.Kind), workload.kind) {
		return false
	}
	if filter.Name != nil && *filter.Name != workload.name {
		return false
	}
	return true
}

// MetricsStoreURL returns the in-cluster URL for the trace correlations VictoriaMetrics store.
func MetricsStoreURL(namespace string) string {
	return fmt.Sprintf("http://%s.%s.svc:8428", consts.TraceCorrelationsMetricsServiceName, namespace)
}
