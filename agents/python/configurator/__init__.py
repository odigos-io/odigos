# my_otel_configurator/__init__.py
import opentelemetry.sdk._configuration as sdk_config
import os
from opentelemetry.sdk.resources import Resource
from opentelemetry.sdk.resources import ProcessResourceDetector, OTELResourceDetector
from .version import VERSION


class OdigosPythonConfigurator(sdk_config._BaseConfigurator):
    def _configure(self, **kwargs):
        _initialize_components(kwargs.get("auto_instrumentation_version"))

def _initialize_components(auto_instrumentation_version):
    trace_exporters, metric_exporters, log_exporters = sdk_config._import_exporters(
        sdk_config._get_exporter_names("traces"),
        sdk_config._get_exporter_names("metrics"),
        sdk_config._get_exporter_names("logs"),
    )

    auto_resource = {
        "telemetry.distro.name": "odigos",
        "telemetry.distro.version": VERSION,
    }

    if auto_instrumentation_version:
        auto_resource[sdk_config.ResourceAttributes.TELEMETRY_AUTO_VERSION] = auto_instrumentation_version

    resource = Resource.create(auto_resource) \
        .merge(OTELResourceDetector().detect()) \
        .merge(ProcessResourceDetector().detect())

    initialize_traces_if_enabled(trace_exporters, resource)
    initialize_metrics_if_enabled(metric_exporters, resource)
    initialize_logging_if_enabled(log_exporters, resource)

def initialize_traces_if_enabled(trace_exporters, resource):
    traces_enabled = os.getenv(sdk_config.OTEL_TRACES_EXPORTER, "none").strip().lower()
    if traces_enabled != "none":
        id_generator_name = sdk_config._get_id_generator()
        id_generator = sdk_config._import_id_generator(id_generator_name)
        sdk_config._init_tracing(exporters=trace_exporters, id_generator=id_generator, resource=resource)

def initialize_metrics_if_enabled(metric_exporters, resource):
    metrics_enabled = os.getenv(sdk_config.OTEL_METRICS_EXPORTER, "none").strip().lower()
    if metrics_enabled != "none":
        sdk_config._init_metrics(metric_exporters,resource)

def initialize_logging_if_enabled(log_exporters, resource):
    logging_enabled = os.getenv(sdk_config.OTEL_LOGS_EXPORTER, "none").strip().lower()
    if logging_enabled != "none":
        sdk_config._init_logging(log_exporters, resource)