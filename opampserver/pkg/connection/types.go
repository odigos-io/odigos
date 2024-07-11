package connection

import (
	"time"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/opampserver/protobufs"
	corev1 "k8s.io/api/core/v1"
)

type ConnectionInfo struct {
	DeviceId            string
	Workload            common.PodWorkload
	Pod                 *corev1.Pod
	Pid                 int64
	InstrumentedAppName string
	lastMessageTime     time.Time
	AgentRemoteConfig   *protobufs.AgentRemoteConfig
}
