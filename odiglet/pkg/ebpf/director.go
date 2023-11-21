package ebpf

import (
	"context"

	"github.com/keyval-dev/odigos/common"
	"k8s.io/apimachinery/pkg/types"
)

type Director interface {
	Language() common.ProgrammingLanguage
	Instrument(ctx context.Context, pid int, podDetails types.NamespacedName, podWorkload common.PodWorkload, appName string) error
	GetPodFromWorkload(podWorkload common.PodWorkload) (types.NamespacedName, bool)
	Cleanup(podDetails types.NamespacedName)
	Shutdown()
}
