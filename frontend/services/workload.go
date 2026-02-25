package services

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/executor"
	"github.com/gin-gonic/gin"
	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/frontend/graph/loaders"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const generalQuery = `
	query GetWorkloads($filter: WorkloadFilter) {
		workloads(filter: $filter) {
			id {
				namespace
				kind
				name
			}
			serviceName
			workloadOdigosHealthStatus {
				name
				status
				reasonEnum
				message
			}
			conditions {
				runtimeDetection {
					status
					reasonEnum
					message
				}
				agentInjectionEnabled {
					status
					reasonEnum
					message
				}
				rollout {
					status
					reasonEnum
					message
				}
				agentInjected {
					status
					reasonEnum
					message
				}
				processesAgentHealth {
					status
					reasonEnum
					message
				}
				expectingTelemetry {
					status
					reasonEnum
					message
				}
			}
			markedForInstrumentation {
				markedForInstrumentation
				decisionEnum
				message
			}
			containers {
				containerName
				runtimeInfo {
					language
					runtimeVersion
					otherAgentName
				}
				agentEnabled {
					agentEnabled
					agentEnabledStatus {
						message
						status
						reasonEnum
					}
					otelDistroName
					traces {
						enabled
					}
					metrics {
						enabled
					}
					logs {
						enabled
					}
				}
				overrides {
					runtimeInfo {
						language
						runtimeVersion
					}
				}
				instrumentations {
					name
					isStandardLibrary
				}
			}
			telemetryMetrics {
				totalDataSentBytes
				throughputBytes
				expectingTelemetry {
					isExpectingTelemetry
				}
			}
		}
	}
