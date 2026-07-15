import { gql } from '@apollo/client';

const WORKLOAD_FIELDS_SLIM = `
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
`;

export const GET_WORKLOADS = gql`
  query GetWorkloads($filter: WorkloadFilter) {
    workloads(filter: $filter) {
      ${WORKLOAD_FIELDS_SLIM}
    }
  }
`;

export const GET_WORKLOADS_BY_IDS_SLIM = gql`
  query GetWorkloadsByIdsSlim($ids: [K8sWorkloadIdInput!]!) {
    workloadsByIds(ids: $ids) {
      ${WORKLOAD_FIELDS_SLIM}
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
      rollbackOccurred
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
      podsOdigosHealthStatus {
        name
        status
        reasonEnum
        message
      }
      markedForInstrumentation {
        markedForInstrumentation
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
        podsManifestInjection {
          name
          status
          reasonEnum
          message
          actionItems {
            type
            userFacingText
          }
        }
        autoRollback {
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
      rollout {
        rolloutStatus {
          name
          status
          reasonEnum
          message
        }
        podsManifestInjectionStatus {
          name
          status
          reasonEnum
          message
          actionItems {
            type
            userFacingText
          }
        }
      }
      autoRollback {
        autoRollbackStatus {
          name
          status
          reasonEnum
          message
        }
        rollbackOccurred
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
            status
            reasonEnum
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
        agentConfig {
          traces {
            headSampling {
              dryRun
              spanMetricsMode
              noisyOperations {
                ruleId
                name
                disabled
                operation {
                  httpServer {
                    route
                    routePrefix
                    method
                    queryParams {
                      name
                      valueExact
                    }
                  }
                  httpClient {
                    serverAddress
                    templatedPath
                    templatedPathPrefix
                    method
                  }
                }
                percentageAtMost
              }
            }
          }
        }
        collectorConfig {
          tailSampling {
            noisyOperations {
              ruleId
              name
              disabled
              operation {
                httpServer {
                  route
                  routePrefix
                  method
                  queryParams {
                    name
                    valueExact
                  }
                }
                httpClient {
                  serverAddress
                  templatedPath
                  templatedPathPrefix
                  method
                }
              }
              percentageAtMost
            }
            highlyRelevantOperations {
              ruleId
              name
              disabled
              error
              durationAtLeastMs
              operation {
                httpServer {
                  route
                  routePrefix
                  method
                }
                kafkaConsumer {
                  kafkaTopic
                }
                kafkaProducer {
                  kafkaTopic
                }
              }
              percentageAtLeast
            }
            costReductionRules {
              ruleId
              name
              disabled
              operation {
                httpServer {
                  route
                  routePrefix
                  method
                }
                kafkaConsumer {
                  kafkaTopic
                }
                kafkaProducer {
                  kafkaTopic
                }
              }
              percentageAtMost
            }
          }
        }
        instrumentations {
          name
          healthy
          message
          isStandardLibrary
        }
      }
      pods {
        podName
        nodeName
        startTime
        agentInjected
        agentInjectedStatus {
          name
          status
          reasonEnum
          message
        }
        k8sHealthStatus {
          name
          status
          reasonEnum
          message
        }
        odigosHealthStatus {
          name
          status
          reasonEnum
          message
        }
        containers {
          containerName
          otelDistroName
          started
          ready
          isCrashLoop
          restartCount
          runningStartedTime
          waitingReasonEnum
          waitingMessage
          k8sHealthStatus {
            name
            status
            reasonEnum
            message
          }
          odigosHealthStatus {
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
              healthy
              message
              isStandardLibrary
            }
          }
        }
      }
      telemetryMetrics {
        throughputBytes
      }
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
