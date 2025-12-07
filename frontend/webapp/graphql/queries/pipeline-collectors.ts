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
    }
  }
`;

export const GET_GATEWAY_PODS = gql`
  query GatewayPods {
    gatewayPods {
      namespace
      name
      ready
      started
      status
      restartsCount
      nodeName
      creationTimestamp
      image
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
    }
  }
`;

export const GET_NODE_COLLECTOR_PODS = gql`
  query OdigletPods {
    odigletPods {
      namespace
      name
      ready
      started
      status
      restartsCount
      nodeName
      creationTimestamp
      image
    }
  }
`;

export const GET_POD_INFO = gql`
  query GetPodDetails($namespace: String!, $name: String!) {
    pod(namespace: $namespace, name: $name) {
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
