package odigoslogsresourceattrsprocessor

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/types"

	"github.com/odigos-io/odigos/collector/processor/odigoslogsresourceattrsprocessor/internal/kube"
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
		logger:            zap.NewNop(),
	}
}

func TestProcessorTestSuite(t *testing.T) {
	suite.Run(t, new(ProcessorTestSuite))
}

func (s *ProcessorTestSuite) TestEnrichResource_SetsAllAttributes() {
	s.mockClient.AddPod(types.UID("test-pod-uid-1"), "my-app", kube.WorkloadKindDeployment, "my-app-xyz123", "production")

	attrs := pcommon.NewMap()

	s.processor.enrichResourceWithPodMetadata(attrs, "test-pod-uid-1")

	// Check service.name
	serviceNameAttr, exists := attrs.Get(string(semconv.ServiceNameKey))
	s.Require().True(exists, "service.name attribute should exist")
	s.Equal("my-app", serviceNameAttr.AsString())

	// Check k8s.pod.name
	podNameAttr, exists := attrs.Get(string(semconv.K8SPodNameKey))
	s.Require().True(exists, "k8s.pod.name attribute should exist")
	s.Equal("my-app-xyz123", podNameAttr.AsString())

	// Check k8s.namespace.name
	nsAttr, exists := attrs.Get(string(semconv.K8SNamespaceNameKey))
	s.Require().True(exists, "k8s.namespace.name attribute should exist")
	s.Equal("production", nsAttr.AsString())

	// Check k8s.deployment.name
	deployAttr, exists := attrs.Get(string(semconv.K8SDeploymentNameKey))
	s.Require().True(exists, "k8s.deployment.name attribute should exist")
	s.Equal("my-app", deployAttr.AsString())

	// Check k8s.pod.uid is set
	podUIDAttr, exists := attrs.Get(string(semconv.K8SPodUIDKey))
	s.Require().True(exists, "k8s.pod.uid attribute should exist")
	s.Equal("test-pod-uid-1", podUIDAttr.AsString())
}

func (s *ProcessorTestSuite) TestEnrichResource_PodNotInCache() {
	attrs := pcommon.NewMap()

	s.processor.enrichResourceWithPodMetadata(attrs, "non-existent-uid")

	_, exists := attrs.Get(string(semconv.ServiceNameKey))
	s.False(exists, "service.name attribute should not exist")
}

func (s *ProcessorTestSuite) TestEnrichResource_ArgoRollout() {
	s.mockClient.AddPod(types.UID("rollout-pod-uid"), "my-rollout", kube.WorkloadKindArgoRollout, "my-rollout-pod-abc", "production")

	attrs := pcommon.NewMap()

	s.processor.enrichResourceWithPodMetadata(attrs, "rollout-pod-uid")

	// Check service.name
	serviceNameAttr, exists := attrs.Get(string(semconv.ServiceNameKey))
	s.Require().True(exists, "service.name attribute should exist")
	s.Equal("my-rollout", serviceNameAttr.AsString())

	// Check k8s.argoproj.rollout.name
	rolloutAttr, exists := attrs.Get(kube.K8SArgoRolloutNameAttribute)
	s.Require().True(exists, "k8s.argoproj.rollout.name attribute should exist")
	s.Equal("my-rollout", rolloutAttr.AsString())
}

func (s *ProcessorTestSuite) TestExtractPodUIDFromFilePath() {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "standard deployment pod path",
			path:     "/var/log/pods/default_myapp-abc123-xyz_a1b2c3d4-e5f6-7890-abcd-ef1234567890/container/0.log",
			expected: "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
		},
		{
			name:     "daemonset pod path",
			path:     "/var/log/pods/kube-system_fluentd-abcde_12345678-1234-1234-1234-123456789012/fluentd/0.log",
			expected: "12345678-1234-1234-1234-123456789012",
		},
		{
			name:     "statefulset pod path",
			path:     "/var/log/pods/database_postgres-0_fedcba98-7654-3210-fedc-ba9876543210/postgres/1.log",
			expected: "fedcba98-7654-3210-fedc-ba9876543210",
		},
		{
			name:     "invalid path - no pods directory",
			path:     "/var/log/containers/myapp.log",
			expected: "",
		},
		{
			name:     "invalid path - no underscore",
			path:     "/var/log/pods/invalid/container/0.log",
			expected: "",
		},
		{
			name:     "empty path",
			path:     "",
			expected: "",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			uid := extractPodUIDFromFilePath(tt.path)
			s.Equal(tt.expected, uid)
		})
	}
}

