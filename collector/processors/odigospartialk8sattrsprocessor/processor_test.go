package odigospartialk8sattrsprocessor

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"k8s.io/apimachinery/pkg/types"

	"github.com/odigos-io/odigos/collector/processor/odigospartialk8sattrsprocessor/internal/kube"
)

// ProcessorTestSuite tests the partialK8sAttrsProcessor
type ProcessorTestSuite struct {
	suite.Suite
	mockClient *mockKubeClient
	processor  *partialK8sAttrsProcessor
}

func (s *ProcessorTestSuite) SetupTest() {
	s.mockClient = newMockKubeClient()
	s.processor = &partialK8sAttrsProcessor{
		podMetadataClient: s.mockClient,
	}
}

func TestProcessorTestSuite(t *testing.T) {
	suite.Run(t, new(ProcessorTestSuite))
}

func (s *ProcessorTestSuite) TestProcessResource_SetsAllAttributes() {
	// Workload is Deployment "my-app"
	s.mockClient.AddPod(types.UID("test-pod-uid-1"), "my-app", kube.WorkloadKindDeployment, "my-app-xyz123", "production")

	resource := pcommon.NewResource()
	resource.Attributes().PutStr("k8s.pod.uid", "test-pod-uid-1")

	s.processor.processResource(resource)

	// Check service.name
	serviceNameAttr, exists := resource.Attributes().Get(string(semconv.ServiceNameKey))
	s.Require().True(exists, "service.name attribute should exist")
	s.Equal("my-app", serviceNameAttr.AsString())

	// Check service.namespace
	serviceNsAttr, exists := resource.Attributes().Get(string(semconv.ServiceNamespaceKey))
	s.Require().True(exists, "service.namespace attribute should exist")
	s.Equal("production", serviceNsAttr.AsString())

	// Check k8s.pod.name
	podNameAttr, exists := resource.Attributes().Get(string(semconv.K8SPodNameKey))
	s.Require().True(exists, "k8s.pod.name attribute should exist")
	s.Equal("my-app-xyz123", podNameAttr.AsString())

	// Check k8s.namespace.name
	nsAttr, exists := resource.Attributes().Get(string(semconv.K8SNamespaceNameKey))
	s.Require().True(exists, "k8s.namespace.name attribute should exist")
	s.Equal("production", nsAttr.AsString())

	// Check k8s.deployment.name (semconv for Deployment workload)
	deployAttr, exists := resource.Attributes().Get(string(semconv.K8SDeploymentNameKey))
	s.Require().True(exists, "k8s.deployment.name attribute should exist")
	s.Equal("my-app", deployAttr.AsString())
}

func (s *ProcessorTestSuite) TestProcessResource_PodNotInCache() {
	resource := pcommon.NewResource()
	resource.Attributes().PutStr("k8s.pod.uid", "non-existent-pod-uid")

	s.processor.processResource(resource)

	_, exists := resource.Attributes().Get(string(semconv.ServiceNameKey))
	s.False(exists, "service.name attribute should not exist")
}

func (s *ProcessorTestSuite) TestProcessResource_PodWithoutWorkload() {
	// Pod without workload info (e.g., standalone pod)
	s.mockClient.AddPod(types.UID("standalone-pod-uid"), "", "", "standalone-pod", "default")

	resource := pcommon.NewResource()
	resource.Attributes().PutStr("k8s.pod.uid", "standalone-pod-uid")

	s.processor.processResource(resource)

	// service.name should not exist when workload info is empty
	_, exists := resource.Attributes().Get(string(semconv.ServiceNameKey))
	s.False(exists, "service.name attribute should not exist")

	// But pod name and namespace should still be set
	podNameAttr, exists := resource.Attributes().Get(string(semconv.K8SPodNameKey))
	s.Require().True(exists, "k8s.pod.name should be set even without workload")
	s.Equal("standalone-pod", podNameAttr.AsString())

	nsAttr, exists := resource.Attributes().Get(string(semconv.K8SNamespaceNameKey))
	s.Require().True(exists, "k8s.namespace.name should be set even without workload")
	s.Equal("default", nsAttr.AsString())
}

func (s *ProcessorTestSuite) TestProcessResource_ComplexWorkloadName() {
	s.mockClient.AddPod(types.UID("frontend-api-pod-uid"), "frontend-api-v2", kube.WorkloadKindDeployment, "frontend-api-v2-abc123-xyz", "staging")

	resource := pcommon.NewResource()
	resource.Attributes().PutStr("k8s.pod.uid", "frontend-api-pod-uid")

	s.processor.processResource(resource)

	serviceNameAttr, exists := resource.Attributes().Get(string(semconv.ServiceNameKey))
	s.Require().True(exists, "service.name attribute should exist")
	s.Equal("frontend-api-v2", serviceNameAttr.AsString())
}

