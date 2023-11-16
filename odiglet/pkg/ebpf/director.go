package ebpf

import (
	"github.com/keyval-dev/odigos/common"
	"k8s.io/apimachinery/pkg/types"
)

type Director interface {
	Language() common.ProgrammingLanguage
	Instrument(pid int, podDetails types.NamespacedName, appName string) error
	Cleanup(podDetails types.NamespacedName)
	Shutdown()
}
