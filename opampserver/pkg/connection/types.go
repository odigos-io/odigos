package connection

import (
	"time"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/opampserver/pkg/sdkconfig/configresolvers"
	"github.com/odigos-io/odigos/opampserver/protobufs"
	corev1 "k8s.io/api/core/v1"
)

type ConnectionInfo struct {
	Workload            k8sconsts.PodWorkload
	Pod                 *corev1.Pod
	ContainerName       string
	Pid                 int64
	InstrumentedAppName string
	LastMessageTime     time.Time
	ProgrammingLanguage string

	// config related fields
	// AgentRemoteConfig is the full remote config opamp message to send to the agent when needed
	AgentRemoteConfig        *protobufs.AgentRemoteConfig
	RemoteResourceAttributes []configresolvers.ResourceAttribute
}
