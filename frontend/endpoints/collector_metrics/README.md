# Collector Metrics

The goal of this package is to supply endpoints for querying the different collectors used by Odigos to better understand the state of the different components.
The OpenTelemetry Collector provides [internal telemetry](https://opentelemetry.io/docs/collector/internal-telemetry/) which we are using for that purpose.

## Odigostrafficmetrics processor
Since not all the metrics we are interested in are provided by the collector, this custom processor allows adding our own metrics (by using the collector's meterProvider we can add our metrics to the `/metrics` endpoint the collector exposes).
### Performance
Calculating extra metrics such as the total size of spans/metrics/logs data going through the processor comes with a cost.
For that reason the processor has the `sampling_ratio` configuration which configures the processor to perform the actual measurements on a fraction of the data and extrapolate the metric according to the configured ration.

## Node collectors and sources metrics
In order to calculate the amount of data exported by each source, it is preferred to measure as close as possible in the pipeline to the source. That is the reason for using the node collectors for gathering information about sources.
Each node collector has a configured pipeline (which is **not** connected to the apps pipelines) that is responsible for exporting the metrics to our desired destination (in that case the UI server pod).
This pipeline consists of:
- [prometheus scraper](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/receiver/prometheusreceiver) which scrapes the collector metrics from `/metrics`.
- OTLP exporter which exports the metrics to the UI server pod.

To aggregate the metrics by source the node collector is configured to use the `odigostrafficmetrics`. The processor can be configured with a set of resource attributes which will determine the aggregation. We use:
- `service.name`
- `k8s.namespace.name`
- `k8s.deployment.name`
- `k8s.statefulset.name`
- `k8s.daemonset.name`

## Cluster collector and destination metrics
Collecting the metrics which are related to the amount of data sent to each destination is done in the cluster collector. The actual export to the destination is performed in the cluster collector, hence it is the suitable place for this measurement.
Since the cluster collector can be configured with different processors which will drop or modify the telemetry data, it is important to collect the metrics at the end of the pipeline.
The collector builtin metrics contain `otelcol_exporter_sent_spans`, `otelcol_exporter_sent_metric_points` and `otelcol_exporter_sent_log_records` which are automatically recorded by the collector for each exporter.
These metrics record the number of spans/metric/logs sent to each exporter (destination). In order to calculate the throughput and total data sent to each destination we combine the above metrics with the metrics from `odigostrafficmetrics`. The `odigostrafficmetrics` processor is used to estimate the average size of a span/metric/log - which is then multiplied by the amount of spans/metrics/logs to produce the throughput and total data sent.

## UI server collector metrics package
This package has 2 main jobs:
- An OTLP receiver which receives the metrics from the different collectors.
- Saving an in-memory snapshot of the sources and destinations metrics which the frontend can query. The snapshot is saved as a mapping between a source/destination id to an internal map. The internal mapping is between node-collector/cluster-collector to the last snapshot. Maintaining this mapping requires a notification system to delete collector, source and destination entries once they are removed. For that we set k8s watchers for deletion of these components.

