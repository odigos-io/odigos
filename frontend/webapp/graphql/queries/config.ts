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
      telemetryEnabled {
        reconciledFrom
        value
      }
      openshiftEnabled {
        reconciledFrom
        value
      }
      ignoredNamespaces {
        reconciledFrom
        value
      }
      ignoredContainers {
        reconciledFrom
        value
      }
      ignoreOdigosNamespace {
        reconciledFrom
        value
      }
      psp {
        reconciledFrom
        value
      }
      imagePrefix {
        reconciledFrom
        value
      }
      skipWebhookIssuerCreation {
        reconciledFrom
        value
      }
      collectorGateway {
        minReplicas {
          reconciledFrom
          value
        }
        maxReplicas {
          reconciledFrom
          value
        }
        requestMemoryMiB {
          reconciledFrom
          value
        }
        limitMemoryMiB {
          reconciledFrom
          value
        }
        requestCPUm {
          reconciledFrom
          value
        }
        limitCPUm {
          reconciledFrom
          value
        }
        memoryLimiterLimitMiB {
          reconciledFrom
          value
        }
        memoryLimiterSpikeLimitMiB {
          reconciledFrom
          value
        }
        goMemLimitMiB {
          reconciledFrom
          value
        }
        serviceGraphDisabled {
          reconciledFrom
          value
        }
        clusterMetricsEnabled {
          reconciledFrom
          value
        }
        httpsProxyAddress {
          reconciledFrom
          value
        }
        nodeSelector {
          reconciledFrom
          value
        }
      }
      collectorNode {
        collectorOwnMetricsPort {
          reconciledFrom
          value
        }
        requestMemoryMiB {
          reconciledFrom
          value
        }
        limitMemoryMiB {
          reconciledFrom
          value
        }
        requestCPUm {
          reconciledFrom
          value
        }
        limitCPUm {
          reconciledFrom
          value
        }
        memoryLimiterLimitMiB {
          reconciledFrom
          value
        }
        memoryLimiterSpikeLimitMiB {
          reconciledFrom
          value
        }
        goMemLimitMiB {
          reconciledFrom
          value
        }
        k8sNodeLogsDirectory {
          reconciledFrom
          value
        }
        enableDataCompression {
          reconciledFrom
          value
        }
        otlpExporterConfiguration {
          enableDataCompression {
            reconciledFrom
            value
          }
          timeout {
            reconciledFrom
            value
          }
          retryOnFailure {
            enabled {
              reconciledFrom
              value
            }
            initialInterval {
              reconciledFrom
              value
            }
            maxInterval {
              reconciledFrom
              value
            }
            maxElapsedTime {
              reconciledFrom
              value
            }
          }
        }
      }
      profiles {
        reconciledFrom
        value
      }
      allowConcurrentAgents {
        reconciledFrom
        value
      }
      uiMode {
        reconciledFrom
        value
      }
      uiPaginationLimit {
        reconciledFrom
        value
      }
      uiRemoteUrl {
        reconciledFrom
        value
      }
      centralBackendURL {
        reconciledFrom
        value
      }
      clusterName {
        reconciledFrom
        value
      }
      mountMethod {
        reconciledFrom
        value
      }
      customContainerRuntimeSocketPath {
        reconciledFrom
        value
      }
      agentEnvVarsInjectionMethod {
        reconciledFrom
        value
      }
      userInstrumentationEnvs {
        languages {
          reconciledFrom
          value
        }
      }
      nodeSelector {
        reconciledFrom
        value
      }
      karpenterEnabled {
        reconciledFrom
        value
      }
      rollout {
        automaticRolloutDisabled {
          reconciledFrom
          value
        }
      }
      rollbackDisabled {
        reconciledFrom
        value
      }
      rollbackGraceTime {
        reconciledFrom
        value
      }
      rollbackStabilityWindow {
        reconciledFrom
        value
      }
      oidc {
        tenantUrl {
          reconciledFrom
          value
        }
        clientId {
          reconciledFrom
          value
        }
        clientSecret {
          reconciledFrom
          value
        }
      }
      odigletHealthProbeBindPort {
        reconciledFrom
        value
      }
      goAutoOffsetsCron {
        reconciledFrom
        value
      }
      goAutoOffsetsMode {
        reconciledFrom
        value
      }
      clickhouseJsonTypeEnabled {
        reconciledFrom
        value
      }
      checkDeviceHealthBeforeInjection {
        reconciledFrom
        value
      }
      resourceSizePreset {
        reconciledFrom
        value
      }
      waspEnabled {
        reconciledFrom
        value
      }
      metricsSources {
        spanMetrics {
          disabled {
            reconciledFrom
            value
          }
          interval {
            reconciledFrom
            value
          }
          metricsExpiration {
            reconciledFrom
            value
          }
          additionalDimensions {
            reconciledFrom
            value
          }
          histogramDisabled {
            reconciledFrom
            value
          }
          histogramBuckets {
            reconciledFrom
            value
          }
          includedProcessInDimensions {
            reconciledFrom
            value
          }
          excludedResourceAttributes {
            reconciledFrom
            value
          }
          resourceMetricsKeyAttributes {
            reconciledFrom
            value
          }
        }
        hostMetrics {
          disabled {
            reconciledFrom
            value
          }
          interval {
            reconciledFrom
            value
          }
        }
        kubeletStats {
          disabled {
            reconciledFrom
            value
          }
          interval {
            reconciledFrom
            value
          }
        }
        odigosOwnMetrics {
          interval {
            reconciledFrom
            value
          }
        }
        agentMetrics {
          spanMetrics {
            enabled {
              reconciledFrom
              value
            }
          }
          runtimeMetrics {
            java {
              disabled {
                reconciledFrom
                value
              }
              metrics {
                name {
                  reconciledFrom
                  value
                }
                disabled {
                  reconciledFrom
                  value
                }
              }
            }
          }
        }
      }
      agentsInitContainerResources {
        requestCPUm {
          reconciledFrom
          value
        }
        limitCPUm {
          reconciledFrom
          value
        }
        requestMemoryMiB {
          reconciledFrom
          value
        }
        limitMemoryMiB {
          reconciledFrom
          value
        }
      }
      traceIdSuffix {
        reconciledFrom
        value
      }
      allowedTestConnectionHosts {
        reconciledFrom
        value
      }
      odigosOwnTelemetryStore {
        metricsStoreDisabled {
          reconciledFrom
          value
        }
      }
      imagePullSecrets {
        reconciledFrom
        value
      }
      componentLogLevels {
        default {
          reconciledFrom
          value
        }
        autoscaler {
          reconciledFrom
          value
        }
        scheduler {
          reconciledFrom
          value
        }
        instrumentor {
          reconciledFrom
          value
        }
        odiglet {
          reconciledFrom
          value
        }
        deviceplugin {
          reconciledFrom
          value
        }
        ui {
          reconciledFrom
          value
        }
        collector {
          reconciledFrom
          value
        }
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
