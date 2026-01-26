# Development Guide

This guide provides advanced instructions for contributors and maintainers, covering topics such as debugging specific components, analyzing performance profiles, and working with internal tools. It complements the `CONTRIBUTING.md` by offering insights into advanced development workflows and optimization techniques.

---

## Go Workspace setup

To effectively develop Odigos, it is recommended to use [Go Workspaces](https://go.dev/blog/get-familiar-with-workspaces).

This repository is a mono-repo of multiple component and API modules, and Odigos also relies on other repos such as [OpenTelemetry Go Auto-Instrumentation](https://github.com/open-telemetry/opentelemetry-go-instrumentation)
that rely on locally built eBPF dependencies.

Using Go Workspaces allows IDEs like VSCode to resolve these dependencies locally without having to rely on complex `go mod replace` statements.

Go Workspaces are intended to be customized to your local development environment and folder structure. Because of that, we don't commit any `go.work` files to this repo,
and you may need to do some custom tweaking to make Go Workspaces work for you. But a good starting point is to run the following from the root folder of this repo:

```
go work init
go work use -r .
```

This will create a `go.work` file using all of the Odigos modules in this repository. It will look something like this:

```
go 1.23.5

use (
	./api
	./autoscaler
	./cli
<several more>
)
```

Other dependencies like Go Auto-Instrumentation or Odigos's [Runtime Detector](https://github.com/odigos-io/runtime-detector) will need to be added based on your local setup.

For example, if your local structure looks like this:

```
code/
-- github.com/
---- odigos-io/
------ odigos/
------ runtime-detector/
---- open-telemetry/
------ opentelemetry-go-instrumentation/
```

Update your `go.work` you created above in `odigos` to the following:

```
go 1.23.5

use (
   ../../open-telemetry/opentelemetry-go-instrumentation
   ../runtime-detector

	./api
	./autoscaler
	./cli
<several more>
)
```

This now tells VSCode that when you are working in the `odigos` folder to reference `../../open-telemetry/opentelemetry-go-instrumentation` and `../runtime-detector` for those dependencies.

## CPU and Memory Profiling for the Collectors

### Step 1: Port Forward the Gateway or Data Collection Pod
Forward the relevant pod to your local machine to enable profiling access:

```bash
kubectl port-forward pod/<pod-name> -n odigos-system 1777:1777
```

### Step 2: Collect Profiling Data

- **CPU Profile**
   Captures data about the time your application spends executing functions. Use this profile to identify performance bottlenecks, optimize CPU-intensive operations, and analyze which parts of the code consume the most CPU resources.

   ```bash
   curl -o cpu_profile.prof http://localhost:1777/debug/pprof/profile?seconds=30
   ```

- **Heap Memory Profile**
   Captures a snapshot of memory currently in use by your application after the latest garbage collection. Use this profile to identify memory leaks, track high memory usage, and analyze memory consumption by specific parts of the code.

   ```bash
   curl -o heap.out http://localhost:1777/debug/pprof/heap
   ```

- **Historical Memory Allocation**
   Provides insights into all memory allocations made by the program since it started running, including memory that has already been freed by the garbage collector (GC). This is useful for understanding memory allocation patterns and optimizing allocation behavior.

   ```bash
   curl -o allocs.out http://localhost:1777/debug/pprof/allocs
   ```

### Step 3: Analyze the Profiles
After collecting the profiling data, use the `go tool pprof` command to analyze the profiles visually in your web browser. Replace `<output file>` with the appropriate file (`cpu_profile.prof`, `heap.out`, or `allocs.out`):

```bash
go tool pprof -http=:8080 <output file>
```

This opens an interactive interface in your browser where you can:
- **Visualize Hotspots**: View flame graphs or directed graphs for easy identification of bottlenecks.
- **Drill Down**: Explore specific functions or memory allocations for detailed insights.

---

## Debugging CLI Commands

### 🧩 Debugging CLI `install` / `upgrade` / `uninstall` Commands (Helm-based)

Developers have a few options for debugging the Helm-based CLI commands:

#### Option 1: Run the Go module directly (without the embedded chart)

You can run the CLI from source while specifying the chart and image versions manually:

```
go run -ldflags "-X github.com/odigos-io/odigos/cli/pkg/helm.OdigosChartVersion=<CHART_VERSION>" -tags=embed_manifests . install --set image.tag=v<IMAGE_TAG>
```

This approach is useful when testing changes to the CLI logic itself without rebuilding the binary.

---

#### Option 2: Build the CLI with the embedded Helm chart

To test modifications made to the Helm chart, build the CLI with the chart embedded:

```
make cli-build
```

This command creates a CLI binary under the `cli/` directory bundled with your **local chart version**.
You can then run commands directly using the built binary:
```
cd cli
./odigos help
./odigos install --set image.tag=v<IMAGE_TAG>
```

Example output:
```
📦 Using embedded chart odigos (chart version: 0.0.0-e2e-test)

✅ Installed release "odigos" in namespace "odigos-system" (chart version: 0.0.0-e2e-test)
```

### Debugging the `cli pro` Command

To debug the `cli pro` command in Visual Studio Code, use the following configuration in your `.vscode/launch.json` file:

```jsonc
{
  "name": "cli pro",
  "type": "go",
  "request": "launch",
  "mode": "debug",
  "program": "${workspaceFolder}/cli",
  "cwd": "${workspaceFolder}/cli",
  "args": ["pro", "--onprem-token", "${input:onprem_token}"],
  "buildFlags": "-tags=embed_manifests"
}
```

#### How to Use
1. Open the **Run and Debug** view in Visual Studio Code:
   - Press `Ctrl+Shift+D` (Windows/Linux) or `Cmd+Shift+D` (macOS).
2. Select the `cli pro` configuration from the dropdown menu.
3. Click the green **Play** button to start debugging.
4. When prompted, enter your `onprem-token` value.
5. The debugger will start the `cli pro` command with the provided token and attach to the process for debugging.

---

### Debugging the `cli install` Command

To debug the `cli install` command in Visual Studio Code, use the following configuration in your `launch.json` file:

```jsonc
{
  "name": "cli install",
  "type": "go",
  "request": "launch",
  "mode": "debug",
  "program": "${workspaceFolder}/cli",
  "cwd": "${workspaceFolder}/cli",
  "args": ["install", "--version", "ODIGOS_VERSION"],
  "buildFlags": "-tags=embed_manifests"
}
```

#### How to Use
1. Open the **Run and Debug** view in Visual Studio Code:
   - Press `Ctrl+Shift+D` (Windows/Linux) or `Cmd+Shift+D` (macOS).
2. Select the `cli install` configuration from the dropdown menu.
3. Replace `"ODIGOS_VERSION"` in the `args` section with the desired version number.
4. Click the green **Play** button to start debugging.
5. The debugger will start the `cli install` command with the specified version.

---

### Updating OpenTelemetry dependencies

1. Update builder version and component versions in `collector/builder-config.yaml` and builder version in `collector/Makefile`
2. Update the `BUILDER_VERSION` in `collector/Makefile`
3. In `collector` directory, run `make genodigoscol generate` (may help to run in a Docker container)
4. In root directory, run `make go-mod-tidy`
5. Update `OTEL_*` versions in `Makefile`
6. Run `make update-otel`
7. Run `make go-mod-tidy`

Note that OTel frequently makes breaking changes upstream, deprecating and removing packages that will cause breaks.
Search the upstream OTel collector and collector-contrib repos for package deprecations if you get an error that a package isn't found.

It may help to run commands like `make update-otel` in a container to avoid interference with
your own Go mod cache. Try that if you see errors like this:

```
$ make update-otel
/Library/Developer/CommandLineTools/usr/bin/make update-dep MODULE=go.opentelemetry.io/collector/cmd/mdatagen VERSION=v0.136.0
cd ./api && go get go.opentelemetry.io/collector/cmd/mdatagen@v0.136.0
go: module go.opentelemetry.io/collector@v0.136.0 found, but does not contain package go.opentelemetry.io/collector/cmd/mdatagen
make[1]: *** [update-dep/./api] Error 1
make: *** [update-otel] Error 2
```
