import { gql } from '@apollo/client';

export const GET_GATEWAY_INFO = gql`
  query GatewayDeploymentInfo {
    gatewayDeploymentInfo {
      status
      hpa {
        min
        max
        current
        desired
        conditions {
          status
          type
          reason
          message
          lastTransitionTime
        }
      }
      resources {
        requests {
          cpu
          memory
        }
        limits {
          cpu
          memory
        }
      }
      imageVersion
      lastRolloutAt
      rolloutInProgress
      manifestYAML
      configMapYAML
    }
  }
`;

export const GET_GATEWAY_PODS = gql`
  query GatewayPods {
    gatewayPods {
      namespace
      name
      status
      restartsCount
      nodeName
      creationTimestamp
      image
      collectorMetrics {
        metricsAcceptedRps
        metricsDroppedRps
        exporterSuccessRps
        exporterFailedRps
        window
        lastScrape
      }
    }
  }
`;

export const GET_NODE_COLLECTOR_INFO = gql`
  query OdigletDaemonSetInfo {
    odigletDaemonSetInfo {
      status
      nodes {
        desired
        ready
      }
      resources {
        requests {
          cpu
          memory
        }
        limits {
          cpu
          memory
        }
      }
      imageVersion
      lastRolloutAt
      rolloutInProgress
      manifestYAML
      configMapYAML
    }
  }
`;

export const GET_NODE_COLLECTOR_PODS = gql`
  query OdigletPods {
    odigletPods {
      namespace
      name
      status
      restartsCount
      nodeName
      creationTimestamp
      image
      collectorMetrics {
        metricsAcceptedRps
        metricsDroppedRps
        exporterSuccessRps
        exporterFailedRps
        window
        lastScrape
      }
    }
  }
`;

export const GET_COLLECTOR_POD_INFO = gql`
  query GetCollectorPodDetails($namespace: String!, $name: String!) {
    collectorPod(namespace: $namespace, name: $name) {
      namespace
      name
      node
      status
      containers {
        name
        image
        status
        stateReason
        ready
        started
        restarts
        startedAt
        resources {
          requests {
            cpu
            memory
          }
          limits {
            cpu
            memory
          }
        }
      }
      manifestYAML
    }
  }
`;
