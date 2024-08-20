from setuptools import setup, find_packages

setup(
    name="odigos-python-configurator",
    version="0.1.0",
    description="Odigos Configurator for Python OpenTelemetry Auto-Instrumentation",
    author="Tamir David",
    author_email="tamir@odigos.io",
    packages=find_packages(include=["configurator", "configurator.*"]),
    install_requires=[
        "odigos-opentelemetry-python"
    ],
    python_requires=">=3.8",
    entry_points={
        'opentelemetry_configurator': [
            'odigos-python-configurator = configurator:OdigosPythonConfigurator'
        ],
    },
)