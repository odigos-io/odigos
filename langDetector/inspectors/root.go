package inspectors

import (
	v1 "github.com/keyval-dev/odigos/langDetector/kube/apis/v1"
	"github.com/keyval-dev/odigos/langDetector/process"
)

type inspector interface {
	Inspect(process *process.Details) (v1.ProgrammingLanguage, bool)
}

var inspectorsList = []inspector{java, python, dotNet, nodeJs, golang}

// DetectLanguage returns a list of all the detected languages in the process list
// For go applications the process path is also returned, in all other languages the value is empty
func DetectLanguage(processes []*process.Details) ([]v1.ProgrammingLanguage, string) {
	var result []v1.ProgrammingLanguage
	processName := ""
	for _, p := range processes {
		for _, i := range inspectorsList {
			inspectionResult, detected := i.Inspect(p)
			if detected {
				result = append(result, inspectionResult)
				if inspectionResult == v1.GoProgrammingLanguage {
					processName = p.ExeName
				}
				break
			}
		}
	}

	return result, processName
}
