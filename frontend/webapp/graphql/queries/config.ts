import { gql } from '@apollo/client';

export const GET_CONFIG = gql`
  query GetConfig {
    config {
      readonly
      platformType
      tier
      odigosVersion
      installationMethod
      installationStatus
      clusterName
      isCentralProxyRunning
    }
  }
`;

export const GET_EFFECTIVE_CONFIG = gql`
  query GetEffectiveConfig {
    effectiveConfig {
      configVersion
      telemetryEnabled
      openshiftEnabled
      ignoredNamespaces
      ignoredContainers
      ignoreOdigosNamespace
      psp
      imagePrefix
      skipWebhookIssuerCreation
      collectorGateway {
        minReplicas
        maxReplicas
        requestMemoryMiB
        limitMemoryMiB
        requestCPUm
        limitCPUm
        memoryLimiterLimitMiB
        memoryLimiterSpikeLimitMiB
        goMemLimitMiB
        serviceGraphDisabled
        clusterMetricsEnabled
        httpsProxyAddress
        nodeSelector
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
        enableDataCompression
        otlpExporterConfiguration {
          enableDataCompression
          timeout
          retryOnFailure {
            enabled
            initialInterval
            maxInterval
            maxElapsedTime
          }
        }
      }
      profiles
      allowConcurrentAgents
      uiMode
      uiPaginationLimit
      uiRemoteUrl
      centralBackendURL
      clusterName
      mountMethod
      customContainerRuntimeSocketPath
      agentEnvVarsInjectionMethod
      userInstrumentationEnvs {
        languages
      }
      nodeSelector
      karpenterEnabled
      rollout {
        automaticRolloutDisabled
      }
      rollbackDisabled
      rollbackGraceTime
      rollbackStabilityWindow
      oidc {
        tenantUrl
        clientId
        clientSecret
      }
      odigletHealthProbeBindPort
      goAutoOffsetsCron
      goAutoOffsetsMode
      clickhouseJsonTypeEnabled
      checkDeviceHealthBeforeInjection
      resourceSizePreset
      waspEnabled
      metricsSources {
        spanMetrics {
          disabled
          interval
          metricsExpiration
          additionalDimensions
          histogramDisabled
          histogramBuckets
          includedProcessInDimensions
          excludedResourceAttributes
          resourceMetricsKeyAttributes
        }
        hostMetrics {
          disabled
          interval
        }
        kubeletStats {
          disabled
          interval
        }
        odigosOwnMetrics {
          interval
        }
        agentMetrics {
          spanMetrics {
            enabled
          }
          runtimeMetrics {
            java {
              disabled
              metrics {
                name
                disabled
              }
            }
          }
        }
      }
      agentsInitContainerResources {
        requestCPUm
        limitCPUm
        requestMemoryMiB
        limitMemoryMiB
      }
      traceIdSuffix
      allowedTestConnectionHosts
      odigosOwnTelemetryStore {
        metricsStoreDisabled
      }
      imagePullSecrets
      componentLogLevels {
        default
        autoscaler
        scheduler
        instrumentor
        odiglet
        deviceplugin
        ui
        collector
      }
      provenance {
        helmPath
        reconciledFrom
      }
      manifestYAML
    }
  }
`;

export const GET_CONFIG_YAMLS = gql`
  query GetConfigYamls {
    configYamls {
      name
      displayName
      fields {
        displayName
        componentType
        isHelmOnly
        description
        helmValuePath
        docsLink
        componentProps
      }
    }
  }
`;
