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
      allowConcurrentAgents {
        enabled
      }
      uiMode
      uiPaginationLimit
      uiRemoteUrl
      centralBackendURL
      clusterName
      instrumentor {
        mountMethod
        agentEnvVarsInjectionMethod
        checkDeviceHealthBeforeInjection
      }
      customContainerRuntimeSocketPath
      userInstrumentationEnvs {
        languages
      }
      nodeSelector
      karpenter {
        enabled
      }
      rollout {
        automaticRolloutDisabled
      }
      autoRollback {
        disabled
        graceTime
        stabilityWindowTime
      }
      oidc {
        tenantUrl
        clientId
        clientSecret
      }
      odigletHealthProbeBindPort
      goAutoOffsetsCron
      goAutoOffsetsMode
      clickhouseJsonTypeEnabled
      resourceSizePreset
      wasp {
        enabled
      }
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
        isEnterpriseOnly
        description
        helmValuePath
        docsLink
        componentProps
      }
    }
  }
`;
