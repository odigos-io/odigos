# Collector Metrics

The goal of this package is to supply endpoints for querying the different collectors used by Odigos to better understand the state of the different components.
The Opentelemetry Collector provides [internal telemetry](https://opentelemetry.io/docs/collector/internal-telemetry/) which we are using for that purpose.

## Odigostrafficmetrics processor
Since not all the metrics we are interested in are provided by the collector, this custom processor allows adding our own metrics (by using the collector meterProvider we can add our metrics to the `/metrics` endpoint the collector exposed).

## Node collectors and sources metrics
In order to calculate the amount of data exported by each source, it is preferred to measure as close as possible in the pipeline to the source. That is the reason for using the node collectors for gathering information about sources.
Each node collector has a configured pipeline (which is not connected to the apps pipelines)

