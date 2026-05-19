# Source object e2e

This e2e extensively tests the use of Source objects for instrumentation, uninstrumentation, and exclusion.

It has the following phases:

1. **Setup** - Install Odigos, simple-trace-db, and the Demo app
   ([simple-demo](https://github.com/odigos-io/simple-demo) `v0.1.36`). The demo app now
   includes the C++ `shipping` service, which is instrumented in this test via the OBI
   container override.

2. **Workload instrumentation** - Create a Source for each individual workload, include a reported for each source. Add simple-trace-db as a destination. Verify:
    1. InstrumentationConfigs are created for each deployment
    2. Db is ready to receive traces
    3. The Odigos pipeline is ready
    4. Each deployment rolls out a new (instrumented) revision (except `shipping`, see below)
    5. The `shipping` Source uses a `containerOverrides` entry that selects the
       `opentelemetry-ebpf-instrumentation` (OBI) distro, as described in
       [the Odigos OBI docs](https://docs.odigos.io/oss/instrumentations/obi). Because OBI does
       not require a pod restart, `shipping`'s generation remains at `1` and its
       `InstrumentationConfig` is asserted to report `otelDistroName:
       opentelemetry-ebpf-instrumentation` together with `language: cplusplus`.
    6. Generated traffic to the frontend's `/buy` endpoint fans out to the C++ `shipping`
       service (via `SHIPPING_SERVICE_HOST`) and produces server spans observable through
       OBI (verified via `wait-for-shipping-trace.yaml`)
    7. Context propagation works across deployments (service name is identical to the one configured by the Source)
    8. Resource attributes are present
    9. Span attributes are present
    10. Collector metrics are collected by UI

3. **Workload uninstrumentation** - Delete all Source objects for deployments. Verify:
    1. Workloads roll out a new (uninstrumented) revision (except `shipping`, which stays at
       generation `1` because OBI never required a restart in the first place)


## Workload generations and revisions

The various `*-workloads.yaml` files for each phase of the test look the `metadata.generation` value.
It is used to verify that the Odigos controllers have triggered an instrumentation rollout.

`shipping` is an exception: it is instrumented via OBI (eBPF) which sets `noRestartRequired:
true` on its `runtimeAgent`. As a result Odigos does not patch the pod spec to enable OBI, so
the deployment's `metadata.generation` stays at `1` through both the instrumentation and
uninstrumentation phases.

