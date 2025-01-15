# Source object e2e

This e2e extensively tests the use of Source objects for instrumentation, uninstrumentation, and exclusion.

It has the following phases:

1. **Setup** - Install Odigos, Tempo, and the Demo app.
2. **Workload instrumentation** - Create a Source for each individual workload. Add Tempo as a destination. Verify:
   2.1 InstrumentationConfigs are created for each deployment
   2.2 Tempo is ready to receive traces
   2.3 The Odigos pipeline is ready
   2.4 Each deployment rolls out a new (instrumented) revision
   2.5 Generated traffic results in expected spans
   2.6 Context propagation works across deployments
   2.7 Resource attributes are present
   2.8 Span attributes are present
3. **Workload uninstrumentation** - Delete all Source objects for deployments. Verify:
   3.1 Workloads roll out a new (uninstrumented) revision
4. **Namespace instrumentation** - Update the service name for each deployment. Instrument Namespace. Verify:
   4.1 InstrumentationConfigs are present for each workload in the Namespace.
   4.2 Workloads roll out a new (instrumented) revision.
   4.3 Generated traffic results in expected spans (with new service name)
   4.4 Context propagation works
   4.5 Resource attributes are present
   4.6 Span attributes are present
5. **Namespace uninstrumentation** - Delete Namespace Source object. Verify:
   5.1 Workloads roll out a new (uninstrumented) revision
6. **Namespace+Workload instrumentation** - Instrument a single workload, then instrument the rest of the namespace. Verify:
   6.1 InstrumentationConfig is created for the single workload.
   6.2 Single workload rolls out a new revision.
   6.3 InstrumentationConfigs are then detected for all workloads.
   6.4 Remaining workloads roll out a new revision.
   6.5 Deleting Namespace Source does not delete individual Workload source.
7. **Workload exclusion** - Create an Excluded Source for another workload. Instrument the namespace. Verify:
   7.1 InstrumentationConfigs are created for all workloads except excluded workload
   7.2 All workloads except excluded workload roll out a new revision
8. **Workload inclusion** - Delete an Excluded source in an already-instrumented namespace. Verify:
   8.1 InstrumentationConfigs exist for all workloads in the namespace.
   8.2 Only the previously-excluded workload rolls out a new revision.
   8.3 Previously-excluded workload now has runtime detected
9. **Workload exclusion (2)** - Create an Excluded Source in an already-instrumented namespace. Verify:
   9.1 Only the newly excluded workload rolls out a new revision.
   9.2 InstrumentationConfigs exist for all workloads except newly excluded workload.
   9.3 Setting disableInstrumentation=false on excluded workload includes it
   9.4 Setting disableInstrumentation=true on included workload excludes it

## Workload generations and revisions

The various `*-workloads.yaml` files for each phase of the test look at 2 important values:

- The `deployment.kubernetes.io/revision` annotation
- The `metadata.generation` value

Changes to the workload manifest that don't result in a new rollout increase the `generation`, but not the `revision`.

In this case, the numbers become skewed when we annotate the deployments in step 4 (which does not trigger a rollout).

These numbers are used to verify that the Odigos controllers have triggered an instrumentation rollout.
