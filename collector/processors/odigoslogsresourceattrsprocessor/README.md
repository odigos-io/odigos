# Odigos Logs Resource Attributes Processor

A custom Odigos OpenTelemetry Collector processor that enriches log resource attributes with Kubernetes metadata.

## Overview

This processor watches pods on the local node using the Kubernetes metadata API and enriches incoming logs with workload information based on the `k8s.pod.uid` resource attribute.

## Attributes Added

| Attribute | Description |
|-----------|-------------|
| `service.name` | Resolved workload name (e.g., Deployment name) |
| `k8s.pod.name` | Pod name |
| `k8s.namespace.name` | Namespace name |
| `k8s.deployment.name` | Deployment name (if applicable) |
| `k8s.daemonset.name` | DaemonSet name (if applicable) |
| `k8s.statefulset.name` | StatefulSet name (if applicable) |
| `k8s.job.name` | Job name (if applicable) |
| `k8s.cronjob.name` | CronJob name (if applicable) |
| `k8s.argoproj.rollout.name` | Argo Rollout name (if applicable) |

## Configuration

```yaml
processors:
  odigoslogsresourceattrs:
```

No configuration options are required. The processor automatically uses the `NODE_NAME` environment variable to filter pods on the local node.

## Requirements

- `NODE_NAME` environment variable must be set to the node name
- Kubernetes RBAC permissions to watch pod metadata
