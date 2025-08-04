import { gql } from '@apollo/client';

// Define the GraphQL query
export const GET_CONFIG = gql`
  query GetConfig {
    config {
      readonly
      tier
      installationMethod
      installationStatus
    }
  }
`;

export const GET_ODIGOS_CONFIG = gql`
  query GetOdigosConfig {
    odigosConfig {
      karpenterEnabled
      allowConcurrentAgents
      uiPaginationLimit
      centralBackendURL
      oidc {
        tenantUrl
        clientId
        clientSecret
      }
      clusterName
      imagePrefix
      ignoredNamespaces
      ignoredContainers
      mountMethod
      agentEnvVarsInjectionMethod
      customContainerRuntimeSocketPath
      odigletHealthProbeBindPort
      rollbackDisabled
      rollbackGraceTime
      rollbackStabilityWindow
      nodeSelector
      rollout {
        automaticRolloutDisabled
      }
      collectorNode {
        collectorOwnMetricsPort
        requestMemoryMiB
        limitMemoryMiB
        requestCPUm
        limitCPUm
        memoryLimiterLimitMiB
        memoryLimiterSpikeLimitMiB
        goMemLimitMiB
        k8sNodeLogsDirectory
      }
      collectorGateway {
        requestMemoryMiB
        limitMemoryMiB
        requestCPUm
        limitCPUm
        memoryLimiterLimitMiB
        memoryLimiterSpikeLimitMiB
        goMemLimitMiB
        minReplicas
        maxReplicas
      }
    }
  }
`;
