# Odiglet

Building the odiglet binary: **`make build-odiglet`** runs **`setup-obi`** first, then **`go generate`** for other eBPF deps and **`go build`**. **`setup-obi`** materializes OBI's bpf2go artifacts and adds a local-path **`go mod replace`**, via one of two methods selected with **`OBI_SETUP`** (`pin` | `release`, default `pin`): **`setup-obi-pin`** clones the pinned **`OBI_COMMIT`** and runs OBI's **bpf2go** codegen in the clone; **`setup-obi-release`** downloads/verifies OBI's **`obi-$(OBI_VERSION)-source-generated`** tarball. Keep the **`go.opentelemetry.io/obi`** version in **`go.mod`** in sync with the selected method (**`OBI_COMMIT`** pseudo-version for pin, **`OBI_VERSION`** tag for release). The **`odiglet/Dockerfile`** uses the same **`make build-odiglet`** / **`make setup-obi`** (no separate OBI image stage). See [docs/obi-bpf2go-module-cache.md](docs/obi-bpf2go-module-cache.md).

## Development
One of Odiglet's jobs is to manage the different eBPF instrumentations. Loading an eBPF instrumentation requires having compiled eBPF programs (.o files). This compilation is taking place in Odiglet's Dockerfile and it requires the auto instrumentation code. This makes debugging locally on a non-linux system different compared to the other Odigos components.
Assuming a setup with an active kind cluster with Odigos installed:
1. Run `make debug-odiglet` or `TAG=<some_tag> make debug-odiglet` which will build Odiglet in a debug image which includes a Go debugger.
In addition, it will port-forward the debugger port for remote debug.
2. Using vscode launch the `Remote Odiglet` configuration.
3. Debug the code.

Odiglet defaults to use the environment variable `OTEL_LOG_LEVEL` with value of `info`. When debugging or developing it is useful to increase the log level to `debug` for instrumentations. Note that this value only controls the log level for the instrumentations Odiglet invokes directly (eBPF) and does not apply for the k8s controllers or 3rd party agents.
