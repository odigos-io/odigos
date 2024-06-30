package workload

import (
	"errors"
	"strings"

	"github.com/odigos-io/odigos/common/consts"
	"k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Workload interface {
	client.Object
	AvailableReplicas() int32
}

// compile time check for interface implementation
var _ Workload = &DeploymentWorkload{}
var _ Workload = &DaemonSetWorkload{}
var _ Workload = &StatefulSetWorkload{}

type DeploymentWorkload struct {
	*v1.Deployment
}

func (d *DeploymentWorkload) AvailableReplicas() int32 {
	return d.Status.AvailableReplicas
}

type DaemonSetWorkload struct {
	*v1.DaemonSet
}

func (d *DaemonSetWorkload) AvailableReplicas() int32 {
	return d.Status.NumberReady
}

type StatefulSetWorkload struct {
	*v1.StatefulSet
}

func (s *StatefulSetWorkload) AvailableReplicas() int32 {
	return s.Status.ReadyReplicas
}

func ObjectToWorkload(obj client.Object) (Workload, error) {
	switch t := obj.(type) {
	case *v1.Deployment:
		return &DeploymentWorkload{Deployment: t}, nil
	case *v1.DaemonSet:
		return &DaemonSetWorkload{DaemonSet: t}, nil
	case *v1.StatefulSet:
		return &StatefulSetWorkload{StatefulSet: t}, nil
	default:
		return nil, errors.New("unknown kind")
	}
}


// runtime name is a way to store workload specific CRs with odigos
// and give the k8s object a name which is unique and can be used to extract the workload name and kind
func GetRuntimeObjectName(name string, kind string) string {
	return strings.ToLower(kind + "-" + name)
}

func GetWorkloadInfoRuntimeName(name string) (workloadName string, workloadKind string, err error) {
	hyphenIndex := strings.Index(name, "-")
	if hyphenIndex == -1 {
		err = errors.New("invalid runtime name")
		return
	}

	workloadKind, err = kindFromLowercase(name[:hyphenIndex])
	if err != nil {
		return
	}
	workloadName = name[hyphenIndex+1:]
	return
}

func kindFromLowercase(lowercaseKind string) (string, error) {
	switch lowercaseKind {
	case "deployment":
		return "Deployment", nil
	case "statefulset":
		return "StatefulSet", nil
	case "daemonset":
		return "DaemonSet", nil
	default:
		return "", errors.New("unknown kind")
	}
}

func IsObjectLabeledForInstrumentation(obj client.Object) bool {
	labels := obj.GetLabels()
	if labels == nil {
		return false
	}

	val, exists := labels[consts.OdigosInstrumentationLabel]
	if !exists {
		return false
	}

	return val == consts.InstrumentationEnabled
}
