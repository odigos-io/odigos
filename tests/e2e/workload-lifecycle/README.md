# Workload Lifecycle

This e2e test verify various scenarios related to the lifecycle of workloads in the cluster.

## Node.js Workloads

### nodejs-unsupported-version

This workload is running Node.js version 8 and it has the NODE_VERSION environment variable set in the image so odigos can detect it.
Odigos is expected to not add instrumentation device to the deployment, should not restart the pods, and report the issue in instrumented application CR.

### nodejs-very-old-version

This workload is running Node.js version 8 and it has the NODE_VERSION environment variable set in the image so odigos can detect it.
Odigos is expected to add instrumentation device to the deployment, should restart the pods, but the agent should not load due to the unsupported version.

### nodejs-minimum-version

This workload runs the nodejs http server test app with node 14.0.0 which is the minimum supported version by the agent.
Instrumentation is expected to work for this workload.

### nodejs-latest-version

Workload that runs nodejs http server with the latest version of nodejs from dockerhub.
Make sure application is stable and the agent is able to instrument it.

### cpp-http-server

A workload in CPP which odigos does not support.

## Steps

## Step 01

Adds the initial workloads, instrument the ns and add destination to odigos.
Verify the expected state for each workload according to it's caracteristics.

In this step we deploy the following workloads:

- nodejs-unsupported-version - should detect the runtime version and avoid instrumentation device.
- nodejs-very-old-version - should not detect the runtime version and add instrumentation device but the agent should not load and application can run as usual.
- nodejs-minimum-version - should detect the runtime version and add instrumentation device, and report success in the instrumented instance CR.
- nodejs-latest-version - should detect the runtime version and add instrumentation device, and report success in the instrumented instance CR.
- cpp-http-server - should detect the runtime language as unknown and avoid instrumentation device.
