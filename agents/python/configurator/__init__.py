import opentelemetry.sdk._configuration as sdk_config
from initializer.components import initialize_components


MINIMUM_PYTHON_SUPPORTED_VERSION = (3, 8)

class OdigosPythonConfigurator(sdk_config._BaseConfigurator):
    def _configure(self, **kwargs):
        initialize_components(trace_exporters=True)