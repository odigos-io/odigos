import { gql } from '@apollo/client';

export const GET_WORKLOADS = gql`
  query GetWorkloads($filter: WorkloadFilter) {
    workloads(filter: $filter) {
      id {
        namespace
        kind
        name
      }
      serviceName
      dataStreamNames
      runtimeInfo {
        detectedLanguages
      }
      workloadOdigosHealthStatus {
        name
        status
        reasonEnum
        message
      }
      podsAgentInjectionStatus {
        name
        status
        reasonEnum
        message
      }
    }
  }
`;

export const GET_WORKLOADS_BY_IDS = gql`
  query GetWorkloadsByIds($ids: [K8sWorkloadIdInput!]!) {
    workloadsByIds(ids: $ids) {
      id {
        namespace
        kind
        name
      }
      serviceName
      dataStreamNames
      numberOfInstances
      markedForInstrumentation {
        markedForInstrumentation
      }
      runtimeInfo {
        detectedLanguages
      }
      containers {
        containerName
        runtimeInfo {
          language
          runtimeVersion
        }
        agentEnabled {
          agentEnabled
          agentEnabledStatus {
            message
          }
          otelDistroName
        }
        overrides {
          containerName
          otelDistroName
          runtimeInfo {
            language
            runtimeVersion
          }
        }
      }
      conditions {
        runtimeDetection {
          name
          status
          reasonEnum
          message
        }
        agentInjectionEnabled {
          name
          status
          reasonEnum
          message
        }
        rollout {
          name
          status
          reasonEnum
          message
        }
        agentInjected {
          name
          status
          reasonEnum
          message
        }
        processesAgentHealth {
          name
          status
          reasonEnum
          message
        }
        expectingTelemetry {
          name
          status
          reasonEnum
          message
        }
      }
      workloadOdigosHealthStatus {
        name
        status
        reasonEnum
        message
      }
      podsAgentInjectionStatus {
        name
        status
        reasonEnum
        message
      }
      podsHealthStatus {
        name
        status
        reasonEnum
        message
      }
      pods {
        podName
        nodeName
        startTime
        agentInjected
        startedPostAgentMetaHashChange
        agentInjectedStatus {
          name
          status
          reasonEnum
          message
        }
        runningLatestWorkloadRevision
        podHealthStatus {
          name
          status
          reasonEnum
          message
        }
        containers {
          containerName
          odigosInstrumentationDeviceName
          otelDistroName
          started
          ready
          isCrashLoop
          restartCount
          runningStartedTime
          waitingReasonEnum
          waitingMessage
          healthStatus {
            name
            status
            reasonEnum
            message
          }
          processes {
            healthy
            healthStatus {
              name
              status
              reasonEnum
              message
            }
            identifyingAttributes {
              name
              value
            }
            instrumentations {
              name
              isStandardLibrary
            }
          }
        }
      }
      rollbackOccurred
    }
  }
`;

export const GET_NAMESPACES = gql`
  query GetNamespaces {
    namespaces {
      name
      markedForInstrumentation
      dataStreamNames
    }
  }
`;

export const GET_NAMESPACES_WITH_WORKLOADS = gql`
  query GetNamespacesWithWorkloads {
    namespaces {
      name
      markedForInstrumentation
      dataStreamNames
      workloads {
        id {
          namespace
          kind
          name
        }
        markedForInstrumentation {
          markedForInstrumentation
        }
        dataStreamNames
        numberOfInstances
      }
    }
  }
`;
