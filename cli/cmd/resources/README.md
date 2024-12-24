# RBAC Authorization

In this doc we'll keep track of the permissions requested across different resources, organized by `APIGroups`, `Resources`, and `Verbs`. Each permission includes comments to explain its purpose.

# UI

## ClusterRole

| APIGroups | Resources                                                                  | Verbs                    | Comments                                                                              |
| --------- | -------------------------------------------------------------------------- | ------------------------ | ------------------------------------------------------------------------------------- |
| ""        | namespaces                                                                 | get, list, patch         | Required to retrieve and modify namespace configurations during instrumentation.      |
| ""        | services, pods                                                             | get, list                | Required for discovering potential destinations and describing application workloads. |
| apps      | deployments, statefulsets, daemonsets                                      | get, list, patch, update | Needed for application instrumentation.                                               |
| apps      | replicasets                                                                | get, list                | Used for describing source and application configurations.                            |
| odigos.io | instrumentedapplications, instrumentationinstances, instrumentationconfigs | get, list, watch         | Used to retrieve and monitor instrumented applications and configurations.            |

## Role

| APIGroups         | Resources                          | Verbs                                           | Comments                                                    |
| ----------------- | ---------------------------------- | ----------------------------------------------- | ----------------------------------------------------------- |
| ""                | configmaps                         | get, list                                       | Accesses `odigos-config` for UI configuration settings.     |
| ""                | secrets                            | get, list, create, patch, update                | Manages destination secrets.                                |
| odigos.io         | instrumentationrules, destinations | get, list, create, patch, update, delete, watch | CRUD operations for destinations and instrumentation rules. |
| odigos.io         | collectorsgroups                   | get, list                                       | Monitors instrumentation groups.                            |
| actions.odigos.io | \*                                 | get, list, create, patch, update, delete        | Handles pipeline actions for custom logic.                  |

---

# Instrumentor

## ClusterRole

| APIGroups | Resources                             | Verbs                                           | Comments                                                          |
| --------- | ------------------------------------- | ----------------------------------------------- | ----------------------------------------------------------------- |
| ""        | nodes, namespaces                     | get, list, watch                                | Tracks runtime detection and resource labels for instrumentation. |
| apps      | daemonsets, deployments, statefulsets | get, list, watch, update, patch                 | Adjusts pod specifications for instrumentation.                   |
| odigos.io | instrumentedapplications              | delete, get, list, watch                        | Reacts to runtime detections in workloads.                        |
| odigos.io | instrumentedapplications/status       | get, patch, update                              | Updates application statuses post-injection.                      |
| odigos.io | instrumentationconfigs                | create, delete, get, list, patch, update, watch | Manages instrumentation configurations.                           |

## Role

| APIGroups | Resources        | Verbs            | Comments                                                    |
| --------- | ---------------- | ---------------- | ----------------------------------------------------------- |
| ""        | configmaps       | get, list, watch | Accesses `odigos-config` for instrumentation configuration. |
| odigos.io | collectorsgroups | get, list, watch | Monitors collectors and their statuses.                     |

---

# Scheduler

## ClusterRole

| APIGroups | Resources              | Verbs            | Comments                                                                   |
| --------- | ---------------------- | ---------------- | -------------------------------------------------------------------------- |
| odigos.io | instrumentationconfigs | get, list, watch | Monitors changes in instrumentation configurations for scheduling updates. |

## Role

| APIGroups | Resources        | Verbs                                           | Comments                                          |
| --------- | ---------------- | ----------------------------------------------- | ------------------------------------------------- |
| ""        | configmaps       | get, list, watch                                | Reads configuration details from `odigos-config`. |
| odigos.io | collectorsgroups | get, list, create, patch, update, watch, delete | Manages `collectorsgroups`.                       |
| odigos.io | destinations     | get, list, watch                                | Tracks destinations for scheduling behavior.      |

---

# Autoscaler

## ClusterRole

| APIGroups | Resources              | Verbs            | Comments                                                                          |
| --------- | ---------------------- | ---------------- | --------------------------------------------------------------------------------- |
| odigos.io | instrumentationconfigs | get, list, watch | Reads instrumentation configurations to populate the `data-collector` configmaps. |

## Role

| APIGroups   | Resources                             | Verbs                                                             | Comments                                                             |
| ----------- | ------------------------------------- | ----------------------------------------------------------------- | -------------------------------------------------------------------- |
| ""          | configmaps, services                  | get, list, watch, create, patch, update, delete, deletecollection | Manages collector configurations and services.                       |
| apps        | daemonsets, deployments               | get, list, watch, create, patch, update, delete, deletecollection | Oversees collector deployments and readiness statuses.               |
| apps        | daemonsets/status, deployments/status | get                                                               | Reads readiness statuses.                                            |
| autoscaling | horizontalpodautoscalers              | create, patch, update, delete                                     | Implements autoscaling for gateway collectors.                       |
| odigos.io   | destinations                          | get, list, watch                                                  | Tracks and synchronizes destination configurations.                  |
| odigos.io   | collectorsgroups, destinations/status | get, patch, update                                                | Monitors and updates statuses of collectors groups and destinations. |

---

# Collector

## ClusterRole

| APIGroups | Resources                                          | Verbs     | Comments                                                    |
| --------- | -------------------------------------------------- | --------- | ----------------------------------------------------------- |
| ""        | nodes/stats, nodes/proxy                           | get, list | Retrieves metrics for telemetry purposes.                   |
| ""        | pods                                               | get, list | Accesses metadata for resource name processors.             |
| apps      | replicasets, deployments, daemonsets, statefulsets | get, list | Fetches application details for instrumentation.            |
| policy    | podsecuritypolicies                                | use       | Supports clients enabling pod security policies (optional). |
