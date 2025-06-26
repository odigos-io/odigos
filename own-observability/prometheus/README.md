### 📘 Prometheus Operator Integration Guide

To ensure that Prometheus scrapes metrics from Odigos components, verify that your Prometheus custom resource is configured to discover `ServiceMonitor` objects across namespaces and with the correct label selectors.

#### Prometheus Configuration Requirements

Your Prometheus resource should include the following in its `spec`:

```yaml
serviceMonitorNamespaceSelector: {}
serviceMonitorSelector:
  matchLabels:
    release: prometheus
```

- `serviceMonitorNamespaceSelector: {}` allows Prometheus to discover ServiceMonitors in **all namespaces**, including `odigos-system`.
- `serviceMonitorSelector.matchLabels` ensures Prometheus only scrapes ServiceMonitors that contain a matching `release` label.

#### 🏷️ Required Label on Odigos ServiceMonitors

Odigos applies the following label to all its `ServiceMonitor` resources:

---
metadata:
  labels:
    release: prometheus
---

Make sure that the value (`prometheus`) matches what your Prometheus configuration expects.

> ℹ️ If your Prometheus instance expects a different label (e.g. `release: prometheus-stack` or `release: monitoring`), you must either:
> - Modify the `release` label in the Odigos ServiceMonitor resources to match your Prometheus configuration, **or**
> - Update the `serviceMonitorSelector.matchLabels` in your Prometheus spec to match `release: prometheus`.

#### How to Verify Target Discovery

Once everything is correctly configured, go to your Prometheus targets UI:

```bash
http://<your-prometheus-url>/targets
```

You should see Odigos targets like:
```bash
odigos-autoscaler-monitor   /metrics   UP
odigos-scheduler-monitor    /metrics   UP
odigos-instrumentor-monitor /metrics   UP
```
