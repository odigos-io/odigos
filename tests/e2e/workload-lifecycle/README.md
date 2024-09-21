# Workload Lifecycle

This e2e test verify various scenarios related to the lifecycle of workloads in the cluster.

## Node.js Workloads

### nodejs-unsupported-version

- Node.js version 8 (minimum support version is 14)
- NODE_VERSION environment variable set in the image - detected by runtime inspection
- Should not be instrumented

### nodejs-very-old-version

- Node.js version 8 (minimum support version is 14)
- NODE_VERSION environment variable NOT set in the image - runtime version not detected by runtime inspection
- Instrumentation device should be added, but agent should not load

### nodejs-minimum-version

- Node.js version 14.0.0 (minimum supported version)
- NODE_VERSION environment variable set in the image - detected by runtime inspection
- Instrumentation device should be added, agent should load and report traces correctly.
- This workload verifies that we support the minimum version we claim.

### nodejs-latest-version

- Node.js version with label `current` from dockerhub
- NODE_VERSION environment variable set in the image - detected by runtime inspection
- Instrumentation device should be added, agent should load and report traces correctly
- This workload checks if something in the latest version of nodejs broke the agent.

### nodejs-docker-env

- Node.js version `20.17.0` (common version)
- Application uses NODE_OPTIONS environment variable in the dockerfile to set one `--require` flag and another `--max-old-space-size` flag.
- This workload verifies that after instrumentation is applied, those 2 options still works as expected.

### nodejs-manifest-env

- Node.js version `20.17.0` (common version)
- Application uses NODE_OPTIONS environment variable in the k8s deployment manifest to set one `--require` flag and another `--max-old-space-size` flag.
- This workload verifies that after instrumentation is applied, those 2 options still works as expected.

## CPP Workloads

- Workload with a language odigos does not support.
- Should not be instrumented or restarted.

## Steps

## Step 01 - Deploy Initial Workloads and Instrumentation

Adds the initial workloads, instrument the ns and add destination to odigos.
Verify the expected state for each workload according to it's caracteristics.

- nodejs-unsupported-version
  - detect the runtime version
  - avoid adding instrumentation device
- nodejs-very-old-version
  - should NOT detect the runtime version
  - should add instrumentation device for unknown runtime version
  - should not create instrumentation instance
  - agent should not load
- nodejs-minimum-version
  - should detect the runtime version
  - should add instrumentation device
  - should report health in the instrumented instance CR
  - agent should load and report traces
- nodejs-latest-version - should detect the runtime version and add instrumentation device, and report success in the instrumented instance CR.
  - should detect the runtime version
  - should add instrumentation device
  - should report health in the instrumented instance CR
  - agent should load and report traces
- nodejs-docker-env
  - should detect the runtime version and NODE_OPTIONS value from container env
  - should add instrumentation device and patch the NODE_OPTIONS value
  - should report health in the instrumented instance CR
  - agent should load and report traces, and verify the `--require` script is loaded correctly and the `--max-old-space-size` is in effect in v8 runtime.
- nodejs-manifest-env
  - should detect the runtime version and NODE_OPTIONS value from container env
  - should add instrumentation device and patch the NODE_OPTIONS value
  - should report health in the instrumented instance CR
  - agent should load and report traces, and verify the `--require` script is loaded correctly and the `--max-old-space-size` is in effect in v8 runtime.
- cpp-http-server
  - should NOT detect the runtime language and report it as `unknown`
  - should NOT add instrumentation device

## Step 02 - Update Workload Manifest

This steps will make a change in the workload manifests and make sure that after the new revision, the applications are running, instrumented and produce traces.

- nodejs-unsupported-version
  - deployment manifest should be patched by odigos
  - workload should not restart due to odigos
- nodejs-very-old-version
  - should reinject the instrumentation device as runtime version not available
  - process should restart and agent should not load
- nodejs-minimum-version
  - deployment manifest should be patched by odigos
  - workload should restart and agent should load
  - it should report traces
- nodejs-latest-version
  - deployment manifest should be patched by odigos
  - workload should restart and agent should load
  - it should report traces
- nodejs-docker-env
  - deployment manifest should be patched by odigos, both instrumentation device and NODE_OPTIONS env
  - workload should restart and agent should load
  - it should report traces
- nodejs-manifest-env
  - deployment manifest should be patched by odigos, both instrumentation device and NODE_OPTIONS env
  - workload should restart and agent should load
  - it should report traces
- cpp-http-server
  - deployment manifest should be patched by odigos
  - workload should not restart and agent should not load
