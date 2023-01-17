<p align="center">
    <a href="https://github.com/keyval-dev/odigos/actions/workflows/main.yml" target="_blank">
    <img src="https://github.com/keyval-dev/odigos/actions/workflows/main.yml/badge.svg" />
    </a>
    <a href="https://goreportcard.com/report/github.com/keyval-dev/odigos/cli" target="_blank">
    <img src="https://goreportcard.com/badge/github.com/keyval-dev/odigos/cli">
    </a>
    <a href="https://godoc.org/github.com/keyval-dev/odigos/cli" target="_blank">
    <img src="https://godoc.org/istio.io/istio?status.svg">
    </a>
</p>
<h1 align="center" font-weight="100">
     Odigos - Adopt Distributed Tracing in Minutes
  </h1>
<p align="center">
<a href="https://www.youtube.com/watch?v=9d36AmVtuGU">
  <img
    src="assets/odigos.gif"
    width="1000"
    alt="Odigos - Observability Control Plane"
    border="0"
/>
</a>
</p>

### **Fix production issues faster by generating distributed traces, metrics and logs for any application in minutes, without code changes.**

- 🧑‍💻 **No code changes** - Odigos detects the programming language of your applications and apply automatic instrumentation accordingly.
- 📖 **Open technologies** - Applications are instrumented using well-known, battle-tested open source observability technologies such as [OpenTelemetry](https://opentelemetry.io) and [eBPF](https://ebpf.io).
- 🚀 **Boost your existing monitoring tools** - No need for changing tools. Use your favourite tool, with much more data.
- ✨ **Works on any application** - Get automatic distributed traces and metrics even for applications written in Go. Odigos leverage eBPF in a unique way that removes the need to manually instrument even compiled languages.
- 🔭 **Observability by default** - Automatically get traces, metrics and logs for every new deployed application.

<h2 align="center">
    <a href="https://docs.odigos.io/intro">Getting Started Guide</a> • <a href="https://docs.odigos.io">Documentation</a> • <a href="https://join.slack.com/t/odigos/shared_invite/zt-1d7egaz29-Rwv2T8kyzc3mWP8qKobz~A">Join Slack Community</a>
</h2>

## Installation

The easiest way to install Odigos is to use our [Helm chart](https://github.com/keyval-dev/odigos-charts) by running the following commands:

```console
helm repo add odigos https://keyval-dev.github.io/odigos-charts/

helm install my-odigos odigos/odigos --namespace odigos-system --create-namespace
```

See the [quickstart guide](https://docs.odigos.io/intro) for more details and examples.

## Supported Destinations

See [DESTINATIONS.md](DESTINATIONS.md) file for a complete list of supported destinations and the available signals for every destination.

Can't find the destination you need? Help us by following our quick [adding new destination](https://docs.odigos.io/adding-new-dest) guide and submit a PR.

## Project Status

This project is actively maintained by [keyval](https://keyval.dev). We would love to receive your ideas, feedback & contributions.

## Contributing

Please refer to the [CONTRIBUTING.md](CONTRIBUTING.md) file for information about how to get involved. We welcome issues, questions, and pull requests. You are welcome to join our active [Slack Community](https://join.slack.com/t/odigos/shared_invite/zt-1d7egaz29-Rwv2T8kyzc3mWP8qKobz~A).

## License

This project is licensed under the terms of the [Apache 2.0](LICENSE-Apache-2.0) open source license. Please refer to [LICENSE](LICENSE) for the full terms.
