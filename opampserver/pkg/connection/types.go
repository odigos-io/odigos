package connection

import (
	"time"

	corev1 "k8s.io/api/core/v1"
)

type ConnectionInfo struct {
	DeviceId            string
	Pod                 *corev1.Pod
	Pid                 int64
	InstrumentedAppName string
	lastMessageTime     time.Time
}
