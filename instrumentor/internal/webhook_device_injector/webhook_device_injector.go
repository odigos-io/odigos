package webhookdeviceinjector

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func InjectOdigosInstrumentationDevice(ctx context.Context, p client.Client, logger logr.Logger, podWorkload workload.PodWorkload, container *corev1.Container,
	pl common.ProgrammingLanguage, otelsdk common.OtelSdk) {

}
