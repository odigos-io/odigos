from setuptools import setup, find_packages

setup(
    name="odigos-python-configurator",
    version="0.1.0",
    description="Odigos Configurator for Python OpenTelemetry Auto-Instrumentation",
    author="Tamir David",
    author_email="tamir@odigos.io",
    packages=find_packages(include=["configurator", "configurator.*", "opamp", "opamp.*"]),
    install_requires=[
        "typing-extensions >= 3.7.4",
        "requests == 2.31.0",
        "protobuf == 4.23.4",
        "retry == 0.9.2",
        "uuid7 == 0.1.0"
    ],
    python_requires=">=3.8",
    entry_points={
        'opentelemetry_configurator': [
            'odigos-python-configurator = configurator:OdigosPythonConfigurator'
        ],
    },
)