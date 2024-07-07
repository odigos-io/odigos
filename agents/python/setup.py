from setuptools import setup, find_packages

setup(
    name="odigos-python-configurator",
    version="0.1.0",
    description="My custom OpenTelemetry configurator",
    author="Tamir David",
    author_email="tamir@odigos.io",
    packages=find_packages(include=["configurator", "configurator.*"]),
    install_requires=[
        "opentelemetry-api == 1.24.0",
        "opentelemetry-semantic-conventions == 0.45b0",
        "typing-extensions >= 3.7.4",
    ],
    python_requires=">=3.8",
    entry_points={
        'opentelemetry_configurator': [
            'odigos-python-configurator = configurator:OdigosPythonConfigurator'
        ],
    },
)