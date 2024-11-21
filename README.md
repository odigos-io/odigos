<p align="center">
    <a href="https://github.com/odigos-io/odigos/actions/workflows/release.yml" target="_blank">
        <img src="https://github.com/odigos-io/odigos/actions/workflows/release.yml/badge.svg" alt="Release Odigos CLI" style="margin-right: 10px; border: 1px solid #007acc; border-radius: 4px; padding: 5px;">
    </a>
    <a href="https://goreportcard.com/report/github.com/odigos-io/odigos/cli" target="_blank">
        <img src="https://goreportcard.com/badge/github.com/odigos-io/odigos/cli" alt="Go Report Card" style="margin-right: 10px; border: 1px solid #4CAF50; border-radius: 4px; padding: 5px;">
    </a>
    <a href="https://godoc.org/github.com/odigos-io/odigos/cli" target="_blank">
        <img src="https://godoc.org/github.com/odigos-io/odigos/cli?status.svg" alt="GoDoc" style="border: 1px solid #f39c12; border-radius: 4px; padding: 5px;">
    </a>
</p>


<p align="center">
<img src="assets/logo.png" width="350" /></br>
<h2>Generate distributed traces for any application in k8s without code changes.</h2>
</p>

<h2 align="center">
    <a href="https://www.youtube.com/watch?v=nynyV7FC4VI">Demo Video</a> • <a href="https://docs.odigos.io">Documentation</a> • <a href="https://join.slack.com/t/odigos/shared_invite/zt-1d7egaz29-Rwv2T8kyzc3mWP8qKobz~A">Join Slack Community</a>
</h2>

<<<<<<< Updated upstream
=======
## What is Odigos?

Odigos is an open-source distributed tracing solution that revolutionizes observability for Kubernetes environments. It provides instant tracing capabilities without requiring any code changes to your applications.

## Key Features

* **Code-Free Instrumentation**: Set up distributed tracing in minutes, eliminating manual code modifications.
* **Multi-Language Support**: Works with Java, Python, .NET, Node.js, and Go applications.
* **eBPF-Powered**: Utilizes eBPF technology for high-performance instrumentation of Go applications. eBPF-based instrumentation for Java, Python, .NET, and Node.js is available in the commercial edition.
* **OpenTelemetry Compatible**: Generates traces in OpenTelemetry format for broad tool compatibility.
* **Vendor Agnostic**: Integrates with various monitoring solutions, avoiding vendor lock-in.
* **Automatic Scaling**: Manages and scales OpenTelemetry collectors based on data volume.
* **User-Friendly Management**: Offers a web UI for easy configuration and management.
* **Opinionated Defaults**: Supplies common defaults and best practices out-of-the-box, requiring no deep knowledge of OpenTelemetry.

## Why Choose Odigos

1. **Simplicity**: Implement distributed tracing with minimal effort and complexity.
2. **Performance**: Separates data recording and processing to minimize runtime impact.
3. **Community-Backed**: With 3,000+ GitHub stars and growing contributor base.

Odigos empowers platform engineers, DevOps professionals, and SREs to enhance their observability strategies quickly and effectively, making it an ideal solution for modern cloud-native environments.

## Features
>>>>>>> Stashed changes

### ✨ Language Agnostic Auto-instrumentation