func (s *ProcessorTestSuite) TestProcessLogs_WithPodUIDInResource() {
	s.mockClient.AddPod(types.UID("pod-uid-1"), "test-service", kube.WorkloadKindDeployment, "test-service-pod-abc", "default")

	logs := plog.NewLogs()
	rl := logs.ResourceLogs().AppendEmpty()
	rl.Resource().Attributes().PutStr(string(semconv.K8SPodUIDKey), "pod-uid-1")
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

func (s *ProcessorTestSuite) TestProcessLogs_ExtractsUIDFromFilePath() {
	s.mockClient.AddPod(types.UID("uid-from-filepath"), "my-service", kube.WorkloadKindDeployment, "my-service-pod", "default")

	logs := plog.NewLogs()
	rl := logs.ResourceLogs().AppendEmpty()
	// No k8s.pod.uid in resource attributes - will extract from file path
	sl := rl.ScopeLogs().AppendEmpty()
	lr := sl.LogRecords().AppendEmpty()
	lr.Body().SetStr("test log")
	lr.Attributes().PutStr("log.file.path", "/var/log/pods/default_my-service-pod_uid-from-filepath/container/0.log")

	resultLogs, err := s.processor.processLogs(context.Background(), logs)

	s.Require().NoError(err)
	s.Equal(1, resultLogs.ResourceLogs().Len())

	// Check that service.name was set
	serviceNameAttr, exists := resultLogs.ResourceLogs().At(0).Resource().Attributes().Get(string(semconv.ServiceNameKey))
	s.Require().True(exists, "service.name should be set when UID is extracted from file path")
	s.Equal("my-service", serviceNameAttr.AsString())

	// Check that k8s.pod.uid was also set on resource
	podUIDAttr, exists := resultLogs.ResourceLogs().At(0).Resource().Attributes().Get(string(semconv.K8SPodUIDKey))
	s.Require().True(exists, "k8s.pod.uid should be set on resource")
	s.Equal("uid-from-filepath", podUIDAttr.AsString())
}

func (s *ProcessorTestSuite) TestProcessLogs_MultipleResources() {
	s.mockClient.AddPod(types.UID("app-a-uid"), "app-a", kube.WorkloadKindDeployment, "app-a-pod", "ns-a")
	s.mockClient.AddPod(types.UID("app-b-uid"), "app-b", kube.WorkloadKindDaemonSet, "app-b-pod", "ns-b")

	logs := plog.NewLogs()

	rl1 := logs.ResourceLogs().AppendEmpty()
	rl1.Resource().Attributes().PutStr(string(semconv.K8SPodUIDKey), "app-a-uid")
	sl1 := rl1.ScopeLogs().AppendEmpty()
	lr1 := sl1.LogRecords().AppendEmpty()
	lr1.Body().SetStr("log from app-a")

	rl2 := logs.ResourceLogs().AppendEmpty()
	rl2.Resource().Attributes().PutStr(string(semconv.K8SPodUIDKey), "app-b-uid")
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

	// Check second resource - DaemonSet
	serviceNameAttr2, exists2 := resultLogs.ResourceLogs().At(1).Resource().Attributes().Get(string(semconv.ServiceNameKey))
	s.Require().True(exists2)
	s.Equal("app-b", serviceNameAttr2.AsString())
}

func (s *ProcessorTestSuite) TestProcessLogs_EmptyLogs() {
	logs := plog.NewLogs()

	resultLogs, err := s.processor.processLogs(context.Background(), logs)

	s.Require().NoError(err)
	s.Equal(0, resultLogs.ResourceLogs().Len())
}

func (s *ProcessorTestSuite) TestProcessLogs_SkipsWhenNoUID() {
	logs := plog.NewLogs()
	rl := logs.ResourceLogs().AppendEmpty()
	// No k8s.pod.uid and no log.file.path
	sl := rl.ScopeLogs().AppendEmpty()
	lr := sl.LogRecords().AppendEmpty()
	lr.Body().SetStr("test log message")

	resultLogs, err := s.processor.processLogs(context.Background(), logs)

	s.Require().NoError(err)
	s.Equal(1, resultLogs.ResourceLogs().Len())

	// service.name should not be set because there was no UID
	_, exists := resultLogs.ResourceLogs().At(0).Resource().Attributes().Get(string(semconv.ServiceNameKey))
	s.False(exists, "service.name should not be set when UID is missing")
}
