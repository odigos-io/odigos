package workload

import (
	"errors"
	"strings"

	"github.com/odigos-io/odigos/common/consts"
	"k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Workload interface {
	Name() string
	Namespace() string
	Kind() string
	AvailableReplicas() int32
}

// compile time check for interface implementation
var _ Workload = &DeploymentWorkload{}
var _ Workload = &DaemonSetWorkload{}
var _ Workload = &StatefulSetWorkload{}

type DeploymentWorkload struct {
	Dep *v1.Deployment
}

func (d *DeploymentWorkload) Name() string {
	return d.Dep.Name
}

func (d *DeploymentWorkload) Namespace() string {
	return d.Dep.Namespace
}

func (d *DeploymentWorkload) Kind() string {
	return "Deployment"
}

func (d *DeploymentWorkload) AvailableReplicas() int32 {
	return d.Dep.Status.AvailableReplicas
}

type DaemonSetWorkload struct {
	Ds *v1.DaemonSet
}

func (d *DaemonSetWorkload) Name() string {
	return d.Ds.Name
}

func (d *DaemonSetWorkload) Namespace() string {
	return d.Ds.Namespace
}

func (d *DaemonSetWorkload) Kind() string {
	return "DaemonSet"
}

func (d *DaemonSetWorkload) AvailableReplicas() int32 {
	return d.Ds.Status.NumberReady
}

type StatefulSetWorkload struct {
	Ss *v1.StatefulSet
}

func (s *StatefulSetWorkload) Name() string {
	return s.Ss.Name
}

func (s *StatefulSetWorkload) Namespace() string {
	return s.Ss.Namespace
}

func (s *StatefulSetWorkload) Kind() string {
	return "StatefulSet"
}

func (s *StatefulSetWorkload) AvailableReplicas() int32 {
	return s.Ss.Status.ReadyReplicas
}

func ObjectToWorkload(obj any) (Workload, error) {
	switch t := obj.(type) {
	case *v1.Deployment:
		return &DeploymentWorkload{Dep: t}, nil
	case *v1.DaemonSet:
		return &DaemonSetWorkload{Ds: t}, nil
	case *v1.StatefulSet:
		return &StatefulSetWorkload{Ss: t}, nil
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
