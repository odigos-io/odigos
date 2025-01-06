import { gql } from '@apollo/client';

export const DESCRIBE_ODIGOS = gql`
  query DescribeOdigos {
    describeOdigos {
      odigosVersion {
        name
        value
        status
        explain
      }
      numberOfDestinations
      numberOfSources
      clusterCollector {
        enabled {
          name
          value
          status
          explain
        }
        collectorGroup {
          name
          value
          status
          explain
        }
        deployed {
          name
          value
          status
          explain
        }
        deployedError {
          name
          value
          status
          explain
        }
        collectorReady {
          name
          value
          status
          explain
        }
        deploymentCreated {
          name
          value
          status
          explain
        }
        expectedReplicas {
          name
          value
          status
          explain
        }
        healthyReplicas {
          name
          value
          status
          explain
        }
        failedReplicas {
          name
          value
          status
          explain
        }
        failedReplicasReason {
          name
          value
          status
          explain
        }
      }
      nodeCollector {
        enabled {
          name
          value
          status
          explain
        }
        collectorGroup {
          name
          value
          status
          explain
        }
        deployed {
          name
          value
          status
          explain
        }
        deployedError {
          name
          value
          status
          explain
        }
        collectorReady {
          name
          value
          status
          explain
        }
        daemonSet {
          name
          value
          status
          explain
        }
        desiredNodes {
          name
          value
          status
          explain
        }
        currentNodes {
          name
          value
          status
          explain
        }
        updatedNodes {
          name
          value
          status
          explain
        }
        availableNodes {
          name
          value
          status
          explain
        }
      }
      isSettled
      hasErrors
    }
  }
`;

export const DESCRIBE_SOURCE = gql`
  query DescribeSource($namespace: String!, $kind: String!, $name: String!) {
    describeSource(namespace: $namespace, kind: $kind, name: $name) {
      name {
        name
        value
        status
        explain
      }
      kind {
        name
        value
        status
        explain
      }
      namespace {
        name
        value
        status
        explain
      }
      labels {
        instrumented {
          name
          value
          status
          explain
        }
        workload {
          name
          value
          status
          explain
        }
        namespace {
          name
          value
          status
          explain
        }
        instrumentedText {
          name
          value
          status
          explain
        }
      }
      runtimeInfo {
        containers {
          containerName {
            name
            value
            status
            explain
          }
          language {
            name
            value
            status
            explain
          }
          runtimeVersion {
            name
            value
            status
            explain
          }
          envVars {
            name
            value
            status
            explain
          }
        }
      }
      instrumentationConfig {
        created {
          name
          value
          status
          explain
        }
        createTime {
          name
          value
          status
          explain
        }
        containers {
          containerName {
            name
            value
            status
            explain
          }
          language {
            name
            value
            status
            explain
          }
          runtimeVersion {
            name
            value
            status
            explain
          }
          envVars {
            name
            value
            status
            explain
          }
        }
      }
      instrumentationDevice {
        statusText {
          name
          value
          status
          explain
        }
        containers {
          containerName {
            name
            value
            status
            explain
          }
          devices {
            name
            value
            status
            explain
          }
          originalEnv {
            name
            value
            status
            explain
          }
        }
      }
      totalPods
      podsPhasesCount
      pods {
        podName {
          name
          value
          status
          explain
        }
        nodeName {
          name
          value
          status
          explain
        }
        phase {
          name
          value
          status
          explain
        }
        containers {
          containerName {
            name
            value
            status
            explain
          }
          actualDevices {
            name
            value
            status
            explain
          }
          instrumentationInstances {
            healthy {
              name
              value
              status
              explain
            }
            message {
              name
              value
              status
              explain
            }
            identifyingAttributes {
              name
              value
              status
              explain
            }
          }
        }
      }
    }
  }
`;
