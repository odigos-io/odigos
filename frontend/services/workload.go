package services

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/executor"
	"github.com/gin-gonic/gin"
	"github.com/odigos-io/odigos/frontend/graph/loaders"
	"github.com/odigos-io/odigos/frontend/graph/model"
)

const overviewQuery = `
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
			}
		}
	}
`

const markedForInstrumentationQuery = `
	query GetWorkloads($filter: WorkloadFilter) {
		workloads(filter: $filter) {
			id {
				namespace
				kind
				name
			}
			markedForInstrumentation {
				markedForInstrumentation
			}
		}
	}
`

func getQueryForVerbosity(verbosity string) string {
	switch verbosity {
	case "overview":
		return overviewQuery
	case "markedForInstrumentation":
		return markedForInstrumentationQuery
	}
	return overviewQuery // default to overview
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
	case "":
		return "", nil
	default:
		return "", fmt.Errorf("invalid workload kind: %s", kind)
	}
}

func DescribeWorkload(c *gin.Context, gqlExecutor *executor.Executor) {
	ctx := c.Request.Context()

	// get relevant filters from query params
	namespace := getParamOrQuery(c, "namespace")
	kind, err := senatizeKind(getParamOrQuery(c, "kind"))
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	name := getParamOrQuery(c, "name")
	markedForInstrumentation := getParamOrQuery(c, "markedForInstrumentation")
	verbosity := c.Query("verbosity")

	// Create workload filter based on query parameters
	filterMap := map[string]interface{}{} // how gql expects it
	if namespace != "" || kind != "" || name != "" || markedForInstrumentation != "" {

		if namespace != "" {
			filterMap["namespace"] = namespace
		}
		if kind != "" {
			filterMap["kind"] = kind
		}
		if name != "" {
			filterMap["name"] = name
		}
		if markedForInstrumentation == "true" {
			filterMap["markedForInstrumentation"] = true
		} else if markedForInstrumentation == "false" {
			filterMap["markedForInstrumentation"] = false
		}
	}

	// Create a new context with loaders
	ctx = loaders.WithLoaders(ctx, loaders.NewLoaders())
	ctx = graphql.StartOperationTrace(ctx)

	query := getQueryForVerbosity(verbosity)
	operationContext, errs := gqlExecutor.CreateOperationContext(ctx, &graphql.RawParams{
		Query: query,
		Variables: map[string]interface{}{
			"filter": filterMap,
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
	err = json.Unmarshal(res.Data, &workloadsData)
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
