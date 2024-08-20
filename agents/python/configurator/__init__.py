import opentelemetry.sdk._configuration as sdk_config
from initializer.components import _initialize_components


MINIMUM_PYTHON_SUPPORTED_VERSION = (3, 8)

class OdigosPythonConfigurator(sdk_config._BaseConfigurator):
    
    def _configure(self, **kwargs):
        trace_exporters, metric_exporters, log_exporters = sdk_config._import_exporters(
            sdk_config._get_exporter_names("traces"),
            sdk_config._get_exporter_names("metrics"),
            sdk_config._get_exporter_names("logs"),
        )
        _initialize_components(trace_exporters, metric_exporters, log_exporters)