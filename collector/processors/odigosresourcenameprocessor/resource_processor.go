// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package odigosresourcenameprocessor // import "github.com/open-telemetry/opentelemetry-collector-contrib/processor/odigosresourcenameprocessor"

import (
	"context"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.uber.org/zap"
)

const (
	odigosDeviceKey = "odigos.device"
)

type resourceProcessor struct {
	logger       *zap.Logger
	nameResolver *NameResolver
}

func (rp *resourceProcessor) processAttributes(ctx context.Context, logger *zap.Logger, attrs pcommon.Map) {
	// Get resource name
	_, replaceName := attrs.Get(odigosDeviceKey)
	if replaceName {
		resourceName, ok := attrs.Get(string(semconv.ServiceNameKey))
		if !ok {
			logger.Info("No resource name found, skipping")
			return
		}

		resourceAttributes, err := rp.nameResolver.Resolve(resourceName.AsString())
		if err != nil {
			logger.Error("Could not resolve pod name", zap.Error(err))
			return
		}

		// Replace service name
		resourceName.SetStr(resourceAttributes.OtelServiceName)

		// add k8s resource attributes
		if resourceAttributes.Namespace != "" {
			attrs.PutStr(string(semconv.K8SNamespaceNameKey), resourceAttributes.Namespace)
		}
		if resourceAttributes.WorkloadName != "" {
			switch resourceAttributes.WorkloadKind {
			case "Deployment":
				attrs.PutStr(string(semconv.K8SDeploymentNameKey), resourceAttributes.WorkloadName)
			case "StatefulSet":
				attrs.PutStr(string(semconv.K8SStatefulSetNameKey), resourceAttributes.WorkloadName)
			case "DaemonSet":
				attrs.PutStr(string(semconv.K8SDaemonSetNameKey), resourceAttributes.WorkloadName)
			}
		}
		if resourceAttributes.PodName != "" {
			attrs.PutStr(string(semconv.K8SPodNameKey), resourceAttributes.PodName)
		}
		if resourceAttributes.ContainerName != "" {
			attrs.PutStr(string(semconv.ContainerNameKey), resourceAttributes.ContainerName)
		}
		return
	}
}

func (rp *resourceProcessor) processTraces(ctx context.Context, td ptrace.Traces) (ptrace.Traces, error) {
	if rp.nameResolver == nil {
		return td, nil
	}

	rss := td.ResourceSpans()
	for i := 0; i < rss.Len(); i++ {
		rp.processAttributes(ctx, rp.logger, rss.At(i).Resource().Attributes())
	}
	return td, nil
}

func (rp *resourceProcessor) processMetrics(ctx context.Context, md pmetric.Metrics) (pmetric.Metrics, error) {
	if rp.nameResolver == nil {
		return md, nil
	}

	rms := md.ResourceMetrics()
	for i := 0; i < rms.Len(); i++ {
		rp.processAttributes(ctx, rp.logger, rms.At(i).Resource().Attributes())
	}
	return md, nil
}

func (rp *resourceProcessor) processLogs(ctx context.Context, ld plog.Logs) (plog.Logs, error) {
	if rp.nameResolver == nil {
		return ld, nil
	}

	rls := ld.ResourceLogs()
	for i := 0; i < rls.Len(); i++ {
		rp.processAttributes(ctx, rp.logger, rls.At(i).Resource().Attributes())
	}
	return ld, nil
}

func (rp *resourceProcessor) Shutdown(ctx context.Context) error {
	if rp.nameResolver != nil {
		rp.nameResolver.Shutdown()
	}
	return nil
}
