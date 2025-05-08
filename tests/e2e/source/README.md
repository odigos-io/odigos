# Source object e2e

This e2e extensively tests the use of Source objects for instrumentation, uninstrumentation, and exclusion.

It has the following phases:

1. **Setup** - Install Odigos, simple-trace-db, and the Demo app.

2. **Workload instrumentation** - Create a Source for each individual workload, include a reported for each source. Add simple-trace-db as a destination. Verify:
    1. InstrumentationConfigs are created for each deployment
    2. Db is ready to receive traces
    3. The Odigos pipeline is ready
    4. Each deployment rolls out a new (instrumented) revision
    5. Generated traffic results in expected spans
    6. Context propagation works across deployments (service name is identical to the one configured by the Source)
    7. Resource attributes are present
    8. Span attributes are present
    9. Collector metrics are collected by UI

3. **Workload uninstrumentation** - Delete all Source objects for deployments. Verify:
    1. Workloads roll out a new (uninstrumented) revision

There are also the following temporary tests for migrating `odigos-instrumentation` labels to Sources:

1. Label a workload with `odigos-instrumentation: enabled` creates an Enabled Source
2. Changing the workload to `odigos-instrumentation: disabled` has no effect (existing Enabled Source overrides label)
3. Label a workload with `odigos-instrumentation: disabled` creates a Disabled Source
4. Changing the workload to `odigos-instrumentation: enabled` has no effect (existing Disabled Source overrides label)
5. Label a workload with `odigos-instrumentation: disabled` to create a Disabled Source, then update the Source to enable instrumentation (Source overrides `disabled` label)
6. Label a workload with `odigos-instrumentation: enabled` to create an Enabled Source, then update the Source to disable instrumentation (Source overrides `enabled` label)

## Workload generations and revisions

The various `*-workloads.yaml` files for each phase of the test look the `metadata.generation` value.
It is used to verify that the Odigos controllers have triggered an instrumentation rollout.