`

const verboseQuery = `
	query GetWorkloads($filter: WorkloadFilter) {
		workloads(filter: $filter) {
			id {
				namespace
				kind
				name
			}
			serviceName
			workloadOdigosHealthStatus {
				name
				status
				reasonEnum
				message
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
			markedForInstrumentation {
				markedForInstrumentation
				decisionEnum
				message
			}
			runtimeInfo {
				completed
				completedStatus {
					name
					status
					reasonEnum
					message
				}
				detectedLanguages
				containers {
					containerName
					language
					runtimeVersion
					processEnvVars {
						name
						value
					}
					containerRuntimeEnvVars {
						name
						value
					}
					criErrorMessage
					libcType
					secureExecutionMode
					otherAgentName
				}
			}
			agentEnabled {
				agentEnabled
				enabledStatus {
					name
					status
					reasonEnum
					message
				}
				containers {
					containerName
					agentEnabled
					agentEnabledStatus {
						name
						status
						reasonEnum
						message
					}
					otelDistroName
					envInjectionMethod
					distroParams {
						name
						value
					}
					traces {
						enabled
					}
					metrics {
						enabled
					}
					logs {
						enabled
					}
				}
			}
			rollout {
				rolloutStatus {
					name
					status
					reasonEnum
					message
				}
			}
			containers {
				containerName
				runtimeInfo {
					containerName
					language
					runtimeVersion
					processEnvVars {
						name
						value
					}
					containerRuntimeEnvVars {
						name
						value
					}
					criErrorMessage
					libcType
					secureExecutionMode
					otherAgentName
				}
				agentEnabled {
					containerName
					agentEnabled
					agentEnabledStatus {
						name
						status
						reasonEnum
						message
					}
					otelDistroName
					envInjectionMethod
					distroParams {
						name
						value
					}
					traces {
						enabled
					}
					metrics {
						enabled
					}
					logs {
						enabled
					}
				}
				overrides {
					containerName
					runtimeInfo {
						containerName
						language
						runtimeVersion
						processEnvVars {
							name
							value
						}
						containerRuntimeEnvVars {
							name
							value
						}
						criErrorMessage
						libcType
						secureExecutionMode
						otherAgentName
					}
				}
				instrumentations {
					name
					isStandardLibrary
				}
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
			workloadHealthStatus {
				name
				status
				reasonEnum
				message
			}
			processesHealthStatus {
				name
				status
				reasonEnum
				message
			}
			telemetryMetrics {
				totalDataSentBytes
				throughputBytes
				expectingTelemetry {
					isExpectingTelemetry
					telemetryObservedStatus {
						name
						status
						reasonEnum
						message
					}
				}
			}
		}
	}
`

const overviewQuery = `
	query GetWorkloads($filter: WorkloadFilter) {
		workloads(filter: $filter) {
			id {
				namespace
				kind
				name
			}
			serviceName
			telemetryMetrics {
				throughputBytes
			}
			runtimeInfo {
				detectedLanguages
			}
		}
	}
`

const healthSummaryQuery = `
	query GetWorkloads($filter: WorkloadFilter) {
		workloads(filter: $filter) {
			id {
				namespace
				kind
				name
			}
			workloadOdigosHealthStatus {
				name
				status
				reasonEnum
				message
			}
			podsHealthStatus {
				status
				reasonEnum
				message
			}
			workloadHealthStatus {
				status
				reasonEnum
				message
			}
		}
	}
`

const podsQuery = `
	query GetWorkloads($filter: WorkloadFilter) {
		workloads(filter: $filter) {
			id {
				namespace
				kind
				name
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
			workloadHealthStatus {
				name
				status
				reasonEnum
				message
			}
			processesHealthStatus {
				name
				status
				reasonEnum
				message
			}
		}
	}
`

func getQueryForVerbosity(verbosity string) string {
	switch verbosity {
	case "general":
		return generalQuery
	case "overview":
		return overviewQuery
	case "healthSummary":
		return healthSummaryQuery
	case "verbose":
		return verboseQuery
	case "pods":
		return podsQuery
	}
	return generalQuery // default to general
}

func getParamOrQuery(c *gin.Context, param string) string {
	p := c.Param(param)
	if p != "" {
		return p
	}
	q := c.Query(param)
	if q != "" {
		return q
	}
	return ""
}

func senatizeKind(kind string) (string, error) {
	switch strings.ToLower(kind) {
	case "deployment", "deployments", "deploy":
		return string(model.K8sResourceKindDeployment), nil
	case "statefulset", "statefulsets", "sts":
		return string(model.K8sResourceKindStatefulSet), nil
	case "cronjob", "cronjobs", "cj":
		return string(model.K8sResourceKindCronJob), nil
	case "daemonset", "daemonsets", "ds":
		return string(model.K8sResourceKindDaemonSet), nil
	case "deploymentconfig", "deploymentconfigs", "dc":
		return string(model.K8sResourceKindDeploymentConfig), nil
	case "rollout", "rollouts", "ro":
		return string(model.K8sResourceKindRollout), nil
	case "staticpod", "staticpods", "pod", "pods":
		return string(model.K8sResourceKindStaticPod), nil
	case "":
		return "", nil
	default:
		return "", fmt.Errorf("invalid workload kind: %s", kind)
	}
}

func getFilterAndVerbosityFromContext(c *gin.Context) (map[string]interface{}, string, error) {

	namespace := getParamOrQuery(c, "namespace")
	kind, err := senatizeKind(getParamOrQuery(c, "kind"))
	if err != nil {
		return nil, "", err
	}
	name := getParamOrQuery(c, "name")
	markedForInstrumentation := getParamOrQuery(c, "markedForInstrumentation") == "true"
	verbosity := c.Query("verbosity")

	filterMap := map[string]interface{}{}
	if namespace != "" {
		filterMap["namespace"] = namespace
	}
	if name != "" && kind != "" {
		filterMap["name"] = name
		filterMap["kind"] = kind
	}
	if markedForInstrumentation {
		filterMap["markedForInstrumentation"] = true
	}

	return filterMap, verbosity, nil
}

func DescribeWorkloadWithFilters(c *gin.Context, logger logr.Logger, gqlExecutor *executor.Executor, filter map[string]interface{}, verbosity string, k8sCacheClient client.Client) {
	ctx := c.Request.Context()
	// add things to the ctx
	ctx = loaders.WithLoaders(ctx, loaders.NewLoaders(logger, k8sCacheClient))
	ctx = graphql.StartOperationTrace(ctx)

	query := getQueryForVerbosity(verbosity)
	operationContext, errs := gqlExecutor.CreateOperationContext(ctx, &graphql.RawParams{
		Query: query,
		Variables: map[string]interface{}{
			"filter": filter,
		},
	})
	if len(errs) > 0 {
		c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to create GraphQL operation context: %v", errs)})
		return
	}

	responseHandler, ctx := gqlExecutor.DispatchOperation(ctx, operationContext)
	res := responseHandler(ctx)
	if res == nil {
		c.JSON(500, gin.H{"error": "GraphQL execution returned nil response"})
		return
	}

	// Check for errors
	if len(res.Errors) > 0 {
		c.JSON(500, gin.H{"error": fmt.Sprintf("GraphQL errors: %v", res.Errors)})
		return
	}

	// Extract workloads from the response data
	var workloadsData map[string]interface{}
	err := json.Unmarshal(res.Data, &workloadsData)
	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to unmarshal response data: %v", err)})
		return
	}
	workloads, ok := workloadsData["workloads"]
	if !ok {
		c.JSON(500, gin.H{"error": "Workloads not found in response"})
		return
	}

	c.JSON(200, workloads)
}

func DescribeWorkload(c *gin.Context, logger logr.Logger, gqlExecutor *executor.Executor, overrideVerbosity *string, k8sCacheClient client.Client) {
	filter, verbosity, err := getFilterAndVerbosityFromContext(c)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if overrideVerbosity != nil {
		verbosity = *overrideVerbosity
	}

	DescribeWorkloadWithFilters(c, logger, gqlExecutor, filter, verbosity, k8sCacheClient)
}
