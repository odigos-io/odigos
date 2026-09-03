# Log Collection E2E Test

Verifies log collection through the **filelog receiver**: container stdout from instrumented
workloads is scraped, enriched with kubernetes resource attributes, and delivered to a destination
that declares the `LOGS` signal.

## What it covers

- A logs pipeline is generated once a destination declares `LOGS`, and not before.
- The filelog receiver collects application output written during the traffic run, not only startup
  lines.
- `odigoslogsresourceattrsprocessor` resolves the pod from the log file path and adds
  `k8s.pod.name`, `k8s.namespace.name`, `k8s.container.name` and a `service.name` the application
  never sent.

## Ordering

Two constraints make this test order-sensitive, and both are waited on explicitly:

- The node collector does not build a logs pipeline until the cluster collector reports that a
  destination receives logs.
- filelog is configured with `start_at: end`, so anything written before the receiver is running is
  never collected.

Traffic generated before the pipeline exists produces no logs, which is indistinguishable from
collection being broken.
