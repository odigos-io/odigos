# Workload Lifecycle

This e2e test verify various scenarios related to the lifecycle of workloads in the cluster.

## Node.js Workloads

### nodejs-unsupported-version

This workload is running Node.js version 8 and verify that odigos can ignore it gracefully.
Odigos is expected to detect the runtime version from the environment in the base docker image and not apply any instrumentation device to the deployment, should not restart the pods, and report the issue in instrumented application CR.

## Steps

## Step 01

Adds the initial workloads, instrument the ns and add destination to odigos.
Verify the expected state for each workload according to it's caracteristics.

In this step we deploy the following workloads:

- nodejs-unsupported-version - should detect the runtime version and avoid instrumentation device.