func (s *ProcessorTestSuite) TestProcessResource_NoPodUID() {
	resource := pcommon.NewResource()
	resource.Attributes().PutStr("some.other.attr", "value")

	s.processor.processResource(resource)

	_, exists := resource.Attributes().Get(string(semconv.ServiceNameKey))
	s.False(exists, "service.name should not be set when k8s.pod.uid is missing")
}

func (s *ProcessorTestSuite) TestProcessLogs_SingleResource() {
	s.mockClient.AddPod(types.UID("pod-uid-1"), "test-service", kube.WorkloadKindDeployment, "test-service-pod-abc", "default")

	logs := plog.NewLogs()
	rl := logs.ResourceLogs().AppendEmpty()
	rl.Resource().Attributes().PutStr("k8s.pod.uid", "pod-uid-1")
	sl := rl.ScopeLogs().AppendEmpty()
	lr := sl.LogRecords().AppendEmpty()
	lr.Body().SetStr("test log message")
	lr.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))

	resultLogs, err := s.processor.processLogs(context.Background(), logs)

	s.Require().NoError(err)
	s.Equal(1, resultLogs.ResourceLogs().Len())

	serviceNameAttr, exists := resultLogs.ResourceLogs().At(0).Resource().Attributes().Get(string(semconv.ServiceNameKey))
	s.Require().True(exists)
	s.Equal("test-service", serviceNameAttr.AsString())
}

func (s *ProcessorTestSuite) TestProcessLogs_MultipleResources() {
	s.mockClient.AddPod(types.UID("app-a-uid"), "app-a", kube.WorkloadKindDeployment, "app-a-pod", "ns-a")
	s.mockClient.AddPod(types.UID("app-b-uid"), "app-b", kube.WorkloadKindDaemonSet, "app-b-pod", "ns-b")

	logs := plog.NewLogs()

	rl1 := logs.ResourceLogs().AppendEmpty()
	rl1.Resource().Attributes().PutStr("k8s.pod.uid", "app-a-uid")
	sl1 := rl1.ScopeLogs().AppendEmpty()
	lr1 := sl1.LogRecords().AppendEmpty()
	lr1.Body().SetStr("log from app-a")

	rl2 := logs.ResourceLogs().AppendEmpty()
	rl2.Resource().Attributes().PutStr("k8s.pod.uid", "app-b-uid")
	sl2 := rl2.ScopeLogs().AppendEmpty()
	lr2 := sl2.LogRecords().AppendEmpty()
	lr2.Body().SetStr("log from app-b")

	resultLogs, err := s.processor.processLogs(context.Background(), logs)

	s.Require().NoError(err)
	s.Equal(2, resultLogs.ResourceLogs().Len())

	// Check first resource - Deployment
	serviceNameAttr1, exists1 := resultLogs.ResourceLogs().At(0).Resource().Attributes().Get(string(semconv.ServiceNameKey))
	s.Require().True(exists1)
	s.Equal("app-a", serviceNameAttr1.AsString())

	deployAttr1, exists1 := resultLogs.ResourceLogs().At(0).Resource().Attributes().Get(string(semconv.K8SDeploymentNameKey))
	s.Require().True(exists1)
	s.Equal("app-a", deployAttr1.AsString())

	// Check second resource - DaemonSet
	serviceNameAttr2, exists2 := resultLogs.ResourceLogs().At(1).Resource().Attributes().Get(string(semconv.ServiceNameKey))
	s.Require().True(exists2)
	s.Equal("app-b", serviceNameAttr2.AsString())

	dsAttr2, exists2 := resultLogs.ResourceLogs().At(1).Resource().Attributes().Get(string(semconv.K8SDaemonSetNameKey))
	s.Require().True(exists2)
	s.Equal("app-b", dsAttr2.AsString())
}

func (s *ProcessorTestSuite) TestProcessLogs_EmptyLogs() {
	logs := plog.NewLogs()

	resultLogs, err := s.processor.processLogs(context.Background(), logs)

	s.Require().NoError(err)
	s.Equal(0, resultLogs.ResourceLogs().Len())
}

func (s *ProcessorTestSuite) TestProcessResource_ArgoRollout() {
	s.mockClient.AddPod(types.UID("rollout-pod-uid"), "my-rollout", kube.WorkloadKindArgoRollout, "my-rollout-pod-abc", "production")

	resource := pcommon.NewResource()
	resource.Attributes().PutStr("k8s.pod.uid", "rollout-pod-uid")

	s.processor.processResource(resource)

	// Check service.name
	serviceNameAttr, exists := resource.Attributes().Get(string(semconv.ServiceNameKey))
	s.Require().True(exists, "service.name attribute should exist")
	s.Equal("my-rollout", serviceNameAttr.AsString())

	// Check k8s.argoproj.rollout.name (custom attribute for Argo Rollout)
	rolloutAttr, exists := resource.Attributes().Get(kube.K8SArgoRolloutNameAttribute)
	s.Require().True(exists, "k8s.argoproj.rollout.name attribute should exist")
	s.Equal("my-rollout", rolloutAttr.AsString())
}
