from setuptools import setup, find_packages

index_url = None

# DEV - Local Uncomment to develop locally \/
# index_url = 'http://host.docker.internal:8080/packages/odigos_opentelemetry_python-1.0.56-py3-none-any.whl'

install_requires = [
    f"odigos-opentelemetry-python @ {index_url}" if index_url else "odigos-opentelemetry-python==1.0.56"
]

setup(
    name="odigos-python-configurator",
    version="0.1.0",
    description="Odigos Configurator for Python OpenTelemetry Auto-Instrumentation",
    author="Tamir David",
    author_email="tamir@odigos.io",
    packages=find_packages(include=["configurator", "configurator.*"]),
    install_requires=install_requires,
    python_requires=">=3.8",
    entry_points={
        'opentelemetry_configurator': [
            'odigos-python-configurator = configurator:OdigosPythonConfigurator'
        ],
    },
)
