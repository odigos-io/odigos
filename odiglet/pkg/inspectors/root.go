package inspectors

import (
	"github.com/keyval-dev/odigos/common"
	procdiscovery "github.com/keyval-dev/odigos/procdiscovery/pkg/process"
)

type inspector interface {
	Inspect(process *procdiscovery.Details) (common.ProgrammingLanguage, bool)
}

var inspectorsList = []inspector{java, python, dotNet, nodeJs, golang}

// DetectLanguage returns a list of all the detected languages in the process list
// For go applications the process path is also returned, in all other languages the value is empty
func DetectLanguage(processes []procdiscovery.Details) ([]common.ProgrammingLanguage, string) {
	var result []common.ProgrammingLanguage
	processName := ""
	for _, p := range processes {
		for _, i := range inspectorsList {
			inspectionResult, detected := i.Inspect(&p)
			if detected {
				result = append(result, inspectionResult)
				if inspectionResult == common.GoProgrammingLanguage {
					processName = p.ExeName
				}
				break
			}
		}
	}

	return result, processName
}
