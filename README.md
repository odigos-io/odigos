<p align="center">
    <a href="https://github.com/odigos-io/odigos/actions/workflows/release.yml" target="_blank">
        <img src="https://github.com/odigos-io/odigos/actions/workflows/release.yml/badge.svg" alt="Release Odigos CLI"/>
    </a>
    <a href="https://goreportcard.com/report/github.com/odigos-io/odigos/cli" target="_blank">
        <img src="https://goreportcard.com/badge/github.com/odigos-io/odigos/cli" alt="Go Report Card"/>
    </a>
    <a href="https://godoc.org/github.com/odigos-io/odigos/cli" target="_blank">
        <img src="https://godoc.org/github.com/odigos-io/odigos/cli?status.svg" alt="GoDoc"/>
    </a>
</p>

[![Logo Dark Mode](./docs/logo/logo_white.svg)](https://odigos.io#gh-dark-mode-only)
[![Logo Light Mode](./docs/logo/logo_black.svg)](https://odigos.io#gh-light-mode-only)

<div align="center">
    <h2>Generate distributed traces for any application without code changes.</h2>
    <h4>If you find Odigos helpful, please ⭐ this repository to show your support!</h4>
    <h2>
        <a href="https://odigos.io/?demo">Demo Video</a>
        • <a href="https://docs.odigos.io">Documentation</a>
        • <a href="https://join.slack.com/t/odigos/shared_invite/zt-1d7egaz29-Rwv2T8kyzc3mWP8qKobz~A">Join Slack Community</a>
    </h2>
</div>

## What is Odigos?

Odigos is an open-source distributed tracing solution that simplifies and improves observability for Kubernetes environments and Virtual Machines. It provides instant tracing capabilities without requiring any code changes to your applications.

Odigos empowers platform engineers, DevOps professionals, and SREs to enhance their observability strategies quickly and effectively. It is an ideal solution for modern cloud-native environments, combining simplicity, performance, and industry expertise.

## Why choose Odigos?

1. **Simplicity** : Implement distributed tracing with minimal effort and complexity.
2. **Performance** : Separates data recording and processing to minimize runtime impact.
3. **Community-Backed** : With 3,000+ GitHub stars and a growing contributor base.
4. **Expertise** : Created by multiple maintainers of OpenTelemetry, ensuring deep integration and alignment with industry standards.

## Key Features

- **Code-Free Instrumentation** : Set up distributed tracing in minutes, eliminating manual code modifications.
- **Multi-Language Support** : Works with [Java](https://docs.odigos.io/instrumentations/java), [Python](https://docs.odigos.io/instrumentations/python), [.NET](https://docs.odigos.io/instrumentations/dotnet), [Node.js](https://docs.odigos.io/instrumentations/nodejs), and [Go](https://docs.odigos.io/instrumentations/golang) applications.
- **eBPF-Powered** : Utilizes eBPF technology for high-performance instrumentation.
- **OpenTelemetry Compatible** : Generates traces in OpenTelemetry format for broad tool compatibility.
- **Vendor Agnostic** : Integrates with various [monitoring solutions](https://docs.odigos.io/backends-overview), avoiding vendor lock-in.
- **Automatic Scaling** : Manages and scales OpenTelemetry collectors based on data volume.
- **Opinionated Defaults** : Supplies common defaults and best practices out-of-the-box, requiring no deep knowledge of OpenTelemetry.

### ✨ Language Agnostic Auto-instrumentation

Odigos supports any application written in Java, Python, .NET, Node.js, and **Go**.
Historically, compiled languages like Go have been difficult to instrument without code changes. Odigos solves this problem by uniquely leveraging [eBPF](https://ebpf.io).

![Works on any application](./docs/images/ui_choose_apps.png)

### 🤝 Keep your existing observability tools

Odigos currently supports all the popular managed and open-source destinations.
By producing data in the [OpenTelemetry](https://opentelemetry.io) format, Odigos can be used with any observability tool that supports OTLP.

For a complete list of supported destinations, see [here](#supported-destinations).

![Works with any observability tool](./docs/images/ui_choose_dest.png)

### 🎛️ Collectors Management

Odigos automatically scales OpenTelemetry collectors based on observability data volume.
Manage and configure collectors via a convenient web UI.

![Collectors Management](./docs/images/ui_overview.png)

## Installation

Installing Odigos takes less than 5 minutes and requires no code changes.<br />
Download our [CLI](https://docs.odigos.io/quickstart/installation) and run the following command:

```bash
odigos install
```

For more information, see our [quickstart guide](https://docs.odigos.io/quickstart/introduction).

## Contributing

Please refer to the [CONTRIBUTING.md](CONTRIBUTING.md) file for information about how to get involved. We welcome issues, questions, and pull requests.

## Thanks to our Contributors

<a href="https://github.com/odigos-io/odigos/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=keyval-dev/odigos" />
</a>

## License

This project is licensed under the terms of the Apache 2.0 open-source license. Please refer to [LICENSE](LICENSE) for the full terms.
