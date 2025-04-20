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

3. **Workload uninstrumentation** - Delete all Source objects for deployments. Verify:
    1. Workloads roll out a new (uninstrumented) revision

4. **Namespace instrumentation** - Instrument Namespace. Verify:
    1. InstrumentationConfigs are present for each workload in the Namespace.
    2. Workloads roll out a new (instrumented) revision.
    3. Generated traffic results in expected spans (now the default service names which are the deployments names are used)
    4. Context propagation works
    5. Resource attributes are present
    6. Span attributes are present

5. **Namespace uninstrumentation** - Delete Namespace Source object. Verify:
    1. Workloads roll out a new (uninstrumented) revision

6. **Namespace+Workload instrumentation** - Instrument a single workload, then instrument the rest of the namespace. Verify:
    1. InstrumentationConfig is created for the single workload.
    2. Single workload rolls out a new revision.
    3. InstrumentationConfigs are then detected for all workloads.
    4. Remaining workloads roll out a new revision.
    5. Deleting Namespace Source does not delete individual Workload source.

7. **Workload exclusion** - Create an Excluded Source for another workload. Instrument the namespace. Verify:
    1. InstrumentationConfigs are created for all workloads except excluded workload
    2. All workloads except excluded workload roll out a new revision

8. **Workload inclusion** - Delete an Excluded source in an already-instrumented namespace. Verify:
    1. InstrumentationConfigs exist for all workloads in the namespace.
    2. Only the previously-excluded workload rolls out a new revision.
    3. Previously-excluded workload now has runtime detected

9. **Workload exclusion (2)** - Create an Excluded Source in an already-instrumented namespace. Verify:
    1. Only the newly excluded workload rolls out a new revision.
    2. InstrumentationConfigs exist for all workloads except newly excluded workload.
    3. Setting disableInstrumentation=false on excluded workload includes it
    4. Setting disableInstrumentation=true on included workload excludes it
    5. Triggering an irrelevant namespace update event does not trigger instrumentation
    6. Deleting an excluded Source in a non-instrumented namespace does not have any effect

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

