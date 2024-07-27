package connection

import (
	"time"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/opampserver/pkg/sdkconfig/configresolvers"
	"github.com/odigos-io/odigos/opampserver/protobufs"
	corev1 "k8s.io/api/core/v1"
)

type ConnectionInfo struct {
	DeviceId            string
	Workload            common.PodWorkload
	Pod                 *corev1.Pod
	ContainerName       string
	Pid                 int64
	InstrumentedAppName string
	lastMessageTime     time.Time

	// config related fields
	// AgentRemoteConfig is the full remote config opamp message to send to the agent when needed
	AgentRemoteConfig        *protobufs.AgentRemoteConfig
	RemoteResourceAttributes []configresolvers.ResourceAttribute
}
