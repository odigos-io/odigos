import threading
import atexit
import sys
import os
import opentelemetry.sdk._configuration as sdk_config

from opentelemetry.sdk.resources import Resource
from opentelemetry.sdk.resources import ProcessResourceDetector, OTELResourceDetector
from .lib_handling import reorder_python_path, reload_distro_modules
from .version import VERSION
from opamp.http_client import OpAMPHTTPClient


MINIMUM_PYTHON_SUPPORTED_VERSION = (3, 8)

class OdigosPythonConfigurator(sdk_config._BaseConfigurator):
    
    def _configure(self, **kwargs):
        _initialize_components()
        
        
def _initialize_components():    
    resource_attributes_event = threading.Event()
    client = None
    
    try:
        
        client = start_opamp_client(resource_attributes_event)
        resource_attributes_event.wait(timeout=30)  # Wait for the resource attributes to be received for 30 seconds

        received_value = client.resource_attributes
        
        if received_value:    
            trace_exporters, metric_exporters, log_exporters = sdk_config._import_exporters(
                sdk_config._get_exporter_names("traces"),
                sdk_config._get_exporter_names("metrics"),
                sdk_config._get_exporter_names("logs"),
            )

            auto_resource = {
                "telemetry.distro.name": "odigos",
                "telemetry.distro.version": VERSION,
            }
                        
            auto_resource.update(received_value)

            resource = Resource.create(auto_resource) \
                .merge(OTELResourceDetector().detect()) \
                .merge(ProcessResourceDetector().detect())

            initialize_traces_if_enabled(trace_exporters, resource)
            initialize_metrics_if_enabled(metric_exporters, resource)
            initialize_logging_if_enabled(log_exporters, resource)

        # Reorder the python sys.path to ensure that the user application's dependencies take precedence over the agent's dependencies.
        # This is necessary because the user application's dependencies may be incompatible with those used by the agent.
        reorder_python_path()
        # Reload distro modules to ensure the new path is used.
        reload_distro_modules()
        
    except Exception as e:
        if client is not None:
            client.shutdown(custom_failure_message=str(e))
        

def initialize_traces_if_enabled(trace_exporters, resource):
    traces_enabled = os.getenv(sdk_config.OTEL_TRACES_EXPORTER, "none").strip().lower()
    if traces_enabled != "none":
        id_generator_name = sdk_config._get_id_generator()
        id_generator = sdk_config._import_id_generator(id_generator_name)
        sdk_config._init_tracing(exporters=trace_exporters, id_generator=id_generator, resource=resource)

def initialize_metrics_if_enabled(metric_exporters, resource):
    metrics_enabled = os.getenv(sdk_config.OTEL_METRICS_EXPORTER, "none").strip().lower()
    if metrics_enabled != "none":
        sdk_config._init_metrics(metric_exporters, resource)

def initialize_logging_if_enabled(log_exporters, resource):
    logging_enabled = os.getenv(sdk_config.OTEL_LOGS_EXPORTER, "none").strip().lower()
    if logging_enabled != "none":
        sdk_config._init_logging(log_exporters, resource)


def start_opamp_client(event):
    condition = threading.Condition(threading.Lock())
    client = OpAMPHTTPClient(event, condition)
    
    python_version_supported = is_supported_python_version()

    client.start(python_version_supported)
    
    def shutdown():
        client.shutdown()

    # Ensure that the shutdown function is called on program exit
    atexit.register(shutdown)

    return client


def is_supported_python_version():
    return sys.version_info >= MINIMUM_PYTHON_SUPPORTED_VERSION