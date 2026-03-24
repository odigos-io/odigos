import { useEffect } from 'react';
import { useQuery } from '@apollo/client';
import { GET_EFFECTIVE_CONFIG } from '@/graphql';
import { Crud, StatusType } from '@odigos/ui-kit/types';
import { useNotificationStore } from '@odigos/ui-kit/store';

// TODO: once we released the kit with the types, we can remove these interfaces and import `EffectiveConfig` from the kit
interface EffectiveConfig {
  configVersion: number;
  telemetryEnabled?: boolean;
  openshiftEnabled?: boolean;
  ignoredNamespaces?: string[];
  ignoredContainers?: string[];
  ignoreOdigosNamespace?: boolean;
  psp?: boolean;
  imagePrefix?: string;
  skipWebhookIssuerCreation?: boolean;
  collectorGateway?: {
    minReplicas?: number;
    maxReplicas?: number;
    requestMemoryMiB?: number;
    limitMemoryMiB?: number;
    requestCPUm?: number;
    limitCPUm?: number;
    memoryLimiterLimitMiB?: number;
    memoryLimiterSpikeLimitMiB?: number;
    goMemLimitMiB?: number;
    serviceGraphDisabled?: boolean;
    clusterMetricsEnabled?: boolean;
    httpsProxyAddress?: string;
    nodeSelector?: string;
  };
  collectorNode?: {
    collectorOwnMetricsPort?: number;
    requestMemoryMiB?: number;
    limitMemoryMiB?: number;
    requestCPUm?: number;
    limitCPUm?: number;
    memoryLimiterLimitMiB?: number;
    memoryLimiterSpikeLimitMiB?: number;
    goMemLimitMiB?: number;
    k8sNodeLogsDirectory?: string;
    enableDataCompression?: boolean;
    otlpExporterConfiguration?: {
      enableDataCompression?: boolean;
      timeout?: string;
      retryOnFailure?: {
        enabled?: boolean;
        initialInterval?: string;
        maxInterval?: string;
        maxElapsedTime?: string;
      };
    };
  };
  profiles?: string[];
  allowConcurrentAgents?: boolean;
  uiMode?: string;
  uiPaginationLimit?: number;
  uiRemoteUrl?: string;
  centralBackendURL?: string;
  clusterName?: string;
  mountMethod?: string;
  customContainerRuntimeSocketPath?: string;
  agentEnvVarsInjectionMethod?: string;
  userInstrumentationEnvs?: {
    languages?: string;
  };
  nodeSelector?: string;
  karpenterEnabled?: boolean;
  rollout?: {
    automaticRolloutDisabled?: boolean;
  };
  rollbackDisabled?: boolean;
  rollbackGraceTime?: string;
  rollbackStabilityWindow?: string;
  oidc?: {
    tenantUrl?: string;
    clientId?: string;
    clientSecret?: string;
  };
  odigletHealthProbeBindPort?: number;
  goAutoOffsetsCron?: string;
  goAutoOffsetsMode?: string;
  clickhouseJsonTypeEnabled?: boolean;
  checkDeviceHealthBeforeInjection?: boolean;
  resourceSizePreset?: string;
  waspEnabled?: boolean;
  metricsSources?: {
    spanMetrics?: {
      disabled?: boolean;
      interval?: string;
      metricsExpiration?: string;
      additionalDimensions?: string[];
      histogramDisabled?: boolean;
      histogramBuckets?: string[];
      includedProcessInDimensions?: boolean;
      excludedResourceAttributes?: string[];
      resourceMetricsKeyAttributes?: string[];
    };
    hostMetrics?: {
      disabled?: boolean;
      interval?: string;
    };
    kubeletStats?: {
      disabled?: boolean;
      interval?: string;
    };
    odigosOwnMetrics?: {
      interval?: string;
    };
    agentMetrics?: {
      spanMetrics?: {
        enabled: boolean;
      };
      runtimeMetrics?: {
        java?: {
          disabled?: boolean;
          metrics?: {
            name: string;
            disabled?: boolean;
          }[];
        };
      };
    };
  };
  agentsInitContainerResources?: {
    requestCPUm?: number;
    limitCPUm?: number;
    requestMemoryMiB?: number;
    limitMemoryMiB?: number;
  };
  traceIdSuffix?: string;
  allowedTestConnectionHosts?: string[];
  odigosOwnTelemetryStore?: {
    metricsStoreDisabled?: boolean;
  };
  imagePullSecrets?: string[];
  componentLogLevels?: {
    default?: string;
    autoscaler?: string;
    scheduler?: string;
    instrumentor?: string;
    odiglet?: string;
    deviceplugin?: string;
    ui?: string;
    collector?: string;
  };
  manifestYAML?: string;
}

interface FetchedEffectiveConfig {
  effectiveConfig?: EffectiveConfig;
}

export const useEffectiveConfig = () => {
  const { data, loading, error } = useQuery<FetchedEffectiveConfig>(GET_EFFECTIVE_CONFIG);
  const { addNotification } = useNotificationStore();

  useEffect(() => {
    if (error) {
      addNotification({
        type: StatusType.Error,
        title: error.name || Crud.Read,
        message: error.cause?.message || error.message,
      });
    }
  }, [error]);

  return {
    effectiveConfig: data?.effectiveConfig || null,
    effectiveConfigLoading: loading,
  };
};
