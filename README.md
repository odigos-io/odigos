# Odigos

[![Release](https://github.com/keyval-dev/odigos/actions/workflows/main.yml/badge.svg)](https://github.com/keyval-dev/odigos/actions/workflows/main.yml) [![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

Odigos is an observability control plane.

Start sending traces, metrics and logs to your favourite observability service (such as Datadog, Honeycomb, Grafana, etc) in a few clicks.

- 🧑‍💻 **No code changes** - Odigos detects the programming language of your applications and apply automatic instrumentation accordingly.
- 📖 **Open Technologies** - Odigos observabiltiy pipelines are based on the best open source observability technologies such as [OpenTelemetry](https://opentelemetry.io) and [eBPF](https://ebpf.io).
- 📝 **Easy to use** - Leverage advanced features such as tail-based sampling without editing complex YAML files.

### For full documentation and getting started guide, visit [odigos.io](https://odigos.io).

## Supported Destinations

|               | Traces | Metrics | Logs |
| ------------- | ------ | ------- | ---- |
| New Relic     | ✅     | ✅      | ✅   |
| Datadog       | ✅     | ✅      |      |
| Honeycomb     | ✅     |         |      |
| Grafana Cloud | ✅     |         |      |

Many more destinations are coming soon.

## Installation

The easiest way to install Odigos is to use our [Helm chart](https://github.com/keyval-dev/odigos-charts) by running the following commands:

```console
helm repo add odigos https://keyval-dev.github.io/odigos-charts/

helm install my-odigos odigos/odigos --namespace odigos-system --create-namespace
```

See the [quickstart guide](https://odigos.io/docs/) for more details and examples.

## Project Status

This project is actively maintained by [keyval](https://keyval.dev) and is currently in its initial days. We would love to receive your ideas, feedback & contributions.

## License

This project is licensed under the terms of the [Apache 2.0](LICENSE-Apache-2.0) open source license. Please refer to [LICENSE](LICENSE) for the full terms.

```

```