Odigos supports any application written in Java, Python, .NET, Node.js, and **Go**.  
Historically, compiled languages like Go have been difficult to instrument without code changes. Odigos solves this problem by uniquely leveraging [eBPF](https://ebpf.io).

![Works on any application](assets/choose_apps.png)


### 🤝 Keep your existing observability tools
Odigos currently supports all the popular managed and open-source destinations.  
By producing data in the [OpenTelemetry](https://opentelemetry.io) format, Odigos can be used with any observability tool that supports OTLP.

For a complete list of supported destinations, see [here](#supported-destinations).

![Works with any observability tool](assets/choose_dest.png)

### 🎛️ Collectors Management 
Odigos automatically scales OpenTelemetry collectors based on observability data volume.  
Manage and configure collectors via a convenient web UI.

![Collectors Management](assets/overview_page.png)

## Installation

Installing Odigos takes less than 5 minutes and requires no code changes.
Download our [CLI](https://docs.odigos.io/installation) and run the following command:


```bash
odigos install
```

For more details, see our [quickstart guide](https://docs.odigos.io/intro).

## Supported Destinations

**For step-by-step instructions detailed for every destination, see these [docs](https://docs.odigos.io/backends).**

### Managed

<<<<<<< Updated upstream
|                         | Traces  | Metrics | Logs |
|-------------------------| ------- | ------- |------|
| New Relic               | ✅      | ✅      | ✅    |
| Datadog                 | ✅      | ✅      | ✅    |
| Grafana Cloud           | ✅      | ✅      | ✅    |
| Honeycomb               | ✅      | ✅      | ✅    |
| Chronosphere            | ✅      | ✅      |       |
| Logz.io                 | ✅      | ✅      | ✅    |
| qryn.cloud              | ✅      | ✅      | ✅    |
| OpsVerse                | ✅      | ✅      | ✅    |
| Dynatrace               | ✅      | ✅      | ✅    |
| AWS S3                  | ✅      | ✅      | ✅    |
| Google Cloud Monitoring | ✅      |         | ✅    |
| Google Cloud Storage    | ✅      |         | ✅    |
| Azure Blob Storage      | ✅      |         | ✅    |
| Splunk                  | ✅      |         |      |
| Lightstep               | ✅      |         |      |
| Sentry                  | ✅      |         |      |
| Axiom                   | ✅      |         | ✅   |
| Sumo Logic              | ✅      | ✅      | ✅   |
| Coralogix               | ✅      | ✅      | ✅   |

### Open Source

|               | Traces | Metrics | Logs |
| ------------- | ------ | ------- | ---- |
| Prometheus    |        | ✅      |      |
| Tempo         | ✅     |         |      |
| Loki          |        |         | ✅   |
| Jaeger        | ✅     |         |      |
| SigNoz        | ✅     | ✅      | ✅   |
| qryn          | ✅     | ✅      | ✅   |
=======



|                         | Traces | Metrics | Logs |
| ------------------------- | -------- | --------- | ------ |
| Axiom                   | ✅     |         | ✅   |
| AWS S3                  | ✅     | ✅      | ✅   |
| Azure Blob Storage      | ✅     |         | ✅   |
| Chronosphere            | ✅     | ✅      |      |
| Coralogix               | ✅     | ✅      | ✅   |
| Datadog                 | ✅     | ✅      | ✅   |
| Dynatrace               | ✅     | ✅      | ✅   |
| Elastic APM             | ✅     |         | ✅   |
| Gigapipe                | ✅     | ✅      | ✅   |
| Google Cloud Monitoring | ✅     |         | ✅   |
| Google Cloud Storage    | ✅     |         | ✅   |
| Grafana Cloud           | ✅     | ✅      | ✅   |
| Honeycomb               | ✅     | ✅      | ✅   |
| Lightstep               | ✅     |         |      |
| Logz.io                 | ✅     | ✅      | ✅   |
| New Relic               | ✅     | ✅      | ✅   |
| OpsVerse                | ✅     | ✅      | ✅   |
| qryn.cloud              | ✅     | ✅      | ✅   |
| Sentry                  | ✅     |         |      |
| Splunk                  | ✅     |         |      |
| Sumo Logic              | ✅     | ✅      | ✅   |

### Open Source




| Backend       | Traces | Metrics | Logs |
| --------------- | -------- | --------- | ------ |
| ClickHouse    | ✅     | ✅      | ✅   |
>>>>>>> Stashed changes
| Elasticsearch | ✅     |         | ✅   |
| Jaeger        | ✅     |         |      |
| Loki          |        |         | ✅   |
| OTLP (gRPC)   | ✅     | ✅      | ✅   |
| OTLP (HTTP)   | ✅     | ✅      | ✅   |
| Prometheus    |        | ✅      |      |
| Quickwit      | ✅     |         | ✅   |
| qryn          | ✅     | ✅      | ✅   |
| SigNoz        | ✅     | ✅      | ✅   |
| Tempo         | ✅     |         |      |

Can't find the destination you need? Help us by following our quick [add new destination](https://docs.odigos.io/adding-new-dest) guide and submitting a PR.

## Contributing

Please refer to the [CONTRIBUTING.md](CONTRIBUTING.md) file for information about how to get involved. We welcome issues, questions, and pull requests. Feel free to join our active [Slack Community](https://join.slack.com/t/odigos/shared_invite/zt-1d7egaz29-Rwv2T8kyzc3mWP8qKobz~A).

## All Thanks To Our Contributors

<a href="https://github.com/odigos-io/odigos/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=keyval-dev/odigos" />
</a>

## License

This project is licensed under the terms of the Apache 2.0 open-source license. Please refer to [LICENSE](LICENSE) for the full terms.
